// Package extract orquesta una descarga: escanear → filtrar → descargar → reportar.
// Solo conoce la interfaz source.MailSource, nunca una implementación concreta.
package extract

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"outlook_descargas_adjuntos/internal/domain"
	"outlook_descargas_adjuntos/internal/fsutil"
	"outlook_descargas_adjuntos/internal/report"
	"outlook_descargas_adjuntos/internal/source"
)

// Phase es la fase actual del job.
type Phase string

const (
	PhaseConnecting  Phase = "connecting"
	PhaseScanning    Phase = "scanning"
	PhaseDownloading Phase = "downloading"
	PhaseReporting   Phase = "reporting"
)

// writeReport es una variable para poder simular un fallo de escritura en los tests.
var writeReport = report.WriteExcel

// earlyExitAfter es cuántos elementos consecutivos anteriores al rango cortan el escaneo de una carpeta.
const earlyExitAfter = 100

// Progress es una foto de los contadores del job.
type Progress struct {
	Phase      Phase `json:"phase"`
	Scanned    int   `json:"scanned"`
	InRange    int   `json:"inRange"`
	MailsDone  int   `json:"mailsDone"`
	MailsTotal int   `json:"mailsTotal"`
	Saved      int   `json:"saved"`
	Skipped    int   `json:"skipped"`
	Errors     int   `json:"errors"`
	Bytes      int64 `json:"bytes"`
	ElapsedMs  int64 `json:"elapsedMs"`
}

// ActivityKind clasifica una línea del registro de actividad.
type ActivityKind string

const (
	ActSaved   ActivityKind = "saved"
	ActRenamed ActivityKind = "renamed"
	ActSkipped ActivityKind = "skipped"
	ActError   ActivityKind = "error"
)

// Activity es una línea del registro de actividad en vivo.
type Activity struct {
	At        time.Time    `json:"at"`
	Kind      ActivityKind `json:"kind"`
	Name      string       `json:"name"`
	Subfolder string       `json:"subfolder"`
	Reason    string       `json:"reason"`
	Size      int64        `json:"size"`
}

// Sink recibe el progreso y la actividad. Limitar la frecuencia es responsabilidad del Sink.
type Sink interface {
	Progress(Progress)
	Activity(Activity)
}

// Result es el resultado de Run, también cuando se cancela.
type Result struct {
	Summary    domain.Summary     `json:"summary"`
	Records    []domain.RunRecord `json:"records"`
	ReportPath string             `json:"reportPath"`
}

type candidate struct {
	mail       domain.Mail
	folderPath string
	eligible   []domain.Attachment
}

type runner struct {
	src   source.MailSource
	req   domain.JobRequest
	sink  Sink
	now   func() time.Time
	sum   domain.Summary
	recs  []domain.RunRecord
	prog  Progress
	start time.Time
	end   time.Time
	cands []candidate
}

// Run ejecuta la descarga completa. Si ctx se cancela, escribe el reporte parcial y devuelve ctx.Err().
func Run(ctx context.Context, src source.MailSource, req domain.JobRequest, sink Sink, now func() time.Time) (Result, error) {
	if err := req.Validate(); err != nil {
		return Result{}, err
	}
	r := &runner{src: src, req: req, sink: sink, now: now}
	r.sum = domain.Summary{Started: now(), ByExt: map[string]int{}, Reasons: map[string]int{},
		ErrorReasons: map[string]int{}, BySubfolder: map[string]int{}}
	r.start, r.end = dayBounds(req.Filters)
	r.emit(PhaseConnecting)

	r.prog.Phase = PhaseScanning
	r.scanFolder(ctx, req.Folder)

	if ctx.Err() == nil {
		r.download(ctx)
	} else {
		r.recordPending(0, 0)
	}
	if ctx.Err() != nil {
		r.sum.Cancelled = true
	}

	r.emit(PhaseReporting)
	r.sum.Finished = now()
	reportPath, err := fsutil.UniquePath(req.DestDir, report.FileName(r.sum.Finished))
	if err == nil {
		err = writeReport(reportPath, req, r.recs, r.sum)
	}
	res := Result{Summary: r.sum, Records: r.recs, ReportPath: reportPath}
	if err != nil {
		// Sin reporte válido no se ofrece abrirlo; se borra lo que haya quedado a medias.
		if reportPath != "" {
			_ = os.Remove(reportPath)
		}
		res.ReportPath = ""
		return res, fmt.Errorf("escribir reporte: %w", err)
	}
	return res, ctx.Err()
}

func (r *runner) emit(p Phase) {
	r.prog.Phase = p
	r.prog.Scanned = r.sum.Scanned
	r.prog.InRange = r.sum.InRange
	r.prog.Saved = r.sum.Saved
	r.prog.Skipped = r.sum.Skipped
	r.prog.Errors = r.sum.Errors
	r.prog.Bytes = r.sum.Bytes
	r.prog.ElapsedMs = r.now().Sub(r.sum.Started).Milliseconds()
	r.sink.Progress(r.prog)
}

func (r *runner) record(rec domain.RunRecord) {
	r.recs = append(r.recs, rec)
	switch rec.Status {
	case domain.StatusSkipped:
		r.sum.Skipped++
		r.sum.Reasons[rec.Reason]++
	case domain.StatusError:
		r.sum.Errors++
		r.sum.ErrorReasons[rec.Reason]++
	}
}

func mailRecord(status domain.RecordStatus, reason, folder string, m domain.Mail) domain.RunRecord {
	return domain.RunRecord{Status: status, Reason: reason, FolderPath: folder, MailEntryID: m.EntryID,
		Subject: m.Subject, Sender: m.Sender, ReceivedAt: m.ReceivedAt}
}

// folderError registra que una carpeta no se pudo leer; el job sigue con las demás.
func (r *runner) folderError(ref domain.FolderRef, err error) {
	r.record(domain.RunRecord{Status: domain.StatusError, Reason: domain.ReasonFolderError, FolderPath: ref.Path,
		Subject: err.Error()})
	r.sink.Activity(Activity{At: r.now(), Kind: ActError, Name: ref.Path, Reason: domain.ReasonFolderError})
}

// scanFolder recorre la carpeta y, si se pidió, sus subcarpetas en profundidad.
// Un error de carpeta se registra y no detiene el recorrido; solo la cancelación lo corta.
func (r *runner) scanFolder(ctx context.Context, ref domain.FolderRef) {
	before := 0
	err := r.src.Scan(ctx, ref, func(it source.ScanItem) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		r.sum.Scanned++
		defer r.emit(PhaseScanning)
		if it.Err != nil {
			r.sum.InRange++
			r.record(mailRecord(domain.StatusError, domain.ReasonReadError, ref.Path, it.Mail))
			r.sink.Activity(Activity{At: r.now(), Kind: ActError, Name: it.Mail.Subject, Reason: domain.ReasonReadError})
			return nil
		}
		switch checkRange(it.Mail.ReceivedAt, r.start, r.end) {
		case afterRange:
			r.sum.OutOfRange++
			return nil
		case beforeRange:
			r.sum.OutOfRange++
			before++
			if before >= earlyExitAfter {
				return source.ErrStopScan
			}
			return nil
		}
		before = 0
		r.sum.InRange++
		eligible, reason := evaluate(it.Mail, r.req.Filters)
		if reason != "" {
			r.record(mailRecord(domain.StatusSkipped, reason, ref.Path, it.Mail))
			r.sink.Activity(Activity{At: r.now(), Kind: ActSkipped, Name: it.Mail.Subject, Reason: reason})
			return nil
		}
		r.cands = append(r.cands, candidate{mail: it.Mail, folderPath: ref.Path, eligible: eligible})
		return nil
	})
	if ctx.Err() != nil {
		return
	}
	if err != nil {
		r.folderError(ref, err)
	}
	if !r.req.Filters.IncludeSubfolders {
		return
	}
	kids, err := r.src.ListChildren(ctx, ref)
	if ctx.Err() != nil {
		return
	}
	if err != nil {
		r.folderError(ref, err)
		return
	}
	for _, k := range kids {
		r.scanFolder(ctx, k.Ref)
		if ctx.Err() != nil {
			return
		}
	}
}

func (r *runner) download(ctx context.Context) {
	r.prog.MailsTotal = len(r.cands)
	r.emit(PhaseDownloading)
	for ci, c := range r.cands {
		savedAny := false
		for ai, a := range c.eligible {
			saved, interrupted := false, ctx.Err() != nil
			if !interrupted {
				saved, interrupted = r.saveOne(ctx, c, a)
			}
			if interrupted {
				if savedAny {
					r.sum.MailsWithDownloads++
				}
				r.recordPending(ci, ai)
				return
			}
			savedAny = savedAny || saved
			r.emit(PhaseDownloading)
		}
		if savedAny {
			r.sum.MailsWithDownloads++
		}
		r.prog.MailsDone++
		r.emit(PhaseDownloading)
	}
}

// recordPending registra como omitidos por cancelación los adjuntos que no se llegaron a descargar,
// desde el adjunto ai del candidato ci en adelante.
func (r *runner) recordPending(ci, ai int) {
	for ; ci < len(r.cands); ci++ {
		c := r.cands[ci]
		for ; ai < len(c.eligible); ai++ {
			rec := mailRecord(domain.StatusSkipped, domain.ReasonCancelled, c.folderPath, c.mail)
			rec.FileName = c.eligible[ai].FileName
			r.record(rec)
		}
		ai = 0
	}
}

// saveOne guarda un adjunto. saved indica si quedó guardado; interrupted, que se canceló a mitad
// (en ese caso no se registra nada: lo registra recordPending).
func (r *runner) saveOne(ctx context.Context, c candidate, a domain.Attachment) (saved, interrupted bool) {
	subdir := ""
	if r.req.Organize == domain.OrganizeByMonth {
		subdir = c.mail.ReceivedAt.In(time.Local).Format("2006-01")
	}
	dir := filepath.Join(r.req.DestDir, subdir)
	name := fsutil.SanitizeFileName(a.FileName)
	fail := func(err error) (bool, bool) {
		rec := mailRecord(domain.StatusError, err.Error(), c.folderPath, c.mail)
		rec.FileName = a.FileName
		r.record(rec)
		r.sink.Activity(Activity{At: r.now(), Kind: ActError, Name: a.FileName, Subfolder: subdir, Reason: err.Error()})
		return false, false
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fail(err)
	}
	fit, err := fsutil.FitPath(dir, name, fsutil.MaxPath)
	if err != nil {
		return fail(err)
	}
	path, err := fsutil.UniquePath(dir, fit)
	if err != nil {
		return fail(err)
	}
	if err := r.src.SaveAttachment(ctx, c.mail.EntryID, a.Index, path); err != nil {
		os.Remove(path)
		if ctx.Err() != nil {
			return false, true // cancelado: no es un error del adjunto
		}
		return fail(err)
	}
	st, err := os.Stat(path)
	if err != nil {
		return fail(err)
	}
	final := filepath.Base(path)
	rec := mailRecord(domain.StatusSaved, "", c.folderPath, c.mail)
	rec.FileName, rec.SavedPath, rec.Size = final, path, st.Size()
	r.record(rec)
	r.sum.Saved++
	r.sum.Bytes += st.Size()
	r.sum.ByExt[strings.ToLower(filepath.Ext(final))]++
	r.sum.BySubfolder[subdir]++
	kind := ActSaved
	if final != name {
		kind = ActRenamed
	}
	r.sink.Activity(Activity{At: r.now(), Kind: kind, Name: final, Subfolder: subdir, Size: st.Size()})
	return true, false
}
