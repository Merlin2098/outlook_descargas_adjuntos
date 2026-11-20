package demo

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"math/rand/v2"
	"path/filepath"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"outlook_descargas_adjuntos/internal/domain"
)

// fileContent genera el contenido de un adjunto demo: archivos válidos y deterministas (una factura PDF
// con su XML, un estado de cuenta XLSX, un DOCX, imágenes) para que se puedan abrir al descargarlos.
// Todos los datos son ficticios. El tamaño no coincide con FixtureAttachment.Size, igual que en
// Outlook, donde el tamaño declarado de un adjunto tampoco es el del archivo guardado.
func fileContent(m domain.Mail, a FixtureAttachment) ([]byte, error) {
	ext := strings.ToLower(filepath.Ext(a.FileName))
	base := strings.TrimSuffix(a.FileName, filepath.Ext(a.FileName))
	switch {
	case ext == ".pdf" && strings.HasPrefix(base, "factura_"):
		return invoicePDF(invoiceFor(m, base)), nil
	case ext == ".xml" && strings.HasPrefix(base, "factura_"):
		return invoiceXML(invoiceFor(m, base)), nil
	case ext == ".pdf":
		return reportPDF(m, base), nil
	case ext == ".xlsx":
		return statementXLSX(m, base)
	case ext == ".docx":
		return letterDOCX(m, base)
	case ext == ".png":
		return logoPNG(m, base)
	case ext == ".jpg" || ext == ".jpeg":
		return photoJPEG(m, base)
	default:
		return []byte(fmt.Sprintf("Archivo demo %s del correo %q (datos ficticios).\n", a.FileName, m.Subject)), nil
	}
}

// seeded devuelve un generador determinista para las partes dadas.
func seeded(parts ...string) *rand.Rand {
	h := fnv.New64a()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return rand.New(rand.NewPCG(h.Sum64(), 0x5eed))
}

// ---------------------------------------------------------------- factura

type invoiceLine struct {
	Desc string
	Qty  int
	Unit float64
}

type invoice struct {
	Number     string
	Issued     time.Time
	Supplier   string
	SupplierID string
	Customer   string
	CustomerID string
	Lines      []invoiceLine
}

const taxRate = 0.18

func (v invoice) Subtotal() float64 {
	var s float64
	for _, l := range v.Lines {
		s += float64(l.Qty) * l.Unit
	}
	return round2(s)
}

func (v invoice) Tax() float64   { return round2(v.Subtotal() * taxRate) }
func (v invoice) Total() float64 { return round2(v.Subtotal() + v.Tax()) }

func round2(x float64) float64 { return math.Round(x*100) / 100 }

var catalog = []invoiceLine{
	{"Cemento tipo I, bolsa de 42,5 kg", 0, 9.80}, {"Varilla de acero corrugado 1/2\"", 0, 7.45},
	{"Servicio de transporte de materiales", 0, 180.00}, {"Alquiler de mezcladora (día)", 0, 95.00},
	{"Pintura látex blanca, galón", 0, 21.90}, {"Mantenimiento preventivo de equipos", 0, 320.00},
	{"Tubería PVC 4\" x 3 m", 0, 14.30}, {"Arena gruesa (m³)", 0, 38.50},
}

// invoiceFor arma los datos de la factura `base` (factura_F001-00100) del correo m. Son los mismos para
// el PDF y el XML de la misma factura.
func invoiceFor(m domain.Mail, base string) invoice {
	r := seeded(m.Sender, base)
	v := invoice{
		Number:     strings.TrimPrefix(base, "factura_"),
		Issued:     m.ReceivedAt.AddDate(0, 0, -r.IntN(3)),
		Supplier:   "Proveedor Demo S.A.",
		SupplierID: "ID fiscal 20-555000-1",
		Customer:   "Empresa Ejemplo S.A.C.",
		CustomerID: "ID fiscal 20-777000-9",
	}
	for _, i := range r.Perm(len(catalog))[:2+r.IntN(3)] {
		l := catalog[i]
		l.Qty = 1 + r.IntN(40)
		v.Lines = append(v.Lines, l)
	}
	return v
}

// formatMoney da el formato de la UI: "$ 1.234,56".
func formatMoney(x float64) string {
	s := fmt.Sprintf("%.2f", x)
	intPart, dec := s[:len(s)-3], s[len(s)-2:]
	var b strings.Builder
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	return "$ " + b.String() + "," + dec
}

func invoicePDF(v invoice) []byte {
	var c pdfCanvas
	c.rect(0, 772, 595, 70, 0.012, 0.412, 0.631)
	c.text(48, 812, 20, true, 1, 1, 1, "FACTURA ELECTRÓNICA")
	c.text(48, 790, 11, false, 0.88, 0.95, 1, "Documento demo · datos ficticios")
	c.textRight(547, 812, 16, true, 1, 1, 1, v.Number)
	c.textRight(547, 790, 11, false, 0.88, 0.95, 1, "Emisión: "+v.Issued.Format("02/01/2006"))

	c.text(48, 730, 9, true, 0.39, 0.45, 0.55, "EMISOR")
	c.text(48, 714, 12, true, 0.12, 0.16, 0.23, v.Supplier)
	c.text(48, 699, 10, false, 0.28, 0.33, 0.41, v.SupplierID)
	c.text(48, 685, 10, false, 0.28, 0.33, 0.41, "Av. de los Ejemplos 123, Ciudad Demo")
	c.text(320, 730, 9, true, 0.39, 0.45, 0.55, "CLIENTE")
	c.text(320, 714, 12, true, 0.12, 0.16, 0.23, v.Customer)
	c.text(320, 699, 10, false, 0.28, 0.33, 0.41, v.CustomerID)
	c.text(320, 685, 10, false, 0.28, 0.33, 0.41, "Condición: crédito a 30 días")

	y := 640.0
	c.rect(48, y-6, 499, 22, 0.945, 0.961, 0.976)
	c.text(56, y, 10, true, 0.12, 0.16, 0.23, "Descripción")
	c.textRight(380, y, 10, true, 0.12, 0.16, 0.23, "Cant.")
	c.textRight(460, y, 10, true, 0.12, 0.16, 0.23, "P. unitario")
	c.textRight(539, y, 10, true, 0.12, 0.16, 0.23, "Importe")
	for _, l := range v.Lines {
		y -= 26
		c.text(56, y, 10, false, 0.12, 0.16, 0.23, l.Desc)
		c.textRight(380, y, 10, false, 0.12, 0.16, 0.23, fmt.Sprint(l.Qty))
		c.textRight(460, y, 10, false, 0.12, 0.16, 0.23, formatMoney(l.Unit))
		c.textRight(539, y, 10, false, 0.12, 0.16, 0.23, formatMoney(float64(l.Qty)*l.Unit))
		c.line(48, y-9, 547, y-9, 0.886, 0.91, 0.941)
	}
	y -= 40
	for _, row := range [][2]string{{"Subtotal", formatMoney(v.Subtotal())}, {"Impuesto (18 %)", formatMoney(v.Tax())}} {
		c.textRight(460, y, 10, false, 0.28, 0.33, 0.41, row[0])
		c.textRight(539, y, 10, false, 0.12, 0.16, 0.23, row[1])
		y -= 18
	}
	y -= 12
	c.rect(330, y-8, 217, 26, 0.012, 0.412, 0.631)
	c.textRight(460, y, 12, true, 1, 1, 1, "Total")
	c.textRight(539, y, 12, true, 1, 1, 1, formatMoney(v.Total()))

	c.text(48, 60, 8, false, 0.58, 0.64, 0.72, "Generado por el modo demo de outlook_descargas_adjuntos. Todos los datos son ficticios.")
	return c.pdf()
}

func invoiceXML(v invoice) []byte {
	esc := func(s string) string {
		var b bytes.Buffer
		_ = xml.EscapeText(&b, []byte(s))
		return b.String()
	}
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<Invoice xmlns="urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"` +
		` xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2"` +
		` xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2">` + "\n")
	b.WriteString("  <!-- Factura demo: datos ficticios -->\n")
	fmt.Fprintf(&b, "  <cbc:ID>%s</cbc:ID>\n  <cbc:IssueDate>%s</cbc:IssueDate>\n  <cbc:DocumentCurrencyCode>USD</cbc:DocumentCurrencyCode>\n",
		esc(v.Number), v.Issued.Format("2006-01-02"))
	party := func(tag, name, id string) {
		fmt.Fprintf(&b, "  <cac:%s>\n    <cac:Party>\n      <cbc:Name>%s</cbc:Name>\n      <cbc:CompanyID>%s</cbc:CompanyID>\n    </cac:Party>\n  </cac:%s>\n",
			tag, esc(name), esc(id), tag)
	}
	party("AccountingSupplierParty", v.Supplier, v.SupplierID)
	party("AccountingCustomerParty", v.Customer, v.CustomerID)
	fmt.Fprintf(&b, "  <cac:TaxTotal>\n    <cbc:TaxAmount currencyID=\"USD\">%.2f</cbc:TaxAmount>\n  </cac:TaxTotal>\n", v.Tax())
	fmt.Fprintf(&b, "  <cac:LegalMonetaryTotal>\n    <cbc:LineExtensionAmount currencyID=\"USD\">%.2f</cbc:LineExtensionAmount>\n    <cbc:PayableAmount currencyID=\"USD\">%.2f</cbc:PayableAmount>\n  </cac:LegalMonetaryTotal>\n",
		v.Subtotal(), v.Total())
	for i, l := range v.Lines {
		fmt.Fprintf(&b, "  <cac:InvoiceLine>\n    <cbc:ID>%d</cbc:ID>\n    <cbc:InvoicedQuantity>%d</cbc:InvoicedQuantity>\n"+
			"    <cbc:LineExtensionAmount currencyID=\"USD\">%.2f</cbc:LineExtensionAmount>\n"+
			"    <cac:Item><cbc:Description>%s</cbc:Description></cac:Item>\n"+
			"    <cac:Price><cbc:PriceAmount currencyID=\"USD\">%.2f</cbc:PriceAmount></cac:Price>\n  </cac:InvoiceLine>\n",
			i+1, l.Qty, float64(l.Qty)*l.Unit, esc(l.Desc), l.Unit)
	}
	b.WriteString("</Invoice>\n")
	return []byte(b.String())
}

// ---------------------------------------------------------------- otros documentos

var reportParagraphs = []string{
	"Se resume el avance de las actividades del periodo y los pendientes para la próxima revisión.",
	"Los entregables comprometidos se completaron dentro del plazo acordado con el cliente.",
	"Se adjuntan las mediciones de obra y el registro fotográfico de la semana.",
	"Quedan pendientes la aprobación del presupuesto adicional y la visita técnica.",
	"El equipo propone revisar el cronograma en la reunión del próximo lunes.",
	"No se registraron incidentes de seguridad durante el periodo informado.",
}

func reportPDF(m domain.Mail, base string) []byte {
	r := seeded(m.EntryID, base)
	var c pdfCanvas
	c.rect(0, 800, 595, 42, 0.012, 0.412, 0.631)
	c.text(48, 815, 14, true, 1, 1, 1, m.Subject)
	c.text(48, 770, 10, false, 0.28, 0.33, 0.41, "Fecha: "+m.ReceivedAt.Format("02/01/2006")+" · De: "+m.Sender)
	y := 735.0
	for _, i := range r.Perm(len(reportParagraphs))[:4] {
		c.text(48, y, 11, false, 0.12, 0.16, 0.23, "• "+reportParagraphs[i])
		y -= 24
	}
	c.text(48, 60, 8, false, 0.58, 0.64, 0.72, "Generado por el modo demo de outlook_descargas_adjuntos. Todos los datos son ficticios.")
	return c.pdf()
}

func statementXLSX(m domain.Mail, base string) ([]byte, error) {
	r := seeded(m.Sender, base)
	f := excelize.NewFile()
	defer f.Close()
	const sheet = "Estado de cuenta"
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		return nil, err
	}
	title := "Estado de cuenta · " + strings.ReplaceAll(strings.TrimPrefix(base, "estado_cuenta_"), "_", " ")
	rows := [][]any{
		{title}, {"Proveedor Demo S.A. · Empresa Ejemplo S.A.C. · datos ficticios"}, {},
		{"Fecha", "Documento", "Detalle", "Cargo", "Abono", "Saldo"},
	}
	saldo := float64(1000 + r.IntN(4000))
	rows = append(rows, []any{m.ReceivedAt.AddDate(0, -1, 0).Format("02/01/2006"), "—", "Saldo anterior", nil, nil, saldo})
	for i := 0; i < 8+r.IntN(5); i++ {
		d := m.ReceivedAt.AddDate(0, -1, 2+i*3).Format("02/01/2006")
		if r.IntN(3) == 0 {
			abono := round2(float64(200+r.IntN(1500)) + float64(r.IntN(100))/100)
			saldo = round2(saldo - abono)
			rows = append(rows, []any{d, fmt.Sprintf("REC-%04d", 300+r.IntN(600)), "Pago recibido, transferencia", nil, abono, saldo})
		} else {
			cargo := round2(float64(150+r.IntN(2200)) + float64(r.IntN(100))/100)
			saldo = round2(saldo + cargo)
			rows = append(rows, []any{d, fmt.Sprintf("F001-%05d", 100+r.IntN(212)), "Factura por suministros", cargo, nil, saldo})
		}
	}
	for i, row := range rows {
		cell, _ := excelize.CoordinatesToCellName(1, i+1)
		if err := f.SetSheetRow(sheet, cell, &row); err != nil {
			return nil, err
		}
	}
	header, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"0369A1"}, Pattern: 1}})
	if err != nil {
		return nil, err
	}
	money, err := f.NewStyle(&excelize.Style{NumFmt: 4})
	if err != nil {
		return nil, err
	}
	bold, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 14, Color: "0369A1"}})
	if err != nil {
		return nil, err
	}
	_ = f.SetCellStyle(sheet, "A1", "A1", bold)
	_ = f.SetCellStyle(sheet, "A4", "F4", header)
	_ = f.SetCellStyle(sheet, "D5", fmt.Sprintf("F%d", len(rows)), money)
	for col, w := range map[string]float64{"A": 12, "B": 13, "C": 32, "D": 12, "E": 12, "F": 13} {
		_ = f.SetColWidth(sheet, col, col, w)
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func letterDOCX(m domain.Mail, base string) ([]byte, error) {
	paras := []string{
		"Consulta de pago",
		"Fecha: " + m.ReceivedAt.Format("02/01/2006"),
		"Asunto del correo: " + m.Subject,
		"Estimados:",
		"Les escribimos para confirmar el estado de pago de las facturas emitidas el mes pasado. " +
			"Según nuestro registro, quedan pendientes los documentos indicados en el estado de cuenta adjunto.",
		"Agradeceremos nos indiquen la fecha prevista de pago o, si ya fue realizado, el número de operación.",
		"Saludos cordiales,",
		"Área de cobranzas · Proveedor Demo S.A. (datos ficticios)",
	}
	var body strings.Builder
	for i, p := range paras {
		var t bytes.Buffer
		_ = xml.EscapeText(&t, []byte(p))
		rpr := ""
		if i == 0 {
			rpr = `<w:rPr><w:b/><w:sz w:val="32"/><w:color w:val="0369A1"/></w:rPr>`
		}
		fmt.Fprintf(&body, `<w:p><w:r>%s<w:t xml:space="preserve">%s</w:t></w:r></w:p>`, rpr, t.String())
	}
	parts := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
			`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
			`<Default Extension="xml" ContentType="application/xml"/>` +
			`<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`,
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`,
		"word/document.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` +
			body.String() + `</w:body></w:document>`,
	}
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	for _, name := range []string{"[Content_Types].xml", "_rels/.rels", "word/document.xml"} {
		w, err := z.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write([]byte(parts[name])); err != nil {
			return nil, err
		}
	}
	if err := z.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// logoPNG es un logo de firma: fondo azul con tres barras claras.
func logoPNG(m domain.Mail, base string) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, 160, 60))
	for y := 0; y < 60; y++ {
		for x := 0; x < 160; x++ {
			c := color.RGBA{3, 105, 161, 255}
			if y >= 18 && y < 42 && (x/14)%2 == 1 && x < 90 {
				c = color.RGBA{224, 242, 254, 255}
			}
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	return buf.Bytes(), err
}

// photoJPEG es una "foto de obra": cielo, horizonte y terreno con un tono que depende del archivo.
func photoJPEG(m domain.Mail, base string) ([]byte, error) {
	r := seeded(m.EntryID, base)
	hz := 140 + r.IntN(80)
	img := image.NewRGBA(image.Rect(0, 0, 640, 400))
	for y := 0; y < 400; y++ {
		for x := 0; x < 640; x++ {
			var c color.RGBA
			if y < hz {
				t := float64(y) / float64(hz)
				c = color.RGBA{uint8(120 + 80*t), uint8(170 + 50*t), 235, 255}
			} else {
				t := float64(y-hz) / float64(400-hz)
				c = color.RGBA{uint8(150 - 60*t), uint8(120 - 40*t), uint8(80 - 30*t), 255}
			}
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
	return buf.Bytes(), err
}

// ---------------------------------------------------------------- PDF mínimo

// pdfCanvas dibuja una página A4 con texto (Helvetica, WinAnsi), rectángulos y líneas.
type pdfCanvas struct{ ops bytes.Buffer }

func (c *pdfCanvas) rect(x, y, w, h, r, g, b float64) {
	fmt.Fprintf(&c.ops, "%.3f %.3f %.3f rg %.1f %.1f %.1f %.1f re f\n", r, g, b, x, y, w, h)
}

func (c *pdfCanvas) line(x1, y1, x2, y2, r, g, b float64) {
	fmt.Fprintf(&c.ops, "%.3f %.3f %.3f RG 0.6 w %.1f %.1f m %.1f %.1f l S\n", r, g, b, x1, y1, x2, y2)
}

func (c *pdfCanvas) text(x, y, size float64, bold bool, r, g, b float64, s string) {
	font := "F1"
	if bold {
		font = "F2"
	}
	fmt.Fprintf(&c.ops, "BT %.3f %.3f %.3f rg /%s %.1f Tf %.1f %.1f Td (", r, g, b, font, size, x, y)
	c.ops.Write(pdfEscape(toWinAnsi(s)))
	c.ops.WriteString(") Tj ET\n")
}

// textRight alinea a la derecha con un ancho aproximado de Helvetica (0,5 em por carácter).
func (c *pdfCanvas) textRight(x, y, size float64, bold bool, r, g, b float64, s string) {
	w := 0.0
	for _, ch := range s {
		switch {
		case ch == ' ' || ch == '.' || ch == ',':
			w += 0.28
		case ch >= '0' && ch <= '9':
			w += 0.556
		case ch >= 'A' && ch <= 'Z':
			w += 0.67
		default:
			w += 0.5
		}
	}
	if bold {
		w *= 1.05
	}
	c.text(x-w*size, y, size, bold, r, g, b, s)
}

func (c *pdfCanvas) pdf() []byte {
	var out bytes.Buffer
	var offsets []int
	obj := func(body string) {
		offsets = append(offsets, out.Len())
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", len(offsets), body)
	}
	out.WriteString("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")
	obj("<< /Type /Catalog /Pages 2 0 R >>")
	obj("<< /Type /Pages /Kids [3 0 R] /Count 1 >>")
	obj("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 4 0 R /F2 5 0 R >> >> /Contents 6 0 R >>")
	obj("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>")
	obj("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>")
	obj(fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", c.ops.Len(), c.ops.String()))
	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(offsets)+1)
	for _, o := range offsets {
		fmt.Fprintf(&out, "%010d 00000 n \n", o)
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets)+1, xref)
	return out.Bytes()
}

// toWinAnsi convierte a la codificación de las fuentes estándar del PDF. Los caracteres latinos
// (á, é, ñ, ³…) coinciden con Latin-1; el resto se cambia por equivalentes ASCII.
func toWinAnsi(s string) []byte {
	var out []byte
	for _, r := range s {
		switch {
		case r == '•':
			out = append(out, 0x95)
		case r == '—':
			out = append(out, 0x97)
		case r < 256:
			out = append(out, byte(r))
		default:
			out = append(out, '?')
		}
	}
	return out
}

func pdfEscape(b []byte) []byte {
	var out []byte
	for _, c := range b {
		if c == '(' || c == ')' || c == '\\' {
			out = append(out, '\\')
		}
		out = append(out, c)
	}
	return out
}
