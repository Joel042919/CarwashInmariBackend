package reportes

import "carwashinmaribackend/internal/modules/atenciones"

type Periodo struct {
	Desde string `json:"desde"`
	Hasta string `json:"hasta"`
}

type MovimientoCategoria struct {
	Categoria string `json:"categoria"`
	Bruto     string `json:"bruto"`
	Reembolso string `json:"reembolso"`
	Neto      string `json:"neto"`
}

type PuntoFinanciero struct {
	Fecha       string `json:"fecha"`
	Bruto       string `json:"bruto"`
	Reembolsado string `json:"reembolsado"`
	Neto        string `json:"neto"`
}

type Ingresos struct {
	Brutos       string                `json:"brutos"`
	Reembolsados string                `json:"reembolsados"`
	Netos        string                `json:"netos"`
	PorTipo      []MovimientoCategoria `json:"por_tipo"`
	PorMetodo    []MovimientoCategoria `json:"por_metodo"`
	Serie        []PuntoFinanciero     `json:"serie_diaria"`
}

type Servicio struct {
	IDServicio string `json:"id_servicio"`
	Nombre     string `json:"nombre"`
	Cantidad   int    `json:"cantidad"`
	Valor      string `json:"valor_operativo"`
}

type EstadoConteo struct {
	Estado   string `json:"estado"`
	Cantidad int    `json:"cantidad"`
}

type Ventas struct {
	PedidosPorEstado []EstadoConteo `json:"pedidos_por_estado"`
	UnidadesVendidas int            `json:"unidades_vendidas"`
	CobradoBruto     string         `json:"cobrado_bruto"`
	Reembolsado      string         `json:"reembolsado"`
	CobradoNeto      string         `json:"cobrado_neto"`
}

type Reservas struct {
	Total           int            `json:"total"`
	Canceladas      int            `json:"canceladas"`
	TasaCancelacion float64        `json:"tasa_cancelacion"`
	PorEstado       []EstadoConteo `json:"por_estado"`
}

type DemandaDia struct {
	Fecha      string `json:"fecha"`
	Total      int    `json:"total"`
	Canceladas int    `json:"canceladas"`
}

type DemandaCategoria struct {
	Categoria  string `json:"categoria"`
	Total      int    `json:"total"`
	Canceladas int    `json:"canceladas"`
}

type Demanda struct {
	PorDia      []DemandaDia       `json:"por_dia"`
	PorHorario  []DemandaCategoria `json:"por_horario"`
	PorServicio []DemandaCategoria `json:"por_servicio"`
}

type TipoVehiculo struct {
	Tipo               string `json:"tipo"`
	Atenciones         int    `json:"atenciones"`
	VehiculosDistintos int    `json:"vehiculos_distintos"`
}

type Dashboard struct {
	Periodo       Periodo                        `json:"periodo"`
	Ingresos      Ingresos                       `json:"ingresos"`
	Servicios     []Servicio                     `json:"servicios"`
	Ventas        Ventas                         `json:"ventas"`
	Reservas      Reservas                       `json:"reservas"`
	Demanda       Demanda                        `json:"demanda"`
	Productividad []atenciones.RendimientoEquipo `json:"productividad"`
	Vehiculos     []TipoVehiculo                 `json:"tipos_vehiculo"`
	Notas         []string                       `json:"notas"`
}
