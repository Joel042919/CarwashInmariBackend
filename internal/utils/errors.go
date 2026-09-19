package utils

import (
	"errors"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgconn"
)

// AppError es un error de negocio con el código HTTP que debe devolverse.
type AppError struct {
	Status int
	Msg    string
}

func (e *AppError) Error() string { return e.Msg }

func BadRequest(msg string) error { return &AppError{Status: http.StatusBadRequest, Msg: msg} }
func NotFound(msg string) error   { return &AppError{Status: http.StatusNotFound, Msg: msg} }
func Conflict(msg string) error   { return &AppError{Status: http.StatusConflict, Msg: msg} }

// WriteError responde con el AppError correspondiente; cualquier otro error
// se registra en el log y se devuelve como 500 sin exponer detalles internos.
func WriteError(w http.ResponseWriter, err error) {
	var ae *AppError
	if errors.As(err, &ae) {
		ErrorJSON(w, ae.Status, ae.Msg)
		return
	}
	log.Printf("[ERROR] %v", err)
	ErrorJSON(w, http.StatusInternalServerError, "Error interno del servidor")
}

// IsUniqueViolation indica si err es una violación de UNIQUE en PostgreSQL.
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
