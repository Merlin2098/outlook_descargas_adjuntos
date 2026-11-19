// Package report escribe el reporte Excel de una ejecución.
package report

import (
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"outlook_descargas_adjuntos/internal/domain"
)

const (
	sheetSummary = "Resumen"
	sheetSaved   = "Adjuntos"
	sheetSkipped = "Omitidos"
	headerFill   = "0369A1"
	dateLayout   = "02/01/2006 15:04"
)

// FileName devuelve el nombre del reporte para el instante t.
func FileName(t time.Time) string {
	return "reporte_" + t.Format("2006-01-02_1504") + ".xlsx"
}

func fmtDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(time.Local).Format(dateLayout)
}

var statusLabel = map[domain.RecordStatus]string{
	domain.StatusSaved:   "guardado",
	domain.StatusSkipped: "omitido",
	domain.StatusError:   "error",
}

// WriteExcel escribe las hojas Resumen, Adjuntos y Omitidos en path.
func WriteExcel(path string, req domain.JobRequest, recs []domain.RunRecord, sum domain.Summary) error {
	f := excelize.NewFile()
	defer f.Close()

	header, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{headerFill}, Pattern: 1},
	})
	if err != nil {
		return fmt.Errorf("estilo de encabezado: %w", err)
	}

	if err := f.SetSheetName("Sheet1", sheetSummary); err != nil {
		return err
	}
	fl := req.Filters
	summaryRows := [][]any{
		{"Campo", "Valor"},
		{"Origen", req.Folder.Path},
		{"Desde", fl.From.In(time.Local).Format("02/01/2006")},
		{"Hasta", fl.To.In(time.Local).Format("02/01/2006")},
		{"Asunto contiene", fl.SubjectContains},
		{"Remitente contiene", fl.SenderContains},
		{"Extensiones", strings.Join(fl.Extensions, " ")},
		{"Omitir inline", fl.SkipInline},
		{"Incluir subcarpetas", fl.IncludeSubfolders},
		{"Destino", req.DestDir},
		{"Inicio", fmtDate(sum.Started)},
		{"Fin", fmtDate(sum.Finished)},
		{"Cancelado", sum.Cancelled},
		{"Elementos revisados", sum.Scanned},
		{"Fuera de rango", sum.OutOfRange},
		{"En rango", sum.InRange},
		{"Correos con descargas", sum.MailsWithDownloads},
		{"Adjuntos guardados", sum.Saved},
		{"Omitidos", sum.Skipped},
		{"Errores", sum.Errors},
		{"Tamaño total (KB)", sum.Bytes / 1024},
	}
	if err := writeSheet(f, sheetSummary, summaryRows, header); err != nil {
		return err
	}

	saved := [][]any{{"Fecha correo", "Remitente", "Asunto", "Archivo", "Ruta guardada", "Tamaño (KB)"}}
	skipped := [][]any{{"Estado", "Motivo", "Fecha correo", "Remitente", "Asunto", "Archivo"}}
	for _, r := range recs {
		if r.Status == domain.StatusSaved {
			saved = append(saved, []any{fmtDate(r.ReceivedAt), r.Sender, r.Subject, r.FileName, r.SavedPath, (r.Size + 1023) / 1024})
			continue
		}
		skipped = append(skipped, []any{statusLabel[r.Status], r.Reason, fmtDate(r.ReceivedAt), r.Sender, r.Subject, r.FileName})
	}
	for _, s := range []struct {
		name string
		rows [][]any
	}{{sheetSaved, saved}, {sheetSkipped, skipped}} {
		if _, err := f.NewSheet(s.name); err != nil {
			return err
		}
		if err := writeSheet(f, s.name, s.rows, header); err != nil {
			return err
		}
	}

	if err := f.SaveAs(path); err != nil {
		return fmt.Errorf("guardar reporte %s: %w", path, err)
	}
	return nil
}

// writeSheet escribe las filas, da estilo al encabezado, lo fija y ajusta el ancho de las columnas.
func writeSheet(f *excelize.File, sheet string, rows [][]any, headerStyle int) error {
	widths := make([]int, len(rows[0]))
	for i, row := range rows {
		cell, _ := excelize.CoordinatesToCellName(1, i+1)
		if err := f.SetSheetRow(sheet, cell, &row); err != nil {
			return fmt.Errorf("hoja %s fila %d: %w", sheet, i+1, err)
		}
		for c, v := range row {
			if n := len([]rune(fmt.Sprint(v))); n > widths[c] {
				widths[c] = n
			}
		}
	}
	last, _ := excelize.CoordinatesToCellName(len(rows[0]), 1)
	if err := f.SetCellStyle(sheet, "A1", last, headerStyle); err != nil {
		return err
	}
	if err := f.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"}); err != nil {
		return err
	}
	for c, w := range widths {
		col, _ := excelize.ColumnNumberToName(c + 1)
		if err := f.SetColWidth(sheet, col, col, float64(max(10, min(60, w+2)))); err != nil {
			return err
		}
	}
	return nil
}
