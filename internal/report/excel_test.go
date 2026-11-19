package report

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"outlook_descargas_adjuntos/internal/domain"
)

func TestFileName(t *testing.T) {
	got := FileName(time.Date(2026, 11, 12, 10, 43, 0, 0, time.Local))
	if got != "reporte_2026-11-12_1043.xlsx" {
		t.Fatalf("FileName = %q", got)
	}
}

func TestWriteExcel(t *testing.T) {
	now := time.Date(2026, 11, 12, 10, 0, 0, 0, time.Local)
	recs := []domain.RunRecord{
		{Status: domain.StatusSaved, FileName: "factura_1.pdf", SavedPath: `C:\x\factura_1.pdf`, Size: 2048, Subject: "Factura 1", Sender: "a@b", ReceivedAt: now},
		{Status: domain.StatusSkipped, Reason: domain.ReasonNotMail, Subject: "Invitación"},
		{Status: domain.StatusError, Reason: "Error de E/S simulado", FileName: "x.pdf"},
		{Status: domain.StatusSaved, FileName: "factura_2.pdf", SavedPath: `C:\x\factura_2.pdf`, Size: 1024},
	}
	sum := domain.Summary{Saved: 2, Skipped: 1, Errors: 1, Started: now, Finished: now}
	req := domain.JobRequest{Folder: domain.FolderRef{Path: `demo\Facturas`}, DestDir: `C:\x`}
	path := filepath.Join(t.TempDir(), "r.xlsx")
	if err := WriteExcel(path, req, recs, sum); err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sheets := f.GetSheetList()
	want := []string{"Resumen", "Adjuntos", "Omitidos"}
	if len(sheets) != 3 || sheets[0] != want[0] || sheets[1] != want[1] || sheets[2] != want[2] {
		t.Fatalf("hojas = %v", sheets)
	}
	if v, _ := f.GetCellValue("Adjuntos", "D1"); v != "Archivo" {
		t.Fatalf("Adjuntos!D1 = %q", v)
	}
	if v, _ := f.GetCellValue("Adjuntos", "D2"); v != "factura_1.pdf" {
		t.Fatalf("Adjuntos!D2 = %q", v)
	}
	rows, _ := f.GetRows("Omitidos")
	if len(rows) != 1+2 {
		t.Fatalf("filas en Omitidos = %d, want 3", len(rows))
	}
}
