//go:build windows

// Package outlook implementa source.MailSource sobre Outlook Classic vía COM (go-ole).
// Todo el acceso a COM ocurre en un único goroutine STA (ver sta.go).
package outlook

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"outlook_descargas_adjuntos/internal/domain"
)

// olMail es OlObjectClass.olMail: solo estos items son correos.
const olMail = 43

// Valores de Attachment.Type (OlAttachmentType).
const (
	olByValue      = 1
	olByReference  = 4
	olEmbeddedItem = 5
	olOLE          = 6
)

func isMailClass(class int) bool { return class == olMail }

// oleEpoch es el día 0 de las fechas OLE Automation.
var oleEpoch = time.Date(1899, 12, 30, 0, 0, 0, 0, time.Local)

// oleDateToLocal convierte una fecha OLE (días desde 1899-12-30, hora de pared local) a time.Local,
// con precisión de milisegundos. Se suman días de calendario y luego la hora, para que un cambio
// de horario no desplace el día. go-ole devuelve esa hora etiquetada como UTC y sin milisegundos.
func oleDateToLocal(days float64) time.Time {
	whole := math.Floor(days)
	frac := days - whole
	d := oleEpoch.AddDate(0, 0, int(whole))
	ms := math.Round(frac * 24 * 60 * 60 * 1000)
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.Local).Add(time.Duration(ms) * time.Millisecond)
}

// classifyAttachment decide la naturaleza de un adjunto. Un adjunto por valor es inline si está
// oculto (PR_ATTACHMENT_HIDDEN) o si su Content-ID se usa como cid: en el HTML del correo.
func classifyAttachment(attType int, hidden bool, contentID string, htmlHasCID bool) domain.AttachmentKind {
	switch attType {
	case olByValue:
		if hidden || (contentID != "" && htmlHasCID) {
			return domain.KindInline
		}
		return domain.KindFile
	case olEmbeddedItem:
		return domain.KindEmbeddedMsg
	default:
		return domain.KindOther
	}
}

// attachmentFileName da un nombre utilizable: los .msg embebidos llevan extensión .msg y los
// adjuntos sin nombre se llaman adjunto_N.
func attachmentFileName(name string, kind domain.AttachmentKind, index int) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "adjunto_" + strconv.Itoa(index)
	}
	if kind == domain.KindEmbeddedMsg && !strings.HasSuffix(strings.ToLower(name), ".msg") {
		name += ".msg"
	}
	return name
}

// senderText combina nombre y dirección para filtrar y reportar: "Nombre <dirección>".
func senderText(name, address string) string {
	name, address = strings.TrimSpace(name), strings.TrimSpace(address)
	switch {
	case name == "" || strings.EqualFold(name, address):
		return address
	case address == "":
		return name
	default:
		return name + " <" + address + ">"
	}
}

// folderPath quita las barras iniciales de Folder.FolderPath ("\cuenta\Bandeja").
func folderPath(p string) string { return strings.TrimLeft(p, `\`) }

// connectError envuelve un fallo de conexión con el mensaje visible.
func connectError(err error) error {
	return fmt.Errorf("%w (%v)", domain.ErrOutlookUnavailable, err)
}
