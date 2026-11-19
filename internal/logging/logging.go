// Package logging configura slog hacia un archivo por sesión en %LOCALAPPDATA%, con retención.
package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	appDir     = "outlook_descargas_adjuntos"
	maxAge     = 30 * 24 * time.Hour
	maxFiles   = 20
	filePrefix = "sesion_"
)

// LogDir devuelve %LOCALAPPDATA%\outlook_descargas_adjuntos\logs.
func LogDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, appDir, "logs"), nil
}

// Setup aplica la retención y abre el log de esta sesión.
func Setup(dir string, now time.Time) (*slog.Logger, io.Closer, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, nil, err
	}
	prune(dir, now)
	name := filepath.Join(dir, filePrefix+now.Format("20060102_150405")+".log")
	f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, nil, err
	}
	return slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelInfo})), f, nil
}

// prune borra los logs de más de 30 días y deja sitio para el nuevo dentro del máximo de 20.
func prune(dir string, now time.Time) {
	files, _ := filepath.Glob(filepath.Join(dir, filePrefix+"*.log"))
	type entry struct {
		path string
		mod  time.Time
	}
	var keep []entry
	for _, p := range files {
		st, err := os.Stat(p)
		if err != nil {
			continue
		}
		if now.Sub(st.ModTime()) > maxAge {
			os.Remove(p)
			continue
		}
		keep = append(keep, entry{p, st.ModTime()})
	}
	sort.Slice(keep, func(i, j int) bool { return keep[i].mod.After(keep[j].mod) })
	for i := maxFiles - 1; i < len(keep); i++ {
		os.Remove(keep[i].path)
	}
}
