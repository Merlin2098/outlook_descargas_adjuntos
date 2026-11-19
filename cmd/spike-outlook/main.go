//go:build windows

// spike-outlook mide el recorrido de una carpeta real de Outlook Classic con la fuente COM de la app.
// Lo ejecuta el usuario en su equipo:
//
//	go run ./cmd/spike-outlook -n 10000
//	go run ./cmd/spike-outlook -n 10000 -folder "cuenta\Bandeja de entrada"
//
// Solo imprime conteos y tiempos: nunca asuntos, remitentes, nombres de carpeta ni de archivo.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"outlook_descargas_adjuntos/internal/domain"
	"outlook_descargas_adjuntos/internal/outlook"
	"outlook_descargas_adjuntos/internal/source"
)

func main() {
	n := flag.Int("n", 10000, "máximo de items a recorrer")
	folder := flag.String("folder", "", `ruta legible de la carpeta (cuenta\Carpeta\Sub); vacío = la subcarpeta con más items de la primera cuenta`)
	flag.Parse()
	if err := run(*n, *folder); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(limit int, path string) error {
	src, err := outlook.New(outlook.Options{})
	if err != nil {
		return err
	}
	defer src.Close()
	ctx := context.Background()

	t0 := time.Now()
	roots, err := src.ListRoots(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("raíces (cuentas y PST): %d  [%v]\n", len(roots), time.Since(t0).Round(time.Millisecond))

	target, err := pickFolder(ctx, src, roots, path)
	if err != nil {
		return err
	}
	fmt.Printf("carpeta elegida: %d items según Outlook\n", target.ItemCount)
	fmt.Println("Observa la memoria de OUTLOOK.EXE en el Administrador de tareas durante la corrida (B11).")

	var mem0, mem1 runtime.MemStats
	runtime.ReadMemStats(&mem0)
	items, mails, atts, readErrs := 0, 0, 0, 0
	t1 := time.Now()
	err = src.Scan(ctx, target.Ref, func(it source.ScanItem) error {
		items++
		switch {
		case it.Err != nil:
			readErrs++
		case it.Mail.IsMail:
			mails++
			atts += len(it.Mail.Attachments)
		}
		if items%1000 == 0 {
			fmt.Printf("  %d items…  %.0f items/s\n", items, float64(items)/time.Since(t1).Seconds())
		}
		if items >= limit {
			return source.ErrStopScan
		}
		return nil
	})
	if err != nil {
		return err
	}
	el := time.Since(t1)
	runtime.ReadMemStats(&mem1)
	fmt.Printf("recorridos: %d items (%d correos, %d adjuntos, %d errores de lectura)\n", items, mails, atts, readErrs)
	fmt.Printf("tiempo: %v  →  %.0f items/s\n", el.Round(time.Millisecond), float64(items)/el.Seconds())
	fmt.Printf("heap Go: %.1f MB → %.1f MB\n", float64(mem0.HeapAlloc)/1e6, float64(mem1.HeapAlloc)/1e6)
	return nil
}

// pickFolder busca la carpeta por ruta legible o, sin ruta, la subcarpeta con más items de la primera raíz.
func pickFolder(ctx context.Context, src *outlook.Source, roots []domain.Folder, path string) (domain.Folder, error) {
	if len(roots) == 0 {
		return domain.Folder{}, errors.New("el perfil no tiene cuentas")
	}
	if path == "" {
		kids, err := src.ListChildren(ctx, roots[0].Ref)
		if err != nil {
			return domain.Folder{}, err
		}
		best := roots[0]
		for _, k := range kids {
			if k.ItemCount > best.ItemCount {
				best = k
			}
		}
		return best, nil
	}
	parts := strings.Split(strings.Trim(path, `\`), `\`)
	level := roots
	var found domain.Folder
	for i, p := range parts {
		ok := false
		for _, f := range level {
			if strings.EqualFold(f.Name, p) {
				found, ok = f, true
				break
			}
		}
		if !ok {
			return domain.Folder{}, fmt.Errorf("no se encontró el segmento %d de la ruta", i+1)
		}
		if i < len(parts)-1 {
			kids, err := src.ListChildren(ctx, found.Ref)
			if err != nil {
				return domain.Folder{}, err
			}
			level = kids
		}
	}
	return found, nil
}
