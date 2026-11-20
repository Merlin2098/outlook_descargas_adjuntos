package demo

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"image/jpeg"
	"image/png"
	"io"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"outlook_descargas_adjuntos/internal/domain"
)

var testMail = domain.Mail{
	EntryID:    `demo@example.com\Facturas proveedores#0024`,
	Subject:    "Factura F001-00100",
	Sender:     "facturacion@proveedor.example",
	ReceivedAt: time.Date(2026, 11, 18, 9, 32, 0, 0, time.Local),
	IsMail:     true,
}

func content(t *testing.T, name string) []byte {
	t.Helper()
	b, err := fileContent(testMail, FixtureAttachment{FileName: name, Kind: domain.KindFile})
	if err != nil {
		t.Fatalf("fileContent(%s): %v", name, err)
	}
	if len(b) == 0 {
		t.Fatalf("fileContent(%s) vacío", name)
	}
	return b
}

// checkPDF verifica la estructura mínima que exige un lector: cabecera, tabla xref con los
// desplazamientos reales de cada objeto, startxref correcto y %%EOF.
func checkPDF(t *testing.T, b []byte) {
	t.Helper()
	s := string(b)
	if !strings.HasPrefix(s, "%PDF-1.") || !strings.HasSuffix(strings.TrimSpace(s), "%%EOF") {
		t.Fatal("falta la cabecera PDF o la marca EOF final")
	}
	m := regexp.MustCompile(`startxref\s+(\d+)\s+%%EOF`).FindStringSubmatch(s)
	if m == nil {
		t.Fatal("falta startxref")
	}
	xref, _ := strconv.Atoi(m[1])
	if !strings.HasPrefix(s[xref:], "xref") {
		t.Fatalf("startxref %d no apunta a la tabla xref", xref)
	}
	entries := regexp.MustCompile(`(\d{10}) 00000 n `).FindAllStringSubmatch(s[xref:], -1)
	if len(entries) < 4 {
		t.Fatalf("la tabla xref tiene %d objetos", len(entries))
	}
	for i, e := range entries {
		off, _ := strconv.Atoi(e[1])
		if want := fmt.Sprintf("%d 0 obj", i+1); !strings.HasPrefix(s[off:], want) {
			t.Fatalf("xref del objeto %d apunta a %q", i+1, s[off:off+12])
		}
	}
}

func TestInvoicePDF(t *testing.T) {
	b := content(t, "factura_F001-00100.pdf")
	checkPDF(t, b)
	inv := invoiceFor(testMail, "factura_F001-00100")
	for _, want := range []string{"F001-00100", inv.Supplier, formatMoney(inv.Total())} {
		if !bytes.Contains(b, toWinAnsi(want)) {
			t.Errorf("el PDF no contiene %q", want)
		}
	}
}

func TestInvoiceXMLMatchesPDF(t *testing.T) {
	var doc struct {
		ID    string `xml:"ID"`
		Total string `xml:"LegalMonetaryTotal>PayableAmount"`
		Lines []struct {
			Qty string `xml:"InvoicedQuantity"`
		} `xml:"InvoiceLine"`
	}
	if err := xml.Unmarshal(content(t, "factura_F001-00100.xml"), &doc); err != nil {
		t.Fatal(err)
	}
	inv := invoiceFor(testMail, "factura_F001-00100")
	if doc.ID != "F001-00100" || doc.Total != fmt.Sprintf("%.2f", inv.Total()) || len(doc.Lines) != len(inv.Lines) {
		t.Fatalf("XML = %+v, factura = %+v", doc, inv)
	}
}

func TestInvoiceDeterministic(t *testing.T) {
	a, b := invoiceFor(testMail, "factura_F001-00100"), invoiceFor(testMail, "factura_F001-00100")
	if a.Total() != b.Total() || len(a.Lines) == 0 || a.Total() <= 0 {
		t.Fatalf("factura no determinista o vacía: %v vs %v", a.Total(), b.Total())
	}
	if c := invoiceFor(testMail, "factura_F001-00101"); c.Total() == a.Total() && c.Lines[0] == a.Lines[0] {
		t.Fatal("dos facturas distintas generaron los mismos datos")
	}
}

func TestReportPDF(t *testing.T) {
	checkPDF(t, content(t, "3_informe.pdf"))
}

func TestStatementXLSX(t *testing.T) {
	f, err := excelize.OpenReader(bytes.NewReader(content(t, "estado_cuenta_agosto.xlsx")))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := f.GetRows(f.GetSheetList()[0])
	if err != nil || len(rows) < 5 {
		t.Fatalf("filas = %d, %v", len(rows), err)
	}
}

func TestDOCX(t *testing.T) {
	b := content(t, "consulta_09.docx")
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	var doc string
	for _, f := range z.File {
		if f.Name == "word/document.xml" {
			r, _ := f.Open()
			raw, _ := io.ReadAll(r)
			doc = string(raw)
		}
	}
	if !strings.Contains(doc, "<w:t") || !strings.Contains(doc, "Factura F001-00100") {
		t.Fatalf("word/document.xml sin el texto esperado: %.120q", doc)
	}
}

func TestImages(t *testing.T) {
	if _, err := png.Decode(bytes.NewReader(content(t, "image001.png"))); err != nil {
		t.Errorf("png: %v", err)
	}
	if _, err := jpeg.Decode(bytes.NewReader(content(t, "2_foto.jpg"))); err != nil {
		t.Errorf("jpeg: %v", err)
	}
}
