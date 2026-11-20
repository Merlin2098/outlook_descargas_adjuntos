// Command tarjetas genera las páginas HTML del final del video demo a partir de datos reales: una vista
// de la carpeta de destino con los archivos descargados, vistas de un XLSX y un XML descargados, las
// hojas del reporte Excel, la salida de `go test ./...` y el tamaño del binario.
// Ver scripts/grabar-demo/README.md.
package main

import (
	"flag"
	"fmt"
	"html"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xuri/excelize/v2"
)

func main() {
	dest := flag.String("dest", `C:\demo\video`, "carpeta de destino de la descarga grabada")
	out := flag.String("out", "", "carpeta donde escribir las tarjetas HTML")
	gotest := flag.String("gotest", "", "archivo con la salida de `go test ./...`")
	exe := flag.String("exe", "build/bin/outlook_descargas_adjuntos.exe", "binario compilado")
	fonts := flag.String("fonts", "frontend/node_modules/@fontsource", "carpeta de @fontsource")
	flag.Parse()
	if *out == "" || *gotest == "" {
		log.Fatal("faltan -out o -gotest")
	}
	fontDir, err := filepath.Abs(*fonts)
	if err != nil {
		log.Fatal(err)
	}
	tree, err := scanDest(*dest)
	if err != nil {
		log.Fatal(err)
	}
	if tree.Report == "" {
		log.Fatalf("no hay reporte_*.xlsx en %s", *dest)
	}
	sheets, err := readSheets(filepath.Join(*dest, tree.Report), 12)
	if err != nil {
		log.Fatal(err)
	}
	testOut, err := os.ReadFile(*gotest)
	if err != nil {
		log.Fatal(err)
	}
	st, err := os.Stat(*exe)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}
	show, err := pickShowcase(tree)
	if err != nil {
		log.Fatal(err)
	}
	folder := filepath.Join(*dest, show.Folder)
	xlsx, err := xlsxViewer(filepath.Join(folder, show.XLSX))
	if err != nil {
		log.Fatal(err)
	}
	pages := map[string]string{
		"explorador.html": page(fontDir, explorerPage(*dest, tree, show)),
		"ver-xlsx.html":   page(fontDir, viewerFrame(filepath.Join(folder, show.XLSX), "Estado de cuenta (XLSX)", xlsx)),
		"reporte.html":    page(fontDir, reportCard(tree.Report, sheets)),
		"terminal.html":   page(fontDir, terminalCard(string(testOut), filepath.Base(*exe), st.Size())),
	}
	if show.XML != "" {
		xmlView, err := xmlViewer(filepath.Join(folder, show.XML))
		if err != nil {
			log.Fatal(err)
		}
		pages["ver-xml.html"] = page(fontDir, viewerFrame(filepath.Join(folder, show.XML), "Factura electrónica (XML)", xmlView))
	}
	for name, body := range pages {
		if err := os.WriteFile(filepath.Join(*out, name), []byte(body), 0o644); err != nil {
			log.Fatal(err)
		}
	}
}

// Folder es una subcarpeta del destino con sus archivos.
type Folder struct {
	Name  string
	Files []File
	Bytes int64
}

// File es un archivo descargado.
type File struct {
	Name  string
	Bytes int64
}

// Tree es el contenido del destino: subcarpetas por mes y el reporte más reciente.
type Tree struct {
	Folders []Folder
	Report  string
}

// scanDest lee el destino: subcarpetas (ordenadas) con sus archivos y el último reporte_*.xlsx.
func scanDest(dir string) (Tree, error) {
	var t Tree
	entries, err := os.ReadDir(dir)
	if err != nil {
		return t, err
	}
	for _, e := range entries {
		switch {
		case e.IsDir():
			f := Folder{Name: e.Name()}
			files, err := os.ReadDir(filepath.Join(dir, e.Name()))
			if err != nil {
				return t, err
			}
			for _, fe := range files {
				if fe.IsDir() {
					continue
				}
				info, err := fe.Info()
				if err != nil {
					return t, err
				}
				f.Files = append(f.Files, File{fe.Name(), info.Size()})
				f.Bytes += info.Size()
			}
			t.Folders = append(t.Folders, f)
		case strings.HasPrefix(e.Name(), "reporte_") && strings.HasSuffix(e.Name(), ".xlsx"):
			if e.Name() > t.Report {
				t.Report = e.Name()
			}
		}
	}
	sort.Slice(t.Folders, func(i, j int) bool { return t.Folders[i].Name < t.Folders[j].Name })
	return t, nil
}

// Sheet son las primeras filas de una hoja del reporte y su total de filas de datos.
type Sheet struct {
	Name  string
	Rows  [][]string
	Total int
}

func readSheets(path string, maxRows int) ([]Sheet, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []Sheet
	for _, name := range f.GetSheetList() {
		rows, err := f.GetRows(name)
		if err != nil {
			return nil, fmt.Errorf("hoja %s: %w", name, err)
		}
		s := Sheet{Name: name, Total: max(len(rows)-1, 0)}
		if len(rows) > maxRows+1 {
			rows = rows[:maxRows+1]
		}
		s.Rows = rows
		out = append(out, s)
	}
	return out, nil
}

// humanSize formatea bytes como en la UI: "18,8 MB", "53 KB".
func humanSize(n int64) string {
	switch {
	case n >= 1<<20:
		return strings.Replace(fmt.Sprintf("%.1f MB", float64(n)/(1<<20)), ".", ",", 1)
	case n >= 1<<10:
		return fmt.Sprintf("%d KB", (n+512)>>10)
	default:
		return fmt.Sprintf("%d B", n)
	}
}

func esc(s string) string { return html.EscapeString(s) }

// pickColumns se queda con las columnas cuyo encabezado empieza por cada prefijo, en ese orden.
func pickColumns(rows [][]string, prefixes []string) [][]string {
	if len(rows) == 0 {
		return rows
	}
	var idx []int
	for _, p := range prefixes {
		for i, h := range rows[0] {
			if strings.HasPrefix(h, p) {
				idx = append(idx, i)
				break
			}
		}
	}
	out := make([][]string, len(rows))
	for r, row := range rows {
		out[r] = make([]string, len(idx))
		for c, i := range idx {
			if i < len(row) {
				out[r][c] = row[i]
			}
		}
	}
	return out
}

// sheetColumns son las columnas que caben en la tarjeta para cada hoja.
var sheetColumns = map[string][]string{
	"Adjuntos": {"Archivo", "Fecha", "Remitente", "Tamaño"},
	"Omitidos": {"Motivo", "Fecha", "Asunto"},
}

func reportCard(name string, sheets []Sheet) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<div class="kicker">El reporte Excel de la ejecución</div><h1>%s</h1><div class="tabs">`, esc(name))
	for _, s := range sheets {
		fmt.Fprintf(&b, `<span class="tab">%s <span class="dim">· %d filas</span></span>`, esc(s.Name), s.Total)
	}
	b.WriteString(`</div><div class="cols">`)
	for _, s := range sheets {
		if s.Name == "Resumen" {
			continue
		}
		fmt.Fprintf(&b, `<div class="panel sheet"><div class="sheet-name">%s</div><table>`, esc(s.Name))
		rows := s.Rows
		if cols, ok := sheetColumns[s.Name]; ok {
			rows = pickColumns(rows, cols)
		}
		for i, row := range rows {
			b.WriteString("<tr>")
			for _, c := range row {
				tag := "td"
				if i == 0 {
					tag = "th"
				}
				fmt.Fprintf(&b, "<%s>%s</%s>", tag, esc(c), tag)
			}
			b.WriteString("</tr>")
		}
		b.WriteString("</table></div>")
	}
	b.WriteString(`</div>`)
	return b.String()
}

func terminalCard(testOut, exe string, size int64) string {
	var b strings.Builder
	b.WriteString(`<div class="kicker">Calidad</div><h1>Tests en verde y un binario único</h1><div class="panel term mono">`)
	b.WriteString(`<div><span class="prompt">$</span> go test ./...</div>`)
	for _, line := range strings.Split(strings.TrimSpace(testOut), "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" || strings.Contains(line, "[no test files]") {
			continue
		}
		cls := ""
		if strings.HasPrefix(line, "ok") {
			cls = "ok"
		} else if strings.HasPrefix(line, "FAIL") || strings.HasPrefix(line, "---") {
			cls = "fail"
		}
		fmt.Fprintf(&b, `<div class="%s">%s</div>`, cls, esc(strings.ReplaceAll(line, "\t", "   ")))
	}
	fmt.Fprintf(&b, `<div class="gap"><span class="prompt">$</span> ls build/bin</div><div>%s <span class="ok">%s</span> · solo necesita WebView2</div></div>`, esc(exe), humanSize(size))
	return b.String()
}

func page(fontDir, body string) string {
	font := func(p string) string { return "file:///" + filepath.ToSlash(filepath.Join(fontDir, p)) }
	return `<!doctype html><html lang="es"><head><meta charset="utf-8"><style>
@font-face{font-family:Inter;font-weight:400;src:url(` + font("inter/files/inter-latin-400-normal.woff2") + `)}
@font-face{font-family:Inter;font-weight:600;src:url(` + font("inter/files/inter-latin-600-normal.woff2") + `)}
@font-face{font-family:Inter;font-weight:700;src:url(` + font("inter/files/inter-latin-700-normal.woff2") + `)}
@font-face{font-family:'Fira Code';font-weight:400;src:url(` + font("fira-code/files/fira-code-latin-400-normal.woff2") + `)}
html,body{margin:0;width:1280px;height:720px;overflow:hidden;background:#0D0F22;color:#E2E8F0;font-family:Inter,system-ui,sans-serif}
.card{box-sizing:border-box;height:720px;padding:48px 64px;display:flex;flex-direction:column;gap:14px}
.kicker{color:#38BDF8;font-weight:600;font-size:15px;text-transform:uppercase;letter-spacing:.08em}
h1{margin:0;font-size:34px;font-weight:700}
.sub{margin:0;color:#94A3B8;font-size:17px}
.mono{font-family:'Fira Code',ui-monospace,monospace}
.dim{color:#94A3B8}
.cols{display:flex;gap:24px;flex:1;min-height:0}
.panel{background:#1B2233;border:1px solid #2C3549;border-radius:8px;padding:16px 20px;min-width:0}
.tabs{display:flex;gap:10px}.tab{background:#252D42;border:1px solid #334155;border-radius:4px;padding:6px 12px;font-size:14px;font-weight:600}
.sheet{flex:1;overflow:hidden}.sheet:first-of-type{flex:1.25}.sheet-name{color:#38BDF8;font-weight:600;margin-bottom:10px}
table{border-collapse:collapse;font-size:12.5px;width:100%;table-layout:fixed}
th{background:#0369A1;color:#fff;text-align:left;padding:5px 8px;font-weight:600;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
td{padding:4px 8px;border-bottom:1px solid #2C3549;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.term{flex:1;font-size:16px;line-height:1.7;background:#0B0E1A}.term div{white-space:pre}
.prompt{color:#38BDF8;margin-right:8px}.ok{color:#4ADE80}.fail{color:#FB7185}.gap{margin-top:18px}
</style></head><body><div class="card">` + body + `</div></body></html>`
}
