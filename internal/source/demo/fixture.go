package demo

import (
	"fmt"
	"math/rand/v2"
	"time"

	"outlook_descargas_adjuntos/internal/domain"
)

// FixtureAttachment es un adjunto simulado. FailWith != "" hace fallar SaveAttachment con ese texto.
type FixtureAttachment struct {
	FileName string
	Size     int64
	Kind     domain.AttachmentKind
	FailWith string
}

// FixtureItem es un elemento simulado. ReadErr != "" simula un elemento ilegible.
type FixtureItem struct {
	Mail    domain.Mail
	ReadErr string
	Files   []FixtureAttachment
}

// FixtureFolder es una carpeta simulada con sus elementos y subcarpetas.
type FixtureFolder struct {
	Name     string
	Items    []FixtureItem
	Children []FixtureFolder
}

// Fixture es el árbol completo: el nivel raíz son las cuentas o stores.
type Fixture struct {
	Stores []FixtureFolder
}

const demoWindow = 120 * 24 * time.Hour

type gen struct {
	r   *rand.Rand
	now time.Time
}

func (g gen) date() time.Time {
	d := time.Duration(g.r.Int64N(int64(demoWindow)))
	return g.now.Add(-d).Truncate(time.Second)
}

func (g gen) dateIn(year int) time.Time {
	start := time.Date(year, 1, 1, 8, 0, 0, 0, time.Local)
	return start.Add(time.Duration(g.r.Int64N(int64(364 * 24 * time.Hour)))).Truncate(time.Second)
}

func (g gen) size(minKB, maxKB int64) int64 {
	return (minKB + g.r.Int64N(maxKB-minKB+1)) * 1024
}

var (
	genericSubjects = []string{"Informe semanal", "Acta de reunión", "Propuesta comercial", "Cotización solicitada",
		"Documentos del proyecto", "Planilla actualizada", "Presentación final", "Contrato revisado"}
	genericSenders = []string{"ana.torres@cliente.example", "jorge.rios@proveedor.example",
		"equipo@consultora.example", "admin@obra.example", "rrhh@empresa.example"}
	genericFiles = []struct {
		name string
		min  int64
		max  int64
	}{{"informe.pdf", 60, 900}, {"planilla.xlsx", 20, 300}, {"documento.docx", 30, 400}, {"foto.jpg", 200, 2400}}
	months = []string{"enero", "febrero", "marzo", "abril", "mayo", "junio", "julio", "agosto",
		"septiembre", "octubre", "noviembre", "diciembre"}
)

// genericFolder crea n correos variados con fechas recientes (o del año indicado si year != 0).
func (g gen) genericFolder(name string, n int, year int) FixtureFolder {
	f := FixtureFolder{Name: name}
	for i := 0; i < n; i++ {
		d := g.date()
		if year != 0 {
			d = g.dateIn(year)
		}
		it := FixtureItem{Mail: domain.Mail{
			Subject:    genericSubjects[g.r.IntN(len(genericSubjects))],
			Sender:     genericSenders[g.r.IntN(len(genericSenders))],
			ReceivedAt: d,
			IsMail:     true,
		}}
		for k := g.r.IntN(4); k > 0; k-- {
			gf := genericFiles[g.r.IntN(len(genericFiles))]
			it.Files = append(it.Files, FixtureAttachment{
				FileName: fmt.Sprintf("%d_%s", i+1, gf.name), Size: g.size(gf.min, gf.max), Kind: domain.KindFile})
		}
		if g.r.IntN(5) == 0 {
			it.Files = append(it.Files, FixtureAttachment{FileName: "image001.png", Size: g.size(4, 12), Kind: domain.KindInline})
		}
		f.Items = append(f.Items, it)
	}
	return f
}

// facturas reproduce la carpeta "Facturas proveedores" del mock, con 312 elementos
// e incluye a propósito los casos difíciles.
func (g gen) facturas() FixtureFolder {
	f := FixtureFolder{Name: "Facturas proveedores"}
	add := func(it FixtureItem) {
		it.Mail.ReceivedAt = g.date()
		it.Mail.IsMail = true
		f.Items = append(f.Items, it)
	}
	for i := 0; i < 3; i++ {
		it := FixtureItem{Mail: domain.Mail{Subject: "Invitación: revisión mensual", Sender: "agenda@proveedor.example"}}
		it.Mail.ReceivedAt = g.date()
		f.Items = append(f.Items, it) // IsMail=false
	}
	f.Items = append(f.Items, FixtureItem{
		Mail:    domain.Mail{Subject: "(ilegible)", ReceivedAt: g.date(), IsMail: true},
		ReadErr: "propiedad no disponible",
	})
	for i := 0; i < 4; i++ {
		add(FixtureItem{Mail: domain.Mail{Subject: "Confirmación de recepción", Sender: "facturacion@proveedor.example"},
			Files: []FixtureAttachment{{FileName: "image001.png", Size: g.size(4, 12), Kind: domain.KindInline}}})
	}
	for i := 0; i < 15; i++ {
		add(FixtureItem{Mail: domain.Mail{Subject: "Re: consulta de pago", Sender: "cobranzas@proveedor.example"},
			Files: []FixtureAttachment{{FileName: fmt.Sprintf("consulta_%02d.docx", i+1), Size: g.size(20, 80), Kind: domain.KindFile}}})
	}
	const invoices = 312 - 3 - 1 - 4 - 15
	for i := 0; i < invoices; i++ {
		num := 100 + i
		if i == 200 {
			num = 223 // nombre duplicado a propósito con el correo de la factura 223
		}
		base := fmt.Sprintf("factura_F001-%05d", num)
		it := FixtureItem{
			Mail: domain.Mail{Subject: fmt.Sprintf("Factura F001-%05d", num), Sender: "facturacion@proveedor.example"},
			Files: []FixtureAttachment{
				{FileName: base + ".pdf", Size: g.size(80, 400), Kind: domain.KindFile},
				{FileName: base + ".xml", Size: g.size(8, 20), Kind: domain.KindFile},
			},
		}
		if g.r.IntN(100) < 15 {
			m := months[g.r.IntN(len(months))]
			it.Files = append(it.Files, FixtureAttachment{FileName: "estado_cuenta_" + m + ".xlsx", Size: g.size(30, 120), Kind: domain.KindFile})
		}
		switch i {
		case 10:
			it.Files[1] = FixtureAttachment{FileName: "herramienta.exe", Size: g.size(100, 200), Kind: domain.KindFile, FailWith: "Adjunto bloqueado por Outlook"}
		case 20:
			it.Files[1].FailWith = "Error de E/S simulado"
		}
		add(it)
	}
	return f
}

// DefaultFixture reproduce el árbol del mock de UI. Es determinista para el mismo now y seed.
func DefaultFixture(now time.Time, seed int64) Fixture {
	g := gen{r: rand.New(rand.NewPCG(uint64(seed), 0)), now: now}
	proyectos := FixtureFolder{Name: "Proyectos", Children: []FixtureFolder{
		g.genericFolder("Obra Norte", 86, 0),
		g.genericFolder("Obra Sur", 41, 0),
	}}
	cuenta := FixtureFolder{Name: "demo@example.com", Children: []FixtureFolder{
		g.genericFolder("Bandeja de entrada", 1248, 0),
		g.genericFolder("Elementos enviados", 402, 0),
		g.facturas(),
		proyectos,
		g.genericFolder("Archivo 2025", 2910, 2025),
	}}
	pst := FixtureFolder{Name: "Archivo local (PST)", Children: []FixtureFolder{
		g.genericFolder("Importados", 120, 0),
	}}
	return Fixture{Stores: []FixtureFolder{cuenta, pst}}
}
