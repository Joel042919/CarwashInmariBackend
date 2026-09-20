--
-- PostgreSQL database dump
--

\restrict IVOxqDO1Ipz63jBlaiXa16TjsD21Uklk6pYMX319fqLrKgFTHPyVJCm3sIN60eX

-- Dumped from database version 18.6 (6569466)
-- Dumped by pg_dump version 18.4

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: public; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA public;


--
-- Name: SCHEMA public; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON SCHEMA public IS 'standard public schema';


--
-- Name: estado_atencion; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.estado_atencion AS ENUM (
    'programada',
    'en_proceso',
    'en_pausa',
    'finalizada',
    'entregada'
);


--
-- Name: estado_documento; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.estado_documento AS ENUM (
    'adjuntado',
    'validado',
    'rechazado'
);


--
-- Name: estado_pago; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.estado_pago AS ENUM (
    'pendiente',
    'pagado',
    'anulado',
    'reembolsado'
);


--
-- Name: estado_pedido; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.estado_pedido AS ENUM (
    'registrado',
    'pagado',
    'preparando',
    'entregado',
    'cancelado'
);


--
-- Name: estado_reclamo; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.estado_reclamo AS ENUM (
    'registrado',
    'en_revision',
    'respondido',
    'cerrado'
);


--
-- Name: estado_reserva; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.estado_reserva AS ENUM (
    'pendiente',
    'confirmada',
    'reprogramada',
    'cancelada',
    'completada'
);


--
-- Name: metodo_pago; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.metodo_pago AS ENUM (
    'efectivo',
    'tarjeta',
    'transferencia',
    'billetera_digital'
);


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: asignaciones_trabajadores; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.asignaciones_trabajadores (
    id_asignacion uuid DEFAULT gen_random_uuid() NOT NULL,
    id_atencion uuid NOT NULL,
    id_trabajador uuid NOT NULL,
    rol_en_atencion character varying(50),
    horas_trabajadas numeric(5,2)
);


--
-- Name: atenciones; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.atenciones (
    id_atencion uuid DEFAULT gen_random_uuid() NOT NULL,
    id_reserva uuid,
    fecha_inicio_real timestamp with time zone,
    fecha_fin_real timestamp with time zone,
    estado public.estado_atencion DEFAULT 'programada'::public.estado_atencion NOT NULL,
    observaciones_internas jsonb
);


--
-- Name: categoria_servicios; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.categoria_servicios (
    id_categoria_servicio uuid DEFAULT gen_random_uuid() NOT NULL,
    categoria_servicio character varying(30) NOT NULL
);


--
-- Name: categorias_producto; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.categorias_producto (
    id_categoria uuid DEFAULT gen_random_uuid() NOT NULL,
    nombre character varying(60) NOT NULL,
    descripcion character varying(200),
    activo boolean DEFAULT true NOT NULL
);


--
-- Name: clientes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.clientes (
    id_usuario uuid NOT NULL,
    dni character varying(20),
    fecha_nacimiento date,
    direccion character varying(200),
    notas text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: detalle_pedido; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.detalle_pedido (
    id_detalle uuid DEFAULT gen_random_uuid() NOT NULL,
    id_pedido uuid NOT NULL,
    id_producto uuid NOT NULL,
    cantidad smallint NOT NULL,
    precio_unitario numeric(10,2) NOT NULL,
    subtotal numeric(10,2) NOT NULL
);


--
-- Name: documentos_previos; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.documentos_previos (
    id_documento uuid DEFAULT gen_random_uuid() NOT NULL,
    id_reserva uuid NOT NULL,
    id_servicio uuid NOT NULL,
    id_cliente uuid NOT NULL,
    ruta_pdf character varying(255) NOT NULL,
    estado public.estado_documento DEFAULT 'adjuntado'::public.estado_documento NOT NULL,
    admin_respuesta text,
    validado_por uuid,
    fecha_validacion timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE documentos_previos; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.documentos_previos IS 'RF-07: si el servicio requiere_documento = true, la reserva no pasa a confirmada hasta tener un documento con estado "validado"';


--
-- Name: espacios_lavado; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.espacios_lavado (
    id_espacio uuid DEFAULT gen_random_uuid() NOT NULL,
    codigo character varying(20) NOT NULL,
    id_tipo_lavadero uuid,
    activo boolean DEFAULT true NOT NULL
);


--
-- Name: evidencias; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.evidencias (
    id_evidencia uuid DEFAULT gen_random_uuid() NOT NULL,
    id_atencion uuid NOT NULL,
    registrado_por uuid NOT NULL,
    id_vehiculo uuid NOT NULL,
    tipo character varying(20) DEFAULT 'antes'::character varying NOT NULL,
    descripcion text,
    ruta_foto character varying(255) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: evidencias_reclamo; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.evidencias_reclamo (
    id_evidencia uuid DEFAULT gen_random_uuid() NOT NULL,
    id_reclamo uuid NOT NULL,
    ruta_archivo character varying(255) NOT NULL,
    descripcion character varying(200),
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: historial_estados_atencion; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.historial_estados_atencion (
    id_historial uuid DEFAULT gen_random_uuid() NOT NULL,
    id_atencion uuid NOT NULL,
    estado public.estado_atencion NOT NULL,
    registrado_por uuid NOT NULL,
    comentario character varying(255),
    fecha_cambio timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: horarios_atencion; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.horarios_atencion (
    id_horario uuid DEFAULT gen_random_uuid() NOT NULL,
    id_espacio uuid NOT NULL,
    dia_semana smallint NOT NULL,
    hora_inicio time without time zone NOT NULL,
    hora_fin time without time zone
);


--
-- Name: COLUMN horarios_atencion.dia_semana; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.horarios_atencion.dia_semana IS '0=Domingo ... 6=Sábado';


--
-- Name: pagos; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.pagos (
    id_pago uuid DEFAULT gen_random_uuid() NOT NULL,
    id_atencion uuid,
    id_pedido uuid,
    monto numeric(10,2) NOT NULL,
    metodo public.metodo_pago NOT NULL,
    estado public.estado_pago DEFAULT 'pendiente'::public.estado_pago NOT NULL,
    comprobante_interno character varying(30) NOT NULL,
    registrado_por uuid,
    fecha_pago timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: pedidos; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.pedidos (
    id_pedido uuid DEFAULT gen_random_uuid() NOT NULL,
    id_cliente uuid NOT NULL,
    estado public.estado_pedido DEFAULT 'registrado'::public.estado_pedido NOT NULL,
    total numeric(10,2) NOT NULL,
    observaciones text,
    fecha_registro timestamp with time zone DEFAULT now() NOT NULL,
    fecha_entrega timestamp with time zone
);


--
-- Name: productos; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.productos (
    id_producto uuid DEFAULT gen_random_uuid() NOT NULL,
    id_categoria uuid NOT NULL,
    codigo character varying(30) NOT NULL,
    nombre character varying(100) NOT NULL,
    descripcion text,
    precio_venta numeric(10,2) NOT NULL,
    stock integer DEFAULT 0 NOT NULL,
    stock_minimo integer DEFAULT 5 NOT NULL,
    activo boolean DEFAULT true NOT NULL
);


--
-- Name: reclamos; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.reclamos (
    id_reclamo uuid DEFAULT gen_random_uuid() NOT NULL,
    id_cliente uuid NOT NULL,
    id_atencion uuid,
    id_pago uuid,
    id_pedido uuid,
    asunto character varying(120) NOT NULL,
    descripcion text NOT NULL,
    estado public.estado_reclamo DEFAULT 'registrado'::public.estado_reclamo NOT NULL,
    respuesta_admin text,
    respondido_por uuid,
    fecha_registro timestamp with time zone DEFAULT now() NOT NULL,
    fecha_respuesta timestamp with time zone
);


--
-- Name: reserva_servicios; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.reserva_servicios (
    id_reserva_servicio uuid DEFAULT gen_random_uuid() NOT NULL,
    id_reserva uuid NOT NULL,
    id_servicio uuid NOT NULL,
    precio_unitario numeric(10,2) NOT NULL,
    cantidad smallint DEFAULT 1 NOT NULL
);


--
-- Name: reservas; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.reservas (
    id_reserva uuid DEFAULT gen_random_uuid() NOT NULL,
    id_cliente uuid NOT NULL,
    id_trabajador uuid,
    id_vehiculo uuid NOT NULL,
    id_espacio uuid NOT NULL,
    fecha_reserva date NOT NULL,
    hora_inicio time without time zone NOT NULL,
    hora_fin time without time zone NOT NULL,
    estado public.estado_reserva DEFAULT 'pendiente'::public.estado_reserva NOT NULL,
    total_estimado numeric(10,2) NOT NULL,
    observaciones text,
    motivo_cancelacion text,
    fecha_creacion timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE reservas; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.reservas IS 'Evitar cruces: UNIQUE parcial sobre (id_espacio, fecha_reserva, hora_inicio) donde estado <> cancelada';


--
-- Name: rol; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.rol (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    rol character varying(30) NOT NULL
);


--
-- Name: sede; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sede (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    sede_numero integer NOT NULL,
    direccion text,
    logo_url text,
    horario_atencion jsonb
);


--
-- Name: servicios; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.servicios (
    id_servicio uuid DEFAULT gen_random_uuid() NOT NULL,
    nombre character varying(80) NOT NULL,
    descripcion text,
    precio numeric(10,2) NOT NULL,
    duracion_estimada_min integer NOT NULL,
    requiere_documento boolean DEFAULT false NOT NULL,
    disponible boolean DEFAULT true NOT NULL,
    id_categoria uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: tipo_lavadero; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tipo_lavadero (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tipo_lavadero character varying(30) NOT NULL
);


--
-- Name: trabajadores; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.trabajadores (
    id_usuario uuid NOT NULL,
    dni character varying(20) NOT NULL,
    fecha_contratacion date NOT NULL,
    especialidades jsonb,
    disponible boolean DEFAULT true NOT NULL,
    fecha_cese date
);


--
-- Name: usuarios; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.usuarios (
    id_usuario uuid DEFAULT gen_random_uuid() NOT NULL,
    id_sede uuid NOT NULL,
    nombre character varying(100) NOT NULL,
    apellido character varying(100) NOT NULL,
    correo character varying(150) NOT NULL,
    telefono character varying(20),
    contrasena_hash character varying(255) NOT NULL,
    id_rol uuid NOT NULL,
    activo boolean DEFAULT true NOT NULL,
    fecha_registro timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: vehiculos; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.vehiculos (
    id_vehiculo uuid DEFAULT gen_random_uuid() NOT NULL,
    id_cliente uuid NOT NULL,
    placa character varying(15) NOT NULL,
    marca character varying(50) NOT NULL,
    modelo character varying(50) NOT NULL,
    color character varying(30),
    anio integer,
    ruta_imagen text,
    tipo_vehiculo character varying(30),
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: asignaciones_trabajadores asignaciones_trabajadores_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.asignaciones_trabajadores
    ADD CONSTRAINT asignaciones_trabajadores_pkey PRIMARY KEY (id_asignacion);


--
-- Name: atenciones atenciones_id_reserva_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.atenciones
    ADD CONSTRAINT atenciones_id_reserva_key UNIQUE (id_reserva);


--
-- Name: atenciones atenciones_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.atenciones
    ADD CONSTRAINT atenciones_pkey PRIMARY KEY (id_atencion);


--
-- Name: categoria_servicios categoria_servicios_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.categoria_servicios
    ADD CONSTRAINT categoria_servicios_pkey PRIMARY KEY (id_categoria_servicio);


--
-- Name: categorias_producto categorias_producto_nombre_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.categorias_producto
    ADD CONSTRAINT categorias_producto_nombre_key UNIQUE (nombre);


--
-- Name: categorias_producto categorias_producto_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.categorias_producto
    ADD CONSTRAINT categorias_producto_pkey PRIMARY KEY (id_categoria);


--
-- Name: clientes clientes_dni_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.clientes
    ADD CONSTRAINT clientes_dni_key UNIQUE (dni);


--
-- Name: clientes clientes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.clientes
    ADD CONSTRAINT clientes_pkey PRIMARY KEY (id_usuario);


--
-- Name: detalle_pedido detalle_pedido_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.detalle_pedido
    ADD CONSTRAINT detalle_pedido_pkey PRIMARY KEY (id_detalle);


--
-- Name: documentos_previos documentos_previos_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.documentos_previos
    ADD CONSTRAINT documentos_previos_pkey PRIMARY KEY (id_documento);


--
-- Name: espacios_lavado espacios_lavado_codigo_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.espacios_lavado
    ADD CONSTRAINT espacios_lavado_codigo_key UNIQUE (codigo);


--
-- Name: espacios_lavado espacios_lavado_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.espacios_lavado
    ADD CONSTRAINT espacios_lavado_pkey PRIMARY KEY (id_espacio);


--
-- Name: evidencias evidencias_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.evidencias
    ADD CONSTRAINT evidencias_pkey PRIMARY KEY (id_evidencia);


--
-- Name: evidencias_reclamo evidencias_reclamo_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.evidencias_reclamo
    ADD CONSTRAINT evidencias_reclamo_pkey PRIMARY KEY (id_evidencia);


--
-- Name: historial_estados_atencion historial_estados_atencion_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.historial_estados_atencion
    ADD CONSTRAINT historial_estados_atencion_pkey PRIMARY KEY (id_historial);


--
-- Name: horarios_atencion horarios_atencion_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.horarios_atencion
    ADD CONSTRAINT horarios_atencion_pkey PRIMARY KEY (id_horario);


--
-- Name: pagos pagos_comprobante_interno_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pagos
    ADD CONSTRAINT pagos_comprobante_interno_key UNIQUE (comprobante_interno);


--
-- Name: pagos pagos_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pagos
    ADD CONSTRAINT pagos_pkey PRIMARY KEY (id_pago);


--
-- Name: pedidos pedidos_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pedidos
    ADD CONSTRAINT pedidos_pkey PRIMARY KEY (id_pedido);


--
-- Name: productos productos_codigo_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.productos
    ADD CONSTRAINT productos_codigo_key UNIQUE (codigo);


--
-- Name: productos productos_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.productos
    ADD CONSTRAINT productos_pkey PRIMARY KEY (id_producto);


--
-- Name: reclamos reclamos_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.reclamos
    ADD CONSTRAINT reclamos_pkey PRIMARY KEY (id_reclamo);


--
-- Name: reserva_servicios reserva_servicios_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.reserva_servicios
    ADD CONSTRAINT reserva_servicios_pkey PRIMARY KEY (id_reserva_servicio);


--
-- Name: reservas reservas_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.reservas
    ADD CONSTRAINT reservas_pkey PRIMARY KEY (id_reserva);


--
-- Name: rol rol_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rol
    ADD CONSTRAINT rol_pkey PRIMARY KEY (id);


--
-- Name: sede sede_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sede
    ADD CONSTRAINT sede_pkey PRIMARY KEY (id);


--
-- Name: servicios servicios_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.servicios
    ADD CONSTRAINT servicios_pkey PRIMARY KEY (id_servicio);


--
-- Name: tipo_lavadero tipo_lavadero_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tipo_lavadero
    ADD CONSTRAINT tipo_lavadero_pkey PRIMARY KEY (id);


--
-- Name: trabajadores trabajadores_dni_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trabajadores
    ADD CONSTRAINT trabajadores_dni_key UNIQUE (dni);


--
-- Name: trabajadores trabajadores_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trabajadores
    ADD CONSTRAINT trabajadores_pkey PRIMARY KEY (id_usuario);


--
-- Name: usuarios usuarios_correo_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.usuarios
    ADD CONSTRAINT usuarios_correo_key UNIQUE (correo);


--
-- Name: usuarios usuarios_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.usuarios
    ADD CONSTRAINT usuarios_pkey PRIMARY KEY (id_usuario);


--
-- Name: vehiculos vehiculos_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vehiculos
    ADD CONSTRAINT vehiculos_pkey PRIMARY KEY (id_vehiculo);


--
-- Name: vehiculos vehiculos_placa_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vehiculos
    ADD CONSTRAINT vehiculos_placa_key UNIQUE (placa);


--
-- Name: idx_asig_atencion; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_asig_atencion ON public.asignaciones_trabajadores USING btree (id_atencion);


--
-- Name: idx_asig_trabajador; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_asig_trabajador ON public.asignaciones_trabajadores USING btree (id_trabajador);


--
-- Name: idx_doc_reserva; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_doc_reserva ON public.documentos_previos USING btree (id_reserva);


--
-- Name: idx_evid_atencion; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_evid_atencion ON public.evidencias USING btree (id_atencion);


--
-- Name: idx_hist_atencion; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_hist_atencion ON public.historial_estados_atencion USING btree (id_atencion);


--
-- Name: idx_horario_espacio_dia; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_horario_espacio_dia ON public.horarios_atencion USING btree (id_espacio, dia_semana);


--
-- Name: idx_pago_atencion; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pago_atencion ON public.pagos USING btree (id_atencion);


--
-- Name: idx_pago_estado; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pago_estado ON public.pagos USING btree (estado);


--
-- Name: idx_pedido_cliente; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pedido_cliente ON public.pedidos USING btree (id_cliente);


--
-- Name: idx_prod_categoria; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_prod_categoria ON public.productos USING btree (id_categoria);


--
-- Name: idx_reclamo_cliente; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_reclamo_cliente ON public.reclamos USING btree (id_cliente);


--
-- Name: idx_reclamo_estado; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_reclamo_estado ON public.reclamos USING btree (estado);


--
-- Name: idx_reserva_cliente; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_reserva_cliente ON public.reservas USING btree (id_cliente);


--
-- Name: idx_reserva_espacio_fecha; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_reserva_espacio_fecha ON public.reservas USING btree (id_espacio, fecha_reserva, hora_inicio);


--
-- Name: idx_vehiculo_cliente; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_vehiculo_cliente ON public.vehiculos USING btree (id_cliente);


--
-- Name: idx_vehiculo_placa; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_vehiculo_placa ON public.vehiculos USING btree (placa);


--
-- Name: asignaciones_trabajadores asignaciones_trabajadores_id_atencion_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.asignaciones_trabajadores
    ADD CONSTRAINT asignaciones_trabajadores_id_atencion_fkey FOREIGN KEY (id_atencion) REFERENCES public.atenciones(id_atencion) DEFERRABLE;


--
-- Name: asignaciones_trabajadores asignaciones_trabajadores_id_trabajador_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.asignaciones_trabajadores
    ADD CONSTRAINT asignaciones_trabajadores_id_trabajador_fkey FOREIGN KEY (id_trabajador) REFERENCES public.usuarios(id_usuario) DEFERRABLE;


--
-- Name: atenciones atenciones_id_reserva_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.atenciones
    ADD CONSTRAINT atenciones_id_reserva_fkey FOREIGN KEY (id_reserva) REFERENCES public.reservas(id_reserva) DEFERRABLE;


--
-- Name: clientes clientes_id_usuario_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.clientes
    ADD CONSTRAINT clientes_id_usuario_fkey FOREIGN KEY (id_usuario) REFERENCES public.usuarios(id_usuario) DEFERRABLE;


--
-- Name: detalle_pedido detalle_pedido_id_pedido_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.detalle_pedido
    ADD CONSTRAINT detalle_pedido_id_pedido_fkey FOREIGN KEY (id_pedido) REFERENCES public.pedidos(id_pedido) DEFERRABLE;


--
-- Name: detalle_pedido detalle_pedido_id_producto_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.detalle_pedido
    ADD CONSTRAINT detalle_pedido_id_producto_fkey FOREIGN KEY (id_producto) REFERENCES public.productos(id_producto) DEFERRABLE;


--
-- Name: documentos_previos documentos_previos_id_cliente_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.documentos_previos
    ADD CONSTRAINT documentos_previos_id_cliente_fkey FOREIGN KEY (id_cliente) REFERENCES public.usuarios(id_usuario) DEFERRABLE;


--
-- Name: documentos_previos documentos_previos_id_reserva_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.documentos_previos
    ADD CONSTRAINT documentos_previos_id_reserva_fkey FOREIGN KEY (id_reserva) REFERENCES public.reservas(id_reserva) DEFERRABLE;


--
-- Name: documentos_previos documentos_previos_id_servicio_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.documentos_previos
    ADD CONSTRAINT documentos_previos_id_servicio_fkey FOREIGN KEY (id_servicio) REFERENCES public.servicios(id_servicio) DEFERRABLE;


--
-- Name: documentos_previos documentos_previos_validado_por_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.documentos_previos
    ADD CONSTRAINT documentos_previos_validado_por_fkey FOREIGN KEY (validado_por) REFERENCES public.usuarios(id_usuario) DEFERRABLE;


--
-- Name: espacios_lavado espacios_lavado_id_tipo_lavadero_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.espacios_lavado
    ADD CONSTRAINT espacios_lavado_id_tipo_lavadero_fkey FOREIGN KEY (id_tipo_lavadero) REFERENCES public.tipo_lavadero(id) DEFERRABLE;


--
-- Name: evidencias evidencias_id_atencion_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.evidencias
    ADD CONSTRAINT evidencias_id_atencion_fkey FOREIGN KEY (id_atencion) REFERENCES public.atenciones(id_atencion) DEFERRABLE;


--
-- Name: evidencias evidencias_id_vehiculo_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.evidencias
    ADD CONSTRAINT evidencias_id_vehiculo_fkey FOREIGN KEY (id_vehiculo) REFERENCES public.vehiculos(id_vehiculo) DEFERRABLE;


--
-- Name: evidencias_reclamo evidencias_reclamo_id_reclamo_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.evidencias_reclamo
    ADD CONSTRAINT evidencias_reclamo_id_reclamo_fkey FOREIGN KEY (id_reclamo) REFERENCES public.reclamos(id_reclamo) DEFERRABLE;


--
-- Name: evidencias evidencias_registrado_por_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.evidencias
    ADD CONSTRAINT evidencias_registrado_por_fkey FOREIGN KEY (registrado_por) REFERENCES public.usuarios(id_usuario) DEFERRABLE;


--
-- Name: historial_estados_atencion historial_estados_atencion_id_atencion_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.historial_estados_atencion
    ADD CONSTRAINT historial_estados_atencion_id_atencion_fkey FOREIGN KEY (id_atencion) REFERENCES public.atenciones(id_atencion) DEFERRABLE;


--
-- Name: historial_estados_atencion historial_estados_atencion_registrado_por_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.historial_estados_atencion
    ADD CONSTRAINT historial_estados_atencion_registrado_por_fkey FOREIGN KEY (registrado_por) REFERENCES public.usuarios(id_usuario) DEFERRABLE;


--
-- Name: horarios_atencion horarios_atencion_id_espacio_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.horarios_atencion
    ADD CONSTRAINT horarios_atencion_id_espacio_fkey FOREIGN KEY (id_espacio) REFERENCES public.espacios_lavado(id_espacio) DEFERRABLE;


--
-- Name: pagos pagos_id_atencion_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pagos
    ADD CONSTRAINT pagos_id_atencion_fkey FOREIGN KEY (id_atencion) REFERENCES public.atenciones(id_atencion) DEFERRABLE;


--
-- Name: pagos pagos_id_pedido_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pagos
    ADD CONSTRAINT pagos_id_pedido_fkey FOREIGN KEY (id_pedido) REFERENCES public.pedidos(id_pedido) DEFERRABLE;


--
-- Name: pagos pagos_registrado_por_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pagos
    ADD CONSTRAINT pagos_registrado_por_fkey FOREIGN KEY (registrado_por) REFERENCES public.usuarios(id_usuario) DEFERRABLE;


--
-- Name: pedidos pedidos_id_cliente_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pedidos
    ADD CONSTRAINT pedidos_id_cliente_fkey FOREIGN KEY (id_cliente) REFERENCES public.usuarios(id_usuario) DEFERRABLE;


--
-- Name: productos productos_id_categoria_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.productos
    ADD CONSTRAINT productos_id_categoria_fkey FOREIGN KEY (id_categoria) REFERENCES public.categorias_producto(id_categoria) DEFERRABLE;


--
-- Name: reclamos reclamos_id_atencion_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.reclamos
    ADD CONSTRAINT reclamos_id_atencion_fkey FOREIGN KEY (id_atencion) REFERENCES public.atenciones(id_atencion) DEFERRABLE;


--
-- Name: reclamos reclamos_id_cliente_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.reclamos
    ADD CONSTRAINT reclamos_id_cliente_fkey FOREIGN KEY (id_cliente) REFERENCES public.usuarios(id_usuario) DEFERRABLE;


--
-- Name: reclamos reclamos_id_pago_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.reclamos
    ADD CONSTRAINT reclamos_id_pago_fkey FOREIGN KEY (id_pago) REFERENCES public.pagos(id_pago) DEFERRABLE;


--
-- Name: reclamos reclamos_id_pedido_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.reclamos
    ADD CONSTRAINT reclamos_id_pedido_fkey FOREIGN KEY (id_pedido) REFERENCES public.pedidos(id_pedido) DEFERRABLE;


--
-- Name: reclamos reclamos_respondido_por_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.reclamos
    ADD CONSTRAINT reclamos_respondido_por_fkey FOREIGN KEY (respondido_por) REFERENCES public.usuarios(id_usuario) DEFERRABLE;


--
-- Name: reserva_servicios reserva_servicios_id_reserva_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.reserva_servicios
    ADD CONSTRAINT reserva_servicios_id_reserva_fkey FOREIGN KEY (id_reserva) REFERENCES public.reservas(id_reserva) DEFERRABLE;


--
-- Name: reserva_servicios reserva_servicios_id_servicio_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.reserva_servicios
    ADD CONSTRAINT reserva_servicios_id_servicio_fkey FOREIGN KEY (id_servicio) REFERENCES public.servicios(id_servicio) DEFERRABLE;


--
-- Name: reservas reservas_id_cliente_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.reservas
    ADD CONSTRAINT reservas_id_cliente_fkey FOREIGN KEY (id_cliente) REFERENCES public.usuarios(id_usuario) DEFERRABLE;


--
-- Name: reservas reservas_id_espacio_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.reservas
    ADD CONSTRAINT reservas_id_espacio_fkey FOREIGN KEY (id_espacio) REFERENCES public.espacios_lavado(id_espacio) DEFERRABLE;


--
-- Name: reservas reservas_id_trabajador_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.reservas
    ADD CONSTRAINT reservas_id_trabajador_fkey FOREIGN KEY (id_trabajador) REFERENCES public.usuarios(id_usuario) DEFERRABLE;


--
-- Name: reservas reservas_id_vehiculo_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.reservas
    ADD CONSTRAINT reservas_id_vehiculo_fkey FOREIGN KEY (id_vehiculo) REFERENCES public.vehiculos(id_vehiculo) DEFERRABLE;


--
-- Name: servicios servicios_id_categoria_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.servicios
    ADD CONSTRAINT servicios_id_categoria_fkey FOREIGN KEY (id_categoria) REFERENCES public.categoria_servicios(id_categoria_servicio) DEFERRABLE;


--
-- Name: trabajadores trabajadores_id_usuario_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trabajadores
    ADD CONSTRAINT trabajadores_id_usuario_fkey FOREIGN KEY (id_usuario) REFERENCES public.usuarios(id_usuario) DEFERRABLE;


--
-- Name: usuarios usuarios_id_rol_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.usuarios
    ADD CONSTRAINT usuarios_id_rol_fkey FOREIGN KEY (id_rol) REFERENCES public.rol(id) DEFERRABLE;


--
-- Name: usuarios usuarios_id_sede_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.usuarios
    ADD CONSTRAINT usuarios_id_sede_fkey FOREIGN KEY (id_sede) REFERENCES public.sede(id) DEFERRABLE;


--
-- Name: vehiculos vehiculos_id_cliente_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vehiculos
    ADD CONSTRAINT vehiculos_id_cliente_fkey FOREIGN KEY (id_cliente) REFERENCES public.usuarios(id_usuario) DEFERRABLE;


--
-- PostgreSQL database dump complete
--

\unrestrict IVOxqDO1Ipz63jBlaiXa16TjsD21Uklk6pYMX319fqLrKgFTHPyVJCm3sIN60eX

