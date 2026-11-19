package domain

import "errors"

// Motivos de omisión o error. Son texto visible: van al reporte y a la UI tal cual.
const (
	ReasonNotMail         = "No es un correo"
	ReasonNoAttachments   = "Sin adjuntos"
	ReasonOnlyInline      = "Solo imágenes inline"
	ReasonNoMatchingExt   = "Sin adjuntos con las extensiones elegidas"
	ReasonUnsupportedKind = "Tipo de adjunto no soportado"
	ReasonSubject         = "El asunto no coincide"
	ReasonSender          = "El remitente no coincide"
	ReasonReadError       = "No se pudo leer el elemento"
	ReasonFolderError     = "No se pudo leer la carpeta"
	ReasonCancelled       = "Cancelado antes de descargar"
)

// Errores de validación (mensajes en español porque llegan a la UI).
//
//nolint:staticcheck // ST1005: mensajes visibles para el usuario.
var (
	ErrNoFolder        = errors.New("Debe seleccionar una carpeta de Outlook")
	ErrNoDest          = errors.New("Debe seleccionar una carpeta de destino")
	ErrDestIsFile      = errors.New("El destino es un archivo, no una carpeta")
	ErrDestNotWritable = errors.New("No se puede escribir en la carpeta de destino")
	ErrBadRange        = errors.New("La fecha de inicio es posterior a la fecha fin")
	ErrDestTooLong     = errors.New("La ruta de la carpeta de destino es demasiado larga")
)
