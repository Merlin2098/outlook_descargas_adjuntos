package extract

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"outlook_descargas_adjuntos/internal/domain"
	"outlook_descargas_adjuntos/internal/source"
	"outlook_descargas_adjuntos/internal/source/demo"
)

var testNow = time.Date(2026, 11, 12, 10, 0, 0, 0, time.Local)

func fixedNow() time.Time { return testNow }

type captureSink struct {
	mu         sync.Mutex
	progress   []Progress
	activity   []Activity
	onActivity func(Activity, int)
}

func (c *captureSink) Progress(p Progress) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.progress = append(c.progress, p)
}

func (c *captureSink) Activity(a Activity) {
	c.mu.Lock()
	c.activity = append(c.activity, a)
	n := len(c.activity)
	cb := c.onActivity
	c.mu.Unlock()
	if cb != nil {
		cb(a, n)
	}
}

func demoSource() *demo.Source { return demo.New(demo.DefaultFixture(testNow, 42), demo.Options{}) }

func folderRef(t *testing.T, src *demo.Source, name string) domain.FolderRef {
	t.Helper()
	roots, _ := src.ListRoots(context.Background())
	kids, _ := src.ListChildren(context.Background(), roots[0].Ref)
	for _, k := range kids {
		if k.Name == name {
			return k.Ref
		}
	}
	t.Fatalf("carpeta %q no encontrada", name)
	return domain.FolderRef{}
}

func baseReq(t *testing.T, ref domain.FolderRef) domain.JobRequest {
	return domain.JobRequest{
		Folder:   ref,
		Filters:  domain.Filters{From: testNow.AddDate(0, 0, -121), To: testNow, SkipInline: true},
		DestDir:  t.TempDir(),
		Organize: domain.OrganizeFlat,
	}
}

func countReason(recs []domain.RunRecord, reason string) int {
	n := 0
	for _, r := range recs {
		if r.Reason == reason {
			n++
		}
	}
	return n
}

func TestRunFacturasProveedores(t *testing.T) {
	src := demoSource()
	req := baseReq(t, folderRef(t, src, "Facturas proveedores"))
	sink := &captureSink{}
	res, err := Run(context.Background(), src, req, sink, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	s := res.Summary
	if s.InRange != s.MailsWithDownloads+s.Skipped+1 {
		t.Fatalf("InRange %d != MailsWithDownloads %d + Skipped %d + 1", s.InRange, s.MailsWithDownloads, s.Skipped)
	}
	if s.Errors != 3 {
		t.Fatalf("Errors = %d, want 3", s.Errors)
	}
	for _, reason := range []string{domain.ReasonNotMail, domain.ReasonOnlyInline, domain.ReasonReadError} {
		if countReason(res.Records, reason) == 0 {
			t.Errorf("falta un registro con motivo %q", reason)
		}
	}
	saveErrs := 0
	for _, r := range res.Records {
		if r.Status == domain.StatusError && r.Reason != domain.ReasonReadError {
			saveErrs++
		}
		if r.Status == domain.StatusSaved {
			if _, err := os.Stat(r.SavedPath); err != nil {
				t.Fatalf("archivo guardado inexistente: %v", err)
			}
		}
	}
	if saveErrs != 2 {
		t.Fatalf("errores de guardado = %d, want 2", saveErrs)
	}
	for _, n := range []string{"factura_F001-00223.pdf", "factura_F001-00223_1.pdf"} {
		if _, err := os.Stat(filepath.Join(req.DestDir, n)); err != nil {
			t.Fatalf("falta %s: %v", n, err)
		}
	}
	renamed := false
	for _, a := range sink.activity {
		if a.Kind == ActRenamed {
			renamed = true
		}
	}
	if !renamed {
		t.Fatal("no hubo ninguna actividad ActRenamed")
	}
	if _, err := os.Stat(res.ReportPath); err != nil {
		t.Fatalf("reporte inexistente: %v", err)
	}
}

func TestRunByMonth(t *testing.T) {
	src := demoSource()
	req := baseReq(t, folderRef(t, src, "Facturas proveedores"))
	req.Organize = domain.OrganizeByMonth
	res, err := Run(context.Background(), src, req, &captureSink{}, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	month := regexp.MustCompile(`^\d{4}-\d{2}$`)
	for _, r := range res.Records {
		if r.Status == domain.StatusSaved && !month.MatchString(filepath.Base(filepath.Dir(r.SavedPath))) {
			t.Fatalf("subcarpeta inesperada: %s", r.SavedPath)
		}
	}
}

func TestRunExtensionsFilter(t *testing.T) {
	src := demoSource()
	req := baseReq(t, folderRef(t, src, "Facturas proveedores"))
	req.Filters.Extensions = []string{".xml"}
	res, err := Run(context.Background(), src, req, &captureSink{}, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if res.Summary.Saved == 0 {
		t.Fatal("no se guardó nada")
	}
	for _, r := range res.Records {
		if r.Status == domain.StatusSaved && !strings.HasSuffix(r.SavedPath, ".xml") {
			t.Fatalf("guardado fuera del filtro: %s", r.SavedPath)
		}
	}
}

func TestRunExistingFiles(t *testing.T) {
	src := demoSource()
	req := baseReq(t, folderRef(t, src, "Facturas proveedores"))
	prev := filepath.Join(req.DestDir, "factura_F001-00218.pdf")
	if err := os.WriteFile(prev, []byte("previo"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(context.Background(), src, req, &captureSink{}, fixedNow); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(prev); string(b) != "previo" {
		t.Fatal("se sobrescribió un archivo existente")
	}
}

func TestRunCancel(t *testing.T) {
	src := demoSource()
	req := baseReq(t, folderRef(t, src, "Facturas proveedores"))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	saved := 0
	sink := &captureSink{onActivity: func(a Activity, _ int) {
		if a.Kind == ActSaved || a.Kind == ActRenamed {
			saved++
			if saved == 5 {
				cancel()
			}
		}
	}}
	res, err := Run(ctx, src, req, sink, fixedNow)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if !res.Summary.Cancelled || res.Summary.Saved < 5 || res.Summary.Saved > 6 {
		t.Fatalf("Cancelled=%v Saved=%d", res.Summary.Cancelled, res.Summary.Saved)
	}
	if _, err := os.Stat(res.ReportPath); err != nil {
		t.Fatalf("reporte parcial inexistente: %v", err)
	}
}

func TestRunIncludeSubfolders(t *testing.T) {
	src := demoSource()
	req := baseReq(t, folderRef(t, src, "Proyectos"))
	res, err := Run(context.Background(), src, req, &captureSink{}, fixedNow)
	if err != nil || res.Summary.Scanned != 0 {
		t.Fatalf("sin subcarpetas: Scanned=%d err=%v", res.Summary.Scanned, err)
	}
	req.DestDir = t.TempDir()
	req.Filters.IncludeSubfolders = true
	res, err = Run(context.Background(), src, req, &captureSink{}, fixedNow)
	if err != nil || res.Summary.Scanned != 127 {
		t.Fatalf("con subcarpetas: Scanned=%d err=%v", res.Summary.Scanned, err)
	}
}

func TestRunInvalidDest(t *testing.T) {
	src := demoSource()
	req := baseReq(t, folderRef(t, src, "Facturas proveedores"))
	file := filepath.Join(req.DestDir, "f.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	req.DestDir = file
	sink := &captureSink{}
	if _, err := Run(context.Background(), src, req, sink, fixedNow); !errors.Is(err, domain.ErrDestIsFile) {
		t.Fatalf("err = %v", err)
	}
	for _, p := range sink.progress {
		if p.Phase == PhaseScanning {
			t.Fatal("escaneó con un destino inválido")
		}
	}
}

// failingSource falla al escanear la carpeta failPath y al listar los hijos de listFailPath.
type failingSource struct {
	*demo.Source
	failPath     string
	listFailPath string
}

func (f failingSource) Scan(ctx context.Context, ref domain.FolderRef, fn func(source.ScanItem) error) error {
	if ref.EntryID == f.failPath {
		return errors.New("acceso denegado (simulado)")
	}
	return f.Source.Scan(ctx, ref, fn)
}

func (f failingSource) ListChildren(ctx context.Context, ref domain.FolderRef) ([]domain.Folder, error) {
	if ref.EntryID == f.listFailPath {
		return nil, errors.New("no se pueden listar subcarpetas (simulado)")
	}
	return f.Source.ListChildren(ctx, ref)
}

func TestRunFolderErrorContinues(t *testing.T) {
	base := demoSource()
	proy := folderRef(t, base, "Proyectos")
	src := failingSource{Source: base, failPath: proy.EntryID + `\Obra Norte`}
	req := baseReq(t, proy)
	req.Filters.IncludeSubfolders = true
	res, err := Run(context.Background(), src, req, &captureSink{}, fixedNow)
	if err != nil {
		t.Fatalf("un error de carpeta abortó el job: %v", err)
	}
	if res.Summary.Scanned != 41 {
		t.Fatalf("Scanned = %d, want 41 (Obra Sur)", res.Summary.Scanned)
	}
	if countReason(res.Records, domain.ReasonFolderError) != 1 {
		t.Fatal("falta el registro del error de carpeta")
	}
	if _, err := os.Stat(res.ReportPath); err != nil {
		t.Fatalf("no se escribió el reporte: %v", err)
	}

	src2 := failingSource{Source: base, listFailPath: proy.EntryID}
	req.DestDir = t.TempDir()
	res, err = Run(context.Background(), src2, req, &captureSink{}, fixedNow)
	if err != nil || countReason(res.Records, domain.ReasonFolderError) != 1 {
		t.Fatalf("ListChildren fallido: err=%v registros=%d", err, countReason(res.Records, domain.ReasonFolderError))
	}
}

func TestRunCancelRecordsPending(t *testing.T) {
	src := demoSource()
	full, err := Run(context.Background(), src, baseReq(t, folderRef(t, src, "Facturas proveedores")), &captureSink{}, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	saveErrs := func(recs []domain.RunRecord) int {
		n := 0
		for _, r := range recs {
			if r.Status == domain.StatusError && r.Reason != domain.ReasonReadError {
				n++
			}
		}
		return n
	}
	totalEligible := full.Summary.Saved + saveErrs(full.Records)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	saved := 0
	sink := &captureSink{onActivity: func(a Activity, _ int) {
		if a.Kind == ActSaved || a.Kind == ActRenamed {
			if saved++; saved == 5 {
				cancel()
			}
		}
	}}
	res, _ := Run(ctx, src, baseReq(t, folderRef(t, src, "Facturas proveedores")), sink, fixedNow)
	pending := countReason(res.Records, domain.ReasonCancelled)
	if got := res.Summary.Saved + saveErrs(res.Records) + pending; got != totalEligible {
		t.Fatalf("guardados+errores+pendientes = %d, want %d (pendientes=%d)", got, totalEligible, pending)
	}
}

func sumMap(m map[string]int) int {
	n := 0
	for _, v := range m {
		n += v
	}
	return n
}

func TestRunSummaryBreakdown(t *testing.T) {
	src := demoSource()
	req := baseReq(t, folderRef(t, src, "Facturas proveedores"))
	req.Organize = domain.OrganizeByMonth
	res, err := Run(context.Background(), src, req, &captureSink{}, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	s := res.Summary
	if got := sumMap(s.BySubfolder); got != s.Saved {
		t.Fatalf("suma BySubfolder = %d, Saved = %d", got, s.Saved)
	}
	month := regexp.MustCompile(`^\d{4}-\d{2}$`)
	for k := range s.BySubfolder {
		if !month.MatchString(k) {
			t.Fatalf("clave de subcarpeta inesperada %q", k)
		}
	}
	for _, r := range []string{"Adjunto bloqueado por Outlook", "Error de E/S simulado"} {
		if s.ErrorReasons[r] != 1 {
			t.Errorf("ErrorReasons[%q] = %d, want 1", r, s.ErrorReasons[r])
		}
		if _, ok := s.Reasons[r]; ok {
			t.Errorf("Reasons contiene el error %q", r)
		}
	}
	if got := sumMap(s.Reasons); got != s.Skipped {
		t.Fatalf("suma Reasons = %d, Skipped = %d", got, s.Skipped)
	}
}

func TestRunSummaryFlat(t *testing.T) {
	src := demoSource()
	req := baseReq(t, folderRef(t, src, "Facturas proveedores"))
	res, err := Run(context.Background(), src, req, &captureSink{}, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	s := res.Summary
	if len(s.BySubfolder) != 1 || s.BySubfolder[""] != s.Saved {
		t.Fatalf("BySubfolder = %v, Saved = %d", s.BySubfolder, s.Saved)
	}
}
