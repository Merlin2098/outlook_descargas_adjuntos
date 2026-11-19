package main

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"outlook_descargas_adjuntos/internal/domain"
	"outlook_descargas_adjuntos/internal/source"
	"outlook_descargas_adjuntos/internal/source/demo"
)

const dateLayout = "2006-01-02"

// JobRequestDTO es la solicitud tal como llega desde el frontend (fechas como texto).
type JobRequestDTO struct {
	Folder            domain.FolderRef `json:"folder"`
	From              string           `json:"from"`
	To                string           `json:"to"`
	SubjectContains   string           `json:"subjectContains"`
	SenderContains    string           `json:"senderContains"`
	Extensions        []string         `json:"extensions"`
	SkipInline        bool             `json:"skipInline"`
	IncludeSubfolders bool             `json:"includeSubfolders"`
	DestDir           string           `json:"destDir"`
	Organize          string           `json:"organize"`
}

// SourceInfo dice a la UI qué fuente de correos está activa.
type SourceInfo struct {
	Name string `json:"name"`
	Demo bool   `json:"demo"`
}

func parseDate(s string) (time.Time, error) {
	t, err := time.ParseInLocation(dateLayout, s, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("Fecha inválida: %s", s) //nolint:staticcheck // texto visible
	}
	return t, nil
}

// ToDomain convierte el DTO en una solicitud de dominio.
func (d JobRequestDTO) ToDomain() (domain.JobRequest, error) {
	from, err := parseDate(d.From)
	if err != nil {
		return domain.JobRequest{}, err
	}
	to, err := parseDate(d.To)
	if err != nil {
		return domain.JobRequest{}, err
	}
	org := domain.Organize(d.Organize)
	if org != domain.OrganizeByMonth {
		org = domain.OrganizeFlat
	}
	return domain.JobRequest{
		Folder: d.Folder,
		Filters: domain.Filters{
			From:              from,
			To:                to,
			SubjectContains:   d.SubjectContains,
			SenderContains:    d.SenderContains,
			Extensions:        domain.NormalizeExtensions(d.Extensions),
			SkipInline:        d.SkipInline,
			IncludeSubfolders: d.IncludeSubfolders,
		},
		DestDir:  d.DestDir,
		Organize: org,
	}, nil
}

// errNoOutlook se devuelve mientras no exista el adaptador COM (Plan 3).
var errNoOutlook = errors.New("La fuente Outlook llega en el Plan 3; ejecute con --demo") //nolint:staticcheck // texto visible

// Latencia simulada del modo demo, pensada para que el progreso se vea en el video.
const (
	demoScanDelay = 10 * time.Millisecond
	demoSaveDelay = 90 * time.Millisecond
	demoSeed      = 42
)

// selectSource elige la fuente de correos según los argumentos de la línea de comandos.
func selectSource(args []string) (source.MailSource, error) {
	if slices.Contains(args, "--demo") {
		return demo.New(demo.DefaultFixture(time.Now(), demoSeed),
			demo.Options{ScanDelay: demoScanDelay, SaveDelay: demoSaveDelay}), nil
	}
	return nil, errNoOutlook
}
