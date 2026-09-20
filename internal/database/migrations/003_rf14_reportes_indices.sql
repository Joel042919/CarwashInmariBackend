CREATE INDEX IF NOT EXISTS pagos_fecha_reversion_idx
    ON pagos (fecha_reversion) WHERE estado = 'reembolsado';

CREATE INDEX IF NOT EXISTS atenciones_fin_estado_idx
    ON atenciones (fecha_fin_real, estado);

CREATE INDEX IF NOT EXISTS reservas_fecha_estado_idx
    ON reservas (fecha_reserva, estado);

CREATE INDEX IF NOT EXISTS asignaciones_atencion_trabajador_idx
    ON asignaciones_trabajadores (id_atencion, id_trabajador);

CREATE INDEX IF NOT EXISTS reserva_servicios_reserva_servicio_idx
    ON reserva_servicios (id_reserva, id_servicio);

CREATE INDEX IF NOT EXISTS pedidos_fecha_estado_idx
    ON pedidos (fecha_registro, estado);
