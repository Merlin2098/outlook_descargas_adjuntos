package main

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"outlook_descargas_adjuntos/internal/domain"
)

func TestToDomain(t *testing.T) {
	d := JobRequestDTO{
		Folder:     domain.FolderRef{StoreID: "s", EntryID: "e", Path: "p"},
		From:       "2026-08-01",
		To:         "2026-10-31",
		Extensions: []string{" PDF"},
		DestDir:    `C:\x`,
		Organize:   "by_month",
		SkipInline: true,
	}
	r, err := d.ToDomain()
	if err != nil {
		t.Fatal(err)
	}
	if r.Filters.From.Location() != time.Local || r.Filters.From.Day() != 1 || r.Filters.To.Month() != time.October {
		t.Fatalf("fechas = %v %v", r.Filters.From, r.Filters.To)
	}
	if !reflect.DeepEqual(r.Filters.Extensions, []string{".pdf"}) {
		t.Fatalf("extensiones = %v", r.Filters.Extensions)
	}
	if r.Organize != domain.OrganizeByMonth || !r.Filters.SkipInline || r.DestDir != `C:\x` {
		t.Fatalf("request = %+v", r)
	}
}

func TestToDomainBadDate(t *testing.T) {
	_, err := JobRequestDTO{From: "31/10/2026", To: "2026-10-31"}.ToDomain()
	if err == nil || !strings.Contains(err.Error(), "Fecha inválida") {
		t.Fatalf("err = %v", err)
	}
}

func TestSelectSource(t *testing.T) {
	src, err := selectSource([]string{"app", "--demo"})
	if err != nil || src.Name() != "demo" {
		t.Fatalf("con --demo: %v", err)
	}
	if _, err := selectSource([]string{"app"}); err == nil {
		t.Fatal("sin --demo debería fallar hasta el Plan 3")
	}
}
