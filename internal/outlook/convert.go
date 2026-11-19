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

// oleDateToLocal convierte una fecha OLE (días desde 1899-12-30 + fracción del día, como hora de
// pared local) a time.Local, con precisión de milisegundos. go-ole devuelve esa hora etiquetada como
// UTC y sin milisegundos, por eso no se usa VARIANT.Value().
func oleDateToLocal(days float64) time.Time { return oleDateIn(days, time.Local) }

// oleDateIn es oleDateToLocal con la zona como parámetro (para probar cambios de horario).
// El día se calcula en UTC (sin horarios de verano) y la hora se arma como hora de reloj con
// time.Date, así un día de cambio de horario no desplaza ni la hora ni el día.
func oleDateIn(days float64, loc *time.Location) time.Time {
	whole := math.Floor(days)
	ms := int64(math.Round((days - whole) * 24 * 60 * 60 * 1000))
	if ms >= 24*60*60*1000 { // el redondeo llegó a medianoche del día siguiente
		whole++
		ms = 0
	}
	d := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC).AddDate(0, 0, int(whole))
	h := ms / 3_600_000
	mi := ms % 3_600_000 / 60_000
	sec := ms % 60_000 / 1000
	ns := ms % 1000 * int64(time.Millisecond)
	return time.Date(d.Year(), d.Month(), d.Day(), int(h), int(mi), int(sec), int(ns), loc)
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

// senderAddress elige la dirección a mostrar. En Exchange (SenderEmailType "EX") SenderEmailAddress
// es un DN X.500 (/O=…/CN=…); si se conoce la SMTP, se usa esa para que el filtro por dirección funcione.
func senderAddress(emailType, address, smtp string) string {
	if strings.EqualFold(emailType, "EX") && smtp != "" {
		return smtp
	}
	return address
}

// folderPath quita las barras iniciales de Folder.FolderPath ("\cuenta\Bandeja").
func folderPath(p string) string { return strings.TrimLeft(p, `\`) }

// connectError envuelve un fallo de conexión con el mensaje visible.
func connectError(err error) error {
	return fmt.Errorf("%w (%v)", domain.ErrOutlookUnavailable, err)
}
