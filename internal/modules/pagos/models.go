package pagos

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	TipoAtencion = "atencion"
	TipoPedido   = "pedido"
	Pagado       = "pagado"
	Reembolsado  = "reembolsado"
)

type Pago struct {
	IDPago             uuid.UUID       `json:"id_pago"`
	IDAtencion         *uuid.UUID      `json:"id_atencion,omitempty"`
	IDPedido           *uuid.UUID      `json:"id_pedido,omitempty"`
	Tipo               string          `json:"tipo"`
	Monto              string          `json:"monto"`
	Moneda             string          `json:"moneda"`
	Metodo             string          `json:"metodo"`
	Estado             string          `json:"estado"`
	ComprobanteInterno string          `json:"comprobante_interno"`
	ReferenciaExterna  *string         `json:"referencia_externa,omitempty"`
	FechaPago          time.Time       `json:"fecha_pago"`
	FechaReversion     *time.Time      `json:"fecha_reversion,omitempty"`
	MotivoReversion    *string         `json:"motivo_reversion,omitempty"`
	Cliente            string          `json:"cliente"`
	Detalle            json.RawMessage `json:"detalle"`
}

type OperacionPendiente struct {
	Tipo        string          `json:"tipo"`
	IDOperacion uuid.UUID       `json:"id_operacion"`
	Cliente     string          `json:"cliente"`
	Descripcion string          `json:"descripcion"`
	Monto       string          `json:"monto"`
	Moneda      string          `json:"moneda"`
	Fecha       time.Time       `json:"fecha"`
	Detalle     json.RawMessage `json:"detalle"`
}

type RegistrarPagoInput struct {
	Tipo              string    `json:"tipo"`
	IDOperacion       uuid.UUID `json:"id_operacion"`
	Metodo            string    `json:"metodo"`
	ReferenciaExterna *string   `json:"referencia_externa"`
	IdempotencyKey    string    `json:"idempotency_key"`
}

type RevertirPagoInput struct {
	Motivo              string `json:"motivo"`
	ReembolsoConfirmado bool   `json:"reembolso_confirmado"`
}
