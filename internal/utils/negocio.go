package utils

import (
	"time"
	_ "time/tzdata"
)

var ZonaNegocio = func() *time.Location {
	zona, err := time.LoadLocation("America/Lima")
	if err != nil {
		panic(err)
	}
	return zona
}()

// RangoFechas representa días del negocio con límite final exclusivo.
type RangoFechas struct{ Desde, Hasta *time.Time }

func ParseRangoFechas(desde, hasta string) (RangoFechas, error) {
	var rango RangoFechas
	for _, campo := range []struct {
		texto   string
		destino **time.Time
	}{{desde, &rango.Desde}, {hasta, &rango.Hasta}} {
		if campo.texto == "" {
			continue
		}
		fecha, err := time.ParseInLocation("2006-01-02", campo.texto, ZonaNegocio)
		if err != nil {
			return rango, BadRequest("las fechas deben tener formato AAAA-MM-DD")
		}
		*campo.destino = &fecha
	}
	if rango.Desde != nil && rango.Hasta != nil && rango.Desde.After(*rango.Hasta) {
		return rango, BadRequest("desde no puede ser posterior a hasta")
	}
	if rango.Hasta != nil {
		fin := rango.Hasta.AddDate(0, 0, 1)
		rango.Hasta = &fin
	}
	return rango, nil
}
