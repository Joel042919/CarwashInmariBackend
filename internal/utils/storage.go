package utils

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

// GuardarArchivo sube el archivo a R2 y, si R2 no está disponible o falla,
// lo guarda en ./uploads/<key>. Devuelve la ruta/URL que se persiste en BD.
func GuardarArchivo(ctx context.Context, r2 *R2Client, file io.ReadSeeker, key, contentType string) (string, error) {
	if r2 != nil {
		if url, err := r2.SubirArchivo(ctx, file, key, contentType); err == nil {
			return url, nil
		}
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return "", err
		}
	}

	localPath := filepath.Join(".", "uploads", filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return "", err
	}
	dst, err := os.Create(localPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, file); err != nil {
		return "", err
	}
	return "/uploads/" + key, nil
}
