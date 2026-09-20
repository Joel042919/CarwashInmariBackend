ALTER TABLE pagos
    ADD COLUMN IF NOT EXISTS moneda character(3) NOT NULL DEFAULT 'PEN',
    ADD COLUMN IF NOT EXISTS referencia_externa character varying(100),
    ADD COLUMN IF NOT EXISTS registrado_por uuid REFERENCES usuarios(id_usuario),
    ADD COLUMN IF NOT EXISTS idempotency_key character varying(100),
    ADD COLUMN IF NOT EXISTS request_hash character(64),
    ADD COLUMN IF NOT EXISTS detalle_snapshot jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS revertido_por uuid REFERENCES usuarios(id_usuario),
    ADD COLUMN IF NOT EXISTS motivo_reversion text,
    ADD COLUMN IF NOT EXISTS fecha_reversion timestamp with time zone;

ALTER TABLE pagos DROP CONSTRAINT IF EXISTS pagos_operacion_exclusiva_ck;
ALTER TABLE pagos ADD CONSTRAINT pagos_operacion_exclusiva_ck CHECK (
    (id_atencion IS NOT NULL AND id_pedido IS NULL) OR
    (id_atencion IS NULL AND id_pedido IS NOT NULL)
);

CREATE UNIQUE INDEX IF NOT EXISTS pagos_idempotencia_uq
    ON pagos (registrado_por, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS pagos_atencion_activa_uq
    ON pagos (id_atencion)
    WHERE id_atencion IS NOT NULL AND estado = 'pagado';

CREATE UNIQUE INDEX IF NOT EXISTS pagos_pedido_activo_uq
    ON pagos (id_pedido)
    WHERE id_pedido IS NOT NULL AND estado = 'pagado';

CREATE INDEX IF NOT EXISTS pagos_fecha_estado_idx ON pagos (fecha_pago, estado);
