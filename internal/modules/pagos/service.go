package pagos

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"carwashinmaribackend/internal/utils"
	"github.com/google/uuid"
)

var metodos = map[string]bool{
	"efectivo": true, "tarjeta": true, "transferencia": true, "billetera_digital": true,
}

type Repository interface {
	Listar(context.Context, uuid.UUID, string) ([]Pago, error)
	ListarPorCliente(context.Context, uuid.UUID) ([]Pago, error)
	ListarPendientes(context.Context, uuid.UUID, string) ([]OperacionPendiente, error)
	Registrar(context.Context, uuid.UUID, uuid.UUID, RegistrarPagoInput, string) (*Pago, error)
	ObtenerComprobante(context.Context, utils.CustomClaims, uuid.UUID) (*Pago, error)
	Revertir(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, RevertirPagoInput) (*Pago, error)
}

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) Listar(ctx context.Context, sede uuid.UUID, estado string) ([]Pago, error) {
	switch estado {
	case "", Pagado, Reembolsado, "anulado", "pendiente":
	default:
		return nil, utils.BadRequest("estado de pago inválido")
	}
	return s.repo.Listar(ctx, sede, estado)
}

func (s *Service) ListarPorCliente(ctx context.Context, cliente uuid.UUID) ([]Pago, error) {
	return s.repo.ListarPorCliente(ctx, cliente)
}

func (s *Service) ListarPendientes(ctx context.Context, sede uuid.UUID, tipo string) ([]OperacionPendiente, error) {
	if tipo != "" && tipo != TipoAtencion && tipo != TipoPedido {
		return nil, utils.BadRequest("tipo de operación inválido")
	}
	return s.repo.ListarPendientes(ctx, sede, tipo)
}

func (s *Service) Registrar(ctx context.Context, sede, admin uuid.UUID, in RegistrarPagoInput) (*Pago, error) {
	in.Tipo = strings.ToLower(strings.TrimSpace(in.Tipo))
	in.Metodo = strings.ToLower(strings.TrimSpace(in.Metodo))
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	if in.Tipo != TipoAtencion && in.Tipo != TipoPedido {
		return nil, utils.BadRequest("tipo de operación inválido")
	}
	if in.IDOperacion == uuid.Nil {
		return nil, utils.BadRequest("la operación es obligatoria")
	}
	if !metodos[in.Metodo] {
		return nil, utils.BadRequest("método de pago inválido")
	}
	if len(in.IdempotencyKey) < 8 || len(in.IdempotencyKey) > 100 {
		return nil, utils.BadRequest("la clave de idempotencia debe tener entre 8 y 100 caracteres")
	}
	if in.ReferenciaExterna != nil {
		referencia := strings.TrimSpace(*in.ReferenciaExterna)
		if len(referencia) > 100 {
			return nil, utils.BadRequest("la referencia externa admite hasta 100 caracteres")
		}
		in.ReferenciaExterna = nil
		if referencia != "" {
			in.ReferenciaExterna = &referencia
		}
	}
	resumen := in.Tipo + "|" + in.IDOperacion.String() + "|" + in.Metodo + "|"
	if in.ReferenciaExterna != nil {
		resumen += *in.ReferenciaExterna
	}
	hash := sha256.Sum256([]byte(resumen))
	return s.repo.Registrar(ctx, sede, admin, in, hex.EncodeToString(hash[:]))
}

func (s *Service) ObtenerComprobante(ctx context.Context, actor utils.CustomClaims, id uuid.UUID) (*Pago, error) {
	return s.repo.ObtenerComprobante(ctx, actor, id)
}

func (s *Service) Revertir(ctx context.Context, sede, admin, id uuid.UUID, in RevertirPagoInput) (*Pago, error) {
	in.Motivo = strings.TrimSpace(in.Motivo)
	if len(in.Motivo) < 8 || len(in.Motivo) > 300 {
		return nil, utils.BadRequest("el motivo debe tener entre 8 y 300 caracteres")
	}
	if !in.ReembolsoConfirmado {
		return nil, utils.BadRequest("confirma que la devolución fue realizada antes de registrar la reversión")
	}
	return s.repo.Revertir(ctx, sede, admin, id, in)
}
