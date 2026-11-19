// Package domain contiene los tipos puros de la descarga de adjuntos.
// No importa nada de Wails, COM ni del sistema de archivos salvo en la validación.
package domain

import "time"

// FolderRef identifica una carpeta de origen de forma estable (StoreID + EntryID).
// Path es solo la ruta legible para la UI (`cuenta\carpeta\sub`).
type FolderRef struct {
	StoreID string `json:"storeId"`
	EntryID string `json:"entryId"`
	Path    string `json:"path"`
}

// Folder es un nodo del árbol de carpetas.
type Folder struct {
	Ref         FolderRef `json:"ref"`
	Name        string    `json:"name"`
	ItemCount   int       `json:"itemCount"`
	HasChildren bool      `json:"hasChildren"`
}

// AttachmentKind clasifica el adjunto según su naturaleza en el correo.
type AttachmentKind string

const (
	KindFile        AttachmentKind = "file"
	KindInline      AttachmentKind = "inline"
	KindEmbeddedMsg AttachmentKind = "msg"
	KindOther       AttachmentKind = "other"
)

// Attachment describe un adjunto. Index es base 1, como en COM.
type Attachment struct {
	Index    int            `json:"index"`
	FileName string         `json:"fileName"`
	Size     int64          `json:"size"`
	Kind     AttachmentKind `json:"kind"`
}

// Mail es un elemento de la carpeta. IsMail=false para citas, reportes, etc.
type Mail struct {
	EntryID     string       `json:"entryId"`
	Subject     string       `json:"subject"`
	Sender      string       `json:"sender"`
	ReceivedAt  time.Time    `json:"receivedAt"`
	IsMail      bool         `json:"isMail"`
	Attachments []Attachment `json:"attachments"`
}

// Organize define cómo se reparten los archivos en el destino.
type Organize string

const (
	OrganizeFlat    Organize = "flat"
	OrganizeByMonth Organize = "by_month"
)

// Filters son los criterios de selección. From/To valen solo por su fecha (día) en time.Local.
type Filters struct {
	From              time.Time `json:"from"`
	To                time.Time `json:"to"`
	SubjectContains   string    `json:"subjectContains"`
	SenderContains    string    `json:"senderContains"`
	Extensions        []string  `json:"extensions"`
	SkipInline        bool      `json:"skipInline"`
	IncludeSubfolders bool      `json:"includeSubfolders"`
}

// JobRequest es todo lo necesario para ejecutar una descarga.
type JobRequest struct {
	Folder   FolderRef `json:"folder"`
	Filters  Filters   `json:"filters"`
	DestDir  string    `json:"destDir"`
	Organize Organize  `json:"organize"`
}

// RecordStatus es el resultado de un registro del reporte.
type RecordStatus string

const (
	StatusSaved   RecordStatus = "saved"
	StatusSkipped RecordStatus = "skipped"
	StatusError   RecordStatus = "error"
)

// RunRecord es una fila del reporte: un adjunto guardado, o un correo/adjunto omitido o con error.
type RunRecord struct {
	Status      RecordStatus `json:"status"`
	Reason      string       `json:"reason"`
	FolderPath  string       `json:"folderPath"`
	MailEntryID string       `json:"mailEntryId"`
	Subject     string       `json:"subject"`
	Sender      string       `json:"sender"`
	ReceivedAt  time.Time    `json:"receivedAt"`
	FileName    string       `json:"fileName"`
	SavedPath   string       `json:"savedPath"`
	Size        int64        `json:"size"`
}

// Summary agrega los contadores de una ejecución.
type Summary struct {
	Scanned            int            `json:"scanned"`
	OutOfRange         int            `json:"outOfRange"`
	InRange            int            `json:"inRange"`
	MailsWithDownloads int            `json:"mailsWithDownloads"`
	Saved              int            `json:"saved"`
	Skipped            int            `json:"skipped"`
	Errors             int            `json:"errors"`
	Bytes              int64          `json:"bytes"`
	ByExt              map[string]int `json:"byExt"`
	Reasons            map[string]int `json:"reasons"`
	Started            time.Time      `json:"started"`
	Finished           time.Time      `json:"finished"`
	Cancelled          bool           `json:"cancelled"`
}
