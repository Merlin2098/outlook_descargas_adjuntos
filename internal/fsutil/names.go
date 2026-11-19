// Package fsutil resuelve nombres de archivo seguros en Windows: saneados, recortados y sin sobrescribir.
package fsutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MaxPath es la longitud máxima (en caracteres) de las rutas que generamos.
// Queda por debajo de MAX_PATH (260) para no depender del prefijo `\\?\`.
const MaxPath = 240

// suffixMargin reserva espacio para el sufijo `_NNNN` de UniquePath.
const suffixMargin = 6

// maxUniqueTries es el número de sufijos que prueba UniquePath antes de rendirse.
const maxUniqueTries = 9999

//nolint:staticcheck // ST1005: mensajes visibles para el usuario.
var (
	ErrPathTooLong = errors.New("Ruta de destino demasiado larga")
	ErrNoFreeName  = errors.New("No hay un nombre libre para el archivo")
)

var reserved = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true,
	"COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true,
	"LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

// SanitizeFileName devuelve un nombre de archivo válido en Windows.
func SanitizeFileName(name string) string {
	var b strings.Builder
	for _, r := range name {
		if r < 0x20 || strings.ContainsRune(`<>:"/\|?*`, r) {
			b.WriteRune('_')
			continue
		}
		b.WriteRune(r)
	}
	s := strings.TrimRight(strings.TrimSpace(b.String()), ". ")
	if s == "" {
		return "adjunto"
	}
	stem, _, _ := strings.Cut(s, ".")
	if reserved[strings.ToUpper(stem)] {
		s = "_" + s
	}
	return s
}

// FitPath recorta el nombre (conservando la extensión) para que dir+nombre+sufijo quepan en max.
func FitPath(dir, name string, max int) (string, error) {
	dirLen := len([]rune(dir))
	if dirLen > max-16 {
		return "", ErrPathTooLong
	}
	avail := max - dirLen - 1 - suffixMargin
	if len([]rune(name)) <= avail {
		return name, nil
	}
	ext := filepath.Ext(name)
	if len([]rune(ext)) > 16 {
		ext = ""
	}
	stem := []rune(strings.TrimSuffix(name, ext))
	keep := avail - len([]rune(ext))
	if keep < 1 {
		return "", ErrPathTooLong
	}
	if len(stem) > keep {
		stem = stem[:keep]
	}
	return strings.TrimRight(string(stem), ". ") + ext, nil
}

// UniquePath devuelve una ruta en dir que no existe todavía: name, name_1, name_2…
func UniquePath(dir, name string) (string, error) {
	p := filepath.Join(dir, name)
	if !exists(p) {
		return p, nil
	}
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for i := 1; i <= maxUniqueTries; i++ {
		p = filepath.Join(dir, fmt.Sprintf("%s_%d%s", stem, i, ext))
		if !exists(p) {
			return p, nil
		}
	}
	return "", ErrNoFreeName
}

func exists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}
