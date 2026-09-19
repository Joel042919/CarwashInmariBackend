package utils

import (
	"fmt"
	"strings"
	"time"
)

// ParseHM convierte "HH:MM" (o "HH:MM:SS") en minutos desde las 00:00.
func ParseHM(s string) (int, error) {
	s = strings.TrimSpace(s)
	if len(s) >= 5 {
		s = s[:5]
	}
	t, err := time.Parse("15:04", s)
	if err != nil {
		return 0, fmt.Errorf("hora inválida %q (use HH:MM)", s)
	}
	return t.Hour()*60 + t.Minute(), nil
}

// FormatHM convierte minutos desde las 00:00 en "HH:MM".
func FormatHM(min int) string {
	return fmt.Sprintf("%02d:%02d", min/60, min%60)
}

// ParseFecha interpreta "AAAA-MM-DD" en la zona horaria local del servidor.
func ParseFecha(s string) (time.Time, error) {
	t, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(s), time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("fecha inválida %q (use AAAA-MM-DD)", s)
	}
	return t, nil
}
