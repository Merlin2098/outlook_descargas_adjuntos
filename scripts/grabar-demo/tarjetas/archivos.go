package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/xuri/excelize/v2"
)

// Showcase son los archivos que el video abre: una factura PDF (y su XML, si existe) y un estado de cuenta.
type Showcase struct {
	Folder string
	PDF    string
	XML    string
	XLSX   string
}

var renamed = regexp.MustCompile(`_\d+$`) // sufijo _N de un nombre repetido

// pickShowcase elige, en la última subcarpeta, la última factura PDF (con su XML si se descargó) y el
// primer estado de cuenta, evitando los nombres renombrados con _N.
func pickShowcase(t Tree) (Showcase, error) {
	if len(t.Folders) == 0 {
		return Showcase{}, fmt.Errorf("el destino no tiene subcarpetas")
	}
	f := t.Folders[len(t.Folders)-1]
	names := map[string]bool{}
	for _, file := range f.Files {
		names[file.Name] = true
	}
	s := Showcase{Folder: f.Name}
	var pair Showcase
	lone := ""
	for _, file := range f.Files {
		base := strings.TrimSuffix(file.Name, filepath.Ext(file.Name))
		if renamed.MatchString(base) {
			continue
		}
		switch {
		case strings.HasPrefix(base, "factura_") && filepath.Ext(file.Name) == ".pdf" && names[base+".xml"]:
			pair = Showcase{PDF: file.Name, XML: base + ".xml"}
		case strings.HasPrefix(base, "factura_") && filepath.Ext(file.Name) == ".pdf":
			lone = file.Name
		case strings.HasPrefix(base, "estado_cuenta_") && filepath.Ext(file.Name) == ".xlsx" && s.XLSX == "":
			s.XLSX = file.Name
		}
	}
	s.PDF, s.XML = pair.PDF, pair.XML
	if s.PDF == "" {
		s.PDF = lone // sin XML descargado (por ejemplo, con el filtro pdf, xlsx)
	}
	if s.PDF == "" || s.XLSX == "" {
		return s, fmt.Errorf("no hay factura PDF o estado de cuenta en %s", f.Name)
	}
	return s, nil
}

var fileTypes = map[string]string{
	".pdf": "Documento PDF", ".xml": "Documento XML", ".xlsx": "Hoja de cálculo", ".docx": "Documento de Word",
	".jpg": "Imagen JPEG", ".png": "Imagen PNG",
}

// explorerPage es una vista de la carpeta de destino, generada con los archivos reales. Doble clic en
// un archivo destacado lo abre (el PDF real en el visor del navegador; XML y XLSX en sus vistas).
func explorerPage(dest string, t Tree, s Showcase) string {
	type row struct {
		Name, Size, Type, Ext, Open string
	}
	folders := map[string][]row{}
	var order []string
	for _, f := range t.Folders {
		order = append(order, f.Name)
		for _, file := range f.Files {
			ext := strings.ToLower(filepath.Ext(file.Name))
			r := row{Name: file.Name, Size: humanSize(file.Bytes), Type: fileTypes[ext], Ext: strings.TrimPrefix(ext, ".")}
			if f.Name == s.Folder {
				switch file.Name {
				case s.PDF:
					r.Open = fileURL(filepath.Join(dest, f.Name, file.Name))
				case s.XML:
					r.Open = "ver-xml.html"
				case s.XLSX:
					r.Open = "ver-xlsx.html"
				}
			}
			folders[f.Name] = append(folders[f.Name], r)
		}
	}
	data, _ := json.Marshal(map[string]any{"order": order, "folders": folders, "report": t.Report, "dest": dest})
	return `<style>
.win{flex:1;min-height:0;display:flex;background:#1B2233;border:1px solid #2C3549;border-radius:8px;overflow:hidden}
.side{width:230px;border-right:1px solid #2C3549;padding:14px 10px;display:flex;flex-direction:column;gap:2px}
.side div{padding:8px 12px;border-radius:4px;font-size:15px;cursor:default;display:flex;justify-content:space-between}
.side .sel{background:#38BDF8;color:#0D0F22;font-weight:600}
.main{flex:1;display:flex;flex-direction:column;min-width:0}
.path{padding:12px 18px;border-bottom:1px solid #2C3549;font-family:'Fira Code',monospace;font-size:14px;color:#94A3B8}
.list{flex:1;overflow-y:auto;scrollbar-width:none;padding:4px 10px}
.r{display:grid;grid-template-columns:1fr 110px 170px;gap:12px;align-items:center;padding:7px 10px;border-radius:4px;font-size:14.5px}
.r.h{color:#94A3B8;font-size:12.5px;font-weight:600;text-transform:uppercase;letter-spacing:.05em}
.r.hot{background:#252D42}.r.on{outline:2px solid #38BDF8}
.n{display:flex;gap:10px;align-items:center;font-family:'Fira Code',monospace;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.sz{text-align:right;font-family:'Fira Code',monospace;color:#94A3B8}.ty{color:#94A3B8}
.ext{font:700 10px Inter,sans-serif;padding:3px 5px;border-radius:3px;width:34px;text-align:center;flex:none;color:#0D0F22}
.pdf{background:#FB7185}.xml{background:#FCD34D}.xlsx{background:#4ADE80}.docx{background:#7DD3FC}.jpg,.png{background:#F0ABFC}
.badge{margin-left:auto;color:#94A3B8;font-size:13px}
</style>
<div class="kicker">Los adjuntos descargados</div><h1 id="title"></h1>
<p class="sub">Vista de la carpeta de destino, generada con los archivos reales de la descarga</p>
<div class="win"><div class="side" id="side"></div><div class="main"><div class="path" id="path"></div><div class="list" id="list"></div></div></div>
<script>
const D = ` + string(data) + `;
const side = document.getElementById('side'), list = document.getElementById('list'), path = document.getElementById('path');
document.getElementById('title').textContent = D.dest;
function show(folder) {
  history.replaceState(null, '', folder ? '#' + encodeURIComponent(folder) : '#');
  side.innerHTML = '';
  const root = document.createElement('div'); root.textContent = '📁 ' + D.dest.split('\\').pop() + '\\';
  if (!folder) root.className = 'sel';
  root.onclick = () => show(null); side.appendChild(root);
  for (const f of D.order) {
    const d = document.createElement('div'); d.dataset.folder = f;
    d.innerHTML = '<span>&nbsp;&nbsp;📁 ' + f + '\\</span><span>' + D.folders[f].length + '</span>';
    if (f === folder) d.className = 'sel';
    d.onclick = () => show(f); side.appendChild(d);
  }
  path.textContent = D.dest + (folder ? '\\' + folder : '');
  list.innerHTML = '<div class="r h"><span>Nombre</span><span class="sz">Tamaño</span><span>Tipo</span></div>';
  const rows = folder ? D.folders[folder]
    : D.order.map((f) => ({Name: f + '\\', Size: D.folders[f].length + ' archivos', Type: 'Carpeta', Ext: 'dir', folder: f}))
        .concat([{Name: D.report, Size: '', Type: 'Reporte de la ejecución', Ext: 'xlsx'}]);
  for (const r of rows) {
    const e = document.createElement('div'); e.className = 'r'; e.dataset.name = r.Name;
    const ext = r.Ext === 'dir' ? '<span class="ext" style="background:#94A3B8">DIR</span>' : '<span class="ext ' + r.Ext + '">' + r.Ext.toUpperCase() + '</span>';
    e.innerHTML = '<span class="n">' + ext + r.Name + '</span><span class="sz">' + r.Size + '</span><span class="ty">' + r.Type + '</span>';
    if (r.Open) e.classList.add('hot');
    e.onclick = () => { list.querySelectorAll('.on').forEach((x) => x.classList.remove('on')); e.classList.add('on'); };
    e.ondblclick = () => { if (r.folder) show(r.folder); else if (r.Open) location.href = r.Open; };
    list.appendChild(e);
  }
}
show(decodeURIComponent(location.hash.slice(1)) || null); // al volver desde un archivo, reabre su carpeta
</script>`
}

func fileURL(p string) string {
	u := url.URL{Scheme: "file", Path: "/" + filepath.ToSlash(p)}
	return u.String()
}

// viewerFrame envuelve una vista de archivo con la ruta y una aclaración.
func viewerFrame(fullPath, kind, body string) string {
	return fmt.Sprintf(`<div class="kicker">%s</div><h1 class="mono" style="font-size:24px;font-weight:400">%s</h1>
<p class="sub">Vista del archivo descargado, leído del disco</p>%s`, esc(kind), esc(fullPath), body)
}

var enNumber = regexp.MustCompile(`^\d{1,3}(,\d{3})*\.\d{2}$`)

// spanishNumber pasa un importe con formato inglés (1,980.52) al de la UI (1.980,52).
func spanishNumber(v string) string {
	if !enNumber.MatchString(v) {
		return v
	}
	return strings.NewReplacer(",", ".", ".", ",").Replace(v)
}

// xlsxViewer muestra la primera hoja como una planilla, con letras de columna y números de fila.
func xlsxViewer(path string) (string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	sheet := f.GetSheetList()[0]
	rows, err := f.GetRows(sheet)
	if err != nil {
		return "", err
	}
	cols := 0
	for _, r := range rows {
		cols = max(cols, len(r))
	}
	var b strings.Builder
	b.WriteString(`<style>.grid{border-collapse:collapse;background:#fff;color:#1E293B;font-size:13.5px;table-layout:auto}
.grid td,.grid th{border:1px solid #CBD5E1;padding:5px 10px;white-space:nowrap}.grid th{background:#F1F5F9;color:#64748B;font-weight:600;font-size:12px}
.grid td.num{text-align:right;font-variant-numeric:tabular-nums}.grid tr.t td{font-weight:700;color:#0369A1;font-size:16px;border-color:transparent}
.grid tr.hd td{background:#0369A1;color:#fff;font-weight:600}.sheetbar{background:#F1F5F9;color:#1E293B;padding:6px 12px;font-size:13px;border:1px solid #CBD5E1;border-top:0;width:max-content}
.sheetbar b{border-bottom:2px solid #0369A1;padding:2px 6px}</style><div style="overflow:hidden;flex:1"><table class="grid"><tr><th></th>`)
	for c := 1; c <= cols; c++ {
		name, _ := excelize.ColumnNumberToName(c)
		fmt.Fprintf(&b, "<th>%s</th>", name)
	}
	b.WriteString("</tr>")
	for i, r := range rows {
		cls := ""
		switch {
		case i == 0:
			cls = ` class="t"`
		case len(r) > 0 && r[0] == "Fecha":
			cls = ` class="hd"`
		}
		fmt.Fprintf(&b, `<tr%s><th>%d</th>`, cls, i+1)
		if len(r) == 1 { // título o subtítulo: ocupa toda la fila, como una celda combinada
			fmt.Fprintf(&b, `<td colspan="%d">%s</td></tr>`, cols, esc(r[0]))
			continue
		}
		for c := 0; c < cols; c++ {
			v := ""
			if c < len(r) {
				v = r[c]
			}
			num := ""
			if c >= 3 && v != "" && v[0] >= '0' && v[0] <= '9' {
				num = ` class="num"`
			}
			fmt.Fprintf(&b, "<td%s>%s</td>", num, esc(spanishNumber(v)))
		}
		b.WriteString("</tr>")
	}
	fmt.Fprintf(&b, `</table><div class="sheetbar"><b>%s</b></div></div>`, esc(sheet))
	return b.String(), nil
}

var (
	xmlComment = regexp.MustCompile(`&lt;!--.*?--&gt;`)
	xmlTag     = regexp.MustCompile(`&lt;(/?)([\w:]+)((?:\s[^&]*?)?)(/?)&gt;`)
	xmlAttr    = regexp.MustCompile(`([\w:]+)=(&#34;.*?&#34;)`)
)

// xmlViewer muestra el XML con resaltado de etiquetas, atributos y comentarios.
func xmlViewer(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString(`<style>.code{flex:1;overflow:hidden;background:#0B0E1A;border:1px solid #2C3549;border-radius:8px;padding:16px 20px;
font:14px/1.6 'Fira Code',monospace;white-space:pre;color:#E2E8F0}.tg{color:#7DD3FC}.at{color:#F0ABFC}.av{color:#FCD34D}.cm{color:#64748B}</style><div class="code">`)
	for _, line := range strings.Split(strings.TrimRight(string(raw), "\n"), "\n") {
		s := esc(line)
		s = xmlComment.ReplaceAllString(s, `<span class="cm">$0</span>`)
		s = xmlTag.ReplaceAllStringFunc(s, func(m string) string {
			p := xmlTag.FindStringSubmatch(m)
			attrs := xmlAttr.ReplaceAllString(p[3], `<span class="at">$1</span>=<span class="av">$2</span>`)
			return `<span class="tg">&lt;` + p[1] + p[2] + `</span>` + attrs + `<span class="tg">` + p[4] + `&gt;</span>`
		})
		b.WriteString(s + "\n")
	}
	b.WriteString(`</div>`)
	return b.String(), nil
}
