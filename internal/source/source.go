// Package source define la interfaz de una fuente de correos (Outlook real o demo).
package source

import (
	"context"
	"errors"

	"outlook_descargas_adjuntos/internal/domain"
)

// ScanItem es un elemento entregado por Scan. Err != nil indica que no se pudo leer.
type ScanItem struct {
	Mail domain.Mail
	Err  error
}

// ErrStopScan lo devuelve la función de Scan para cortar el recorrido sin error.
var ErrStopScan = errors.New("stop scan")

// MailSource es lo único que el pipeline de extracción conoce de la fuente de correos.
type MailSource interface {
	// Name devuelve "demo" u "outlook".
	Name() string
	ListRoots(ctx context.Context) ([]domain.Folder, error)
	ListChildren(ctx context.Context, ref domain.FolderRef) ([]domain.Folder, error)
	// Scan entrega los elementos de la carpeta (sin subcarpetas) por ReceivedAt descendente.
	Scan(ctx context.Context, ref domain.FolderRef, fn func(ScanItem) error) error
	// SaveAttachment guarda el adjunto index (base 1) del correo en destPath.
	SaveAttachment(ctx context.Context, mailEntryID string, index int, destPath string) error
}
