package reservas

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"carwashinmaribackend/internal/modules/documentos"
	"carwashinmaribackend/internal/utils"
)

const (
	// anticipacionMin es el margen mínimo entre "ahora" y el inicio de una reserva.
	anticipacionMin = 30
	// diasMaxAnticipacion es cuánto a futuro se puede reservar.
	diasMaxAnticipacion = 90
	maxServiciosReserva = 10
	maxTrabajadores     = 5
)

type Service interface {
	Disponibilidad(ctx context.Context, fecha string, servicios []uuid.UUID, excluir *uuid.UUID) (*Disponibilidad, error)
	Crear(ctx context.Context, idCliente uuid.UUID, in CrearReservaInput) (*Reserva, error)
	MisReservas(ctx context.Context, idCliente uuid.UUID) ([]Reserva, error)
	// Obtener devuelve la reserva; si idCliente no es nil, debe pertenecerle.
	Obtener(ctx context.Context, id uuid.UUID, idCliente *uuid.UUID) (*Reserva, error)
	Reprogramar(ctx context.Context, idCliente uuid.UUID, id uuid.UUID, in ReprogramarInput) (*Reserva, error)
	// Cancelar: si idCliente no es nil, la reserva debe pertenecerle (cancela el cliente); si es nil cancela el admin.
	Cancelar(ctx context.Context, idCliente *uuid.UUID, id uuid.UUID, motivo string) (*Reserva, error)

	ListarTodas(ctx context.Context, sede uuid.UUID, estado, fecha string) ([]Reserva, error)
	Agenda(ctx context.Context, fecha string) (*AgendaDia, error)
	TrabajadoresDisponibles(ctx context.Context, id uuid.UUID) ([]TrabajadorDisponible, error)
	Programar(ctx context.Context, id, idAdmin uuid.UUID, in ProgramarInput) (*Reserva, error)
}

type service struct {
	repo  Repository
	docs  documentos.Checker
	ahora func() time.Time
}

func NewService(repo Repository, docs documentos.Checker) Service {
	return &service{repo: repo, docs: docs, ahora: time.Now}
}

// ---------------------------------------------------------------- utilidades

func minutosDelDia(t time.Time) int { return t.Hour()*60 + t.Minute() }

func inicioDelDia(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// validarFecha comprueba que la fecha sea de hoy en adelante y dentro del límite de anticipación.
func (s *service) validarFecha(fecha string) (time.Time, error) {
	f, err := utils.ParseFecha(fecha)
	if err != nil {
		return time.Time{}, utils.BadRequest(err.Error())
	}
	hoy := inicioDelDia(s.ahora())
	if f.Before(hoy) {
		return time.Time{}, utils.BadRequest("no se puede reservar en una fecha pasada")
	}
	if f.After(hoy.AddDate(0, 0, diasMaxAnticipacion)) {
		return time.Time{}, utils.BadRequest("solo se puede reservar con hasta 90 días de anticipación")
	}
	return f, nil
}

// minInicioPermitido devuelve el minuto del día desde el cual se puede reservar en esa fecha.
func (s *service) minInicioPermitido(f time.Time) int {
	ahora := s.ahora()
	if inicioDelDia(ahora).Equal(f) {
		return minutosDelDia(ahora) + anticipacionMin
	}
	return 0
}

func dedup(ids []uuid.UUID) []uuid.UUID {
	visto := map[uuid.UUID]bool{}
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if !visto[id] {
			visto[id] = true
			out = append(out, id)
		}
	}
	return out
}

// cargarServicios valida que todos los servicios existan y estén disponibles.
func (s *service) cargarServicios(ctx context.Context, ids []uuid.UUID) ([]ServicioInfo, error) {
	ids = dedup(ids)
	if len(ids) == 0 {
		return nil, utils.BadRequest("selecciona al menos un servicio")
	}
	if len(ids) > maxServiciosReserva {
		return nil, utils.BadRequest("una reserva admite hasta 10 servicios")
	}
	svcs, err := s.repo.ServiciosPorID(ctx, ids)
	if err != nil {
		return nil, err
	}
	if len(svcs) != len(ids) {
		return nil, utils.BadRequest("alguno de los servicios seleccionados no existe")
	}
	for _, sv := range svcs {
		if !sv.Disponible {
			return nil, utils.BadRequest("el servicio \"" + sv.Nombre + "\" no está disponible")
		}
	}
	return svcs, nil
}

func sumaDuracion(svcs []ServicioInfo) int {
	total := 0
	for _, sv := range svcs {
		total += sv.DuracionMin
	}
	return total
}

// validarHueco comprueba fecha/hora futura, espacio activo y que el rango quepa en su horario.
func (s *service) validarHueco(ctx context.Context, idEspacio uuid.UUID, fecha, horaInicio string, duracion int) (ini, fin int, err error) {
	f, err := s.validarFecha(fecha)
	if err != nil {
		return 0, 0, err
	}
	ini, perr := utils.ParseHM(horaInicio)
	if perr != nil {
		return 0, 0, utils.BadRequest(perr.Error())
	}
	fin = ini + duracion
	if fin > 24*60 {
		return 0, 0, utils.BadRequest("la reserva termina después de medianoche")
	}
	if ini < s.minInicioPermitido(f) {
		return 0, 0, utils.BadRequest("elige una hora con al menos 30 minutos de anticipación")
	}

	activo, err := s.repo.EspacioActivo(ctx, idEspacio)
	if err != nil {
		return 0, 0, err
	}
	if !activo {
		return 0, 0, utils.BadRequest("el espacio de lavado no existe o está inactivo")
	}

	tramos, err := s.repo.TramosEspacioDia(ctx, idEspacio, int(f.Weekday()))
	if err != nil {
		return 0, 0, err
	}
	if !dentroDeTramos(Rango{Inicio: ini, Fin: fin}, tramos) {
		return 0, 0, utils.BadRequest("el horario elegido está fuera del horario de atención del espacio")
	}
	return ini, fin, nil
}

// ---------------------------------------------------------------- disponibilidad

func (s *service) Disponibilidad(ctx context.Context, fecha string, servicios []uuid.UUID, excluir *uuid.UUID) (*Disponibilidad, error) {
	f, err := s.validarFecha(fecha)
	if err != nil {
		return nil, err
	}

	var duracion int
	switch {
	case len(servicios) > 0:
		svcs, err := s.cargarServicios(ctx, servicios)
		if err != nil {
			return nil, err
		}
		duracion = sumaDuracion(svcs)
	case excluir != nil:
		// Al reprogramar se conserva la duración de la reserva original.
		r, err := s.repo.ObtenerReserva(ctx, *excluir)
		if err != nil {
			return nil, err
		}
		if r == nil {
			return nil, utils.NotFound("reserva no encontrada")
		}
		ini, _ := utils.ParseHM(r.HoraInicio)
		fin, _ := utils.ParseHM(r.HoraFin)
		duracion = fin - ini
	default:
		return nil, utils.BadRequest("selecciona al menos un servicio")
	}

	horarios, err := s.repo.HorariosActivosDia(ctx, int(f.Weekday()))
	if err != nil {
		return nil, err
	}
	ocupacion, err := s.repo.OcupacionDia(ctx, fecha, excluir)
	if err != nil {
		return nil, err
	}

	minInicio := s.minInicioPermitido(f)
	porEspacio := map[uuid.UUID]*EspacioDisponible{}
	orden := []uuid.UUID{}
	tramos := map[uuid.UUID][]Rango{}
	for _, h := range horarios {
		if _, ok := porEspacio[h.IDEspacio]; !ok {
			porEspacio[h.IDEspacio] = &EspacioDisponible{IDEspacio: h.IDEspacio, Codigo: h.Codigo, Slots: []Slot{}}
			orden = append(orden, h.IDEspacio)
		}
		tramos[h.IDEspacio] = append(tramos[h.IDEspacio], h.Tramo)
	}

	res := &Disponibilidad{Fecha: fecha, DuracionMin: duracion, Espacios: []EspacioDisponible{}}
	for _, id := range orden {
		for _, sl := range GenerarSlots(tramos[id], ocupacion[id], duracion, minInicio) {
			porEspacio[id].Slots = append(porEspacio[id].Slots, Slot{
				HoraInicio: utils.FormatHM(sl.Inicio),
				HoraFin:    utils.FormatHM(sl.Fin),
			})
		}
		if len(porEspacio[id].Slots) > 0 {
			res.Espacios = append(res.Espacios, *porEspacio[id])
		}
	}
	return res, nil
}

// ---------------------------------------------------------------- cliente

func (s *service) Crear(ctx context.Context, idCliente uuid.UUID, in CrearReservaInput) (*Reserva, error) {
	svcs, err := s.cargarServicios(ctx, in.Servicios)
	if err != nil {
		return nil, err
	}

	esSuyo, err := s.repo.VehiculoDeCliente(ctx, in.IDVehiculo, idCliente)
	if err != nil {
		return nil, err
	}
	if !esSuyo {
		return nil, utils.BadRequest("el vehículo seleccionado no existe o no es tuyo")
	}

	ini, fin, err := s.validarHueco(ctx, in.IDEspacio, in.Fecha, in.HoraInicio, sumaDuracion(svcs))
	if err != nil {
		return nil, err
	}

	total := 0.0
	for _, sv := range svcs {
		total += sv.Precio
	}

	var obs *string
	if t := strings.TrimSpace(in.Observaciones); t != "" {
		if len(t) > 300 {
			return nil, utils.BadRequest("las observaciones no pueden superar 300 caracteres")
		}
		obs = &t
	}

	r := &Reserva{
		IDReserva:     uuid.New(),
		IDCliente:     idCliente,
		IDVehiculo:    in.IDVehiculo,
		IDEspacio:     in.IDEspacio,
		FechaReserva:  strings.TrimSpace(in.Fecha),
		HoraInicio:    utils.FormatHM(ini),
		HoraFin:       utils.FormatHM(fin),
		Estado:        EstadoPendiente,
		TotalEstimado: math.Round(total*100) / 100,
		Observaciones: obs,
		FechaCreacion: s.ahora(),
	}
	if err := s.repo.CrearReserva(ctx, r, svcs); err != nil {
		return nil, err
	}
	return s.repo.ObtenerReserva(ctx, r.IDReserva)
}

func (s *service) MisReservas(ctx context.Context, idCliente uuid.UUID) ([]Reserva, error) {
	return s.repo.ListarPorCliente(ctx, idCliente)
}

func (s *service) Obtener(ctx context.Context, id uuid.UUID, idCliente *uuid.UUID) (*Reserva, error) {
	r, err := s.repo.ObtenerReserva(ctx, id)
	if err != nil {
		return nil, err
	}
	if r == nil || (idCliente != nil && r.IDCliente != *idCliente) {
		return nil, utils.NotFound("reserva no encontrada")
	}
	return r, nil
}

// modificable comprueba que la reserva aún se pueda reprogramar o cancelar.
func modificable(r *Reserva) error {
	switch r.Estado {
	case EstadoPendiente, EstadoConfirmada, EstadoReprogramada:
	default:
		return utils.Conflict("una reserva " + r.Estado + " ya no se puede modificar")
	}
	if r.EstadoAtencion != nil && *r.EstadoAtencion != estadoAtencionInicial {
		return utils.Conflict("la atención ya comenzó, la reserva no se puede modificar")
	}
	return nil
}

func (s *service) Reprogramar(ctx context.Context, idCliente uuid.UUID, id uuid.UUID, in ReprogramarInput) (*Reserva, error) {
	r, err := s.Obtener(ctx, id, &idCliente)
	if err != nil {
		return nil, err
	}
	if err := modificable(r); err != nil {
		return nil, err
	}

	// Se conserva la duración original de la reserva.
	iniOrig, _ := utils.ParseHM(r.HoraInicio)
	finOrig, _ := utils.ParseHM(r.HoraFin)
	ini, fin, err := s.validarHueco(ctx, in.IDEspacio, in.Fecha, in.HoraInicio, finOrig-iniOrig)
	if err != nil {
		return nil, err
	}

	if err := s.repo.ReprogramarReserva(ctx, id, strings.TrimSpace(in.Fecha),
		utils.FormatHM(ini), utils.FormatHM(fin), in.IDEspacio); err != nil {
		return nil, err
	}
	return s.repo.ObtenerReserva(ctx, id)
}

func (s *service) Cancelar(ctx context.Context, idCliente *uuid.UUID, id uuid.UUID, motivo string) (*Reserva, error) {
	r, err := s.Obtener(ctx, id, idCliente)
	if err != nil {
		return nil, err
	}
	if err := modificable(r); err != nil {
		return nil, err
	}

	motivo = strings.TrimSpace(motivo)
	if motivo == "" {
		if idCliente != nil {
			motivo = "Cancelada por el cliente"
		} else {
			motivo = "Cancelada por el administrador"
		}
	}
	if len(motivo) > 300 {
		return nil, utils.BadRequest("el motivo no puede superar 300 caracteres")
	}

	if err := s.repo.CancelarReserva(ctx, id, motivo); err != nil {
		return nil, err
	}
	return s.repo.ObtenerReserva(ctx, id)
}

// ---------------------------------------------------------------- administrador (RF-09)

func (s *service) ListarTodas(ctx context.Context, sede uuid.UUID, estado, fecha string) ([]Reserva, error) {
	switch estado {
	case "", EstadoPendiente, EstadoConfirmada, EstadoReprogramada, EstadoCancelada, EstadoCompletada:
	default:
		return nil, utils.BadRequest("estado de reserva no válido")
	}
	if fecha != "" {
		if _, err := utils.ParseFecha(fecha); err != nil {
			return nil, utils.BadRequest(err.Error())
		}
	}
	return s.repo.ListarTodas(ctx, sede, estado, fecha)
}

// Agenda arma la planificación del día: todos los espacios activos (aunque no tengan
// reservas) y sus reservas no canceladas ordenadas por hora de inicio.
func (s *service) Agenda(ctx context.Context, fecha string) (*AgendaDia, error) {
	fecha = strings.TrimSpace(fecha)
	if fecha == "" {
		fecha = s.ahora().Format("2006-01-02")
	}
	if _, err := utils.ParseFecha(fecha); err != nil {
		return nil, utils.BadRequest(err.Error())
	}

	espacios, err := s.repo.EspaciosParaAgenda(ctx)
	if err != nil {
		return nil, err
	}
	reservas, err := s.repo.ListarTodas(ctx, uuid.Nil, "", fecha)
	if err != nil {
		return nil, err
	}

	porEspacio := map[uuid.UUID][]Reserva{}
	for _, r := range reservas {
		if r.Estado == EstadoCancelada {
			continue
		}
		porEspacio[r.IDEspacio] = append(porEspacio[r.IDEspacio], r)
	}

	out := &AgendaDia{Fecha: fecha, Espacios: make([]AgendaEspacio, 0, len(espacios))}
	for _, e := range espacios {
		lista := porEspacio[e.IDEspacio]
		if lista == nil {
			lista = []Reserva{}
		}
		out.Espacios = append(out.Espacios, AgendaEspacio{
			IDEspacio: e.IDEspacio,
			Codigo:    e.Codigo,
			Activo:    e.Activo,
			Reservas:  lista,
		})
	}
	return out, nil
}

func (s *service) TrabajadoresDisponibles(ctx context.Context, id uuid.UUID) ([]TrabajadorDisponible, error) {
	r, err := s.Obtener(ctx, id, nil)
	if err != nil {
		return nil, err
	}
	return s.repo.TrabajadoresParaReserva(ctx, r)
}

// Programar confirma la reserva: exige los documentos previos validados (RF-07),
// crea la atención "programada" y le asigna uno o varios trabajadores.
func (s *service) Programar(ctx context.Context, id, idAdmin uuid.UUID, in ProgramarInput) (*Reserva, error) {
	r, err := s.Obtener(ctx, id, nil)
	if err != nil {
		return nil, err
	}
	switch r.Estado {
	case EstadoPendiente, EstadoReprogramada, EstadoConfirmada:
	default:
		return nil, utils.Conflict("una reserva " + r.Estado + " no se puede programar")
	}
	if r.EstadoAtencion != nil && *r.EstadoAtencion != estadoAtencionInicial {
		return nil, utils.Conflict("la atención ya comenzó, no se pueden cambiar los trabajadores")
	}

	trabajadores := dedup(in.Trabajadores)
	if len(trabajadores) == 0 {
		return nil, utils.BadRequest("asigna al menos un trabajador")
	}
	if len(trabajadores) > maxTrabajadores {
		return nil, utils.BadRequest("una atención admite hasta 5 trabajadores")
	}

	// La fecha de la reserva no puede haber pasado.
	fecha, err := utils.ParseFecha(r.FechaReserva)
	if err != nil {
		return nil, err
	}
	if fecha.Before(inicioDelDia(s.ahora())) {
		return nil, utils.BadRequest("no se puede programar una reserva de una fecha pasada")
	}

	// RF-07: no se confirma sin los documentos previos validados.
	if err := documentos.ExigirDocumentosValidados(ctx, s.docs, id); err != nil {
		return nil, err
	}

	if err := s.repo.ProgramarReserva(ctx, id, idAdmin, trabajadores); err != nil {
		return nil, err
	}
	return s.repo.ObtenerReserva(ctx, id)
}
