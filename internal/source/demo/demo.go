// Package demo implementa una fuente de correos simulada, en memoria y determinista.
// Sirve para los tests, para desarrollar la UI y para grabar el video sin datos reales.
package demo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"outlook_descargas_adjuntos/internal/domain"
	"outlook_descargas_adjuntos/internal/source"
)

// Options controla la latencia simulada (cero en los tests).
type Options struct {
	ScanDelay time.Duration
	SaveDelay time.Duration
}

type node struct {
	folder   domain.Folder
	items    []FixtureItem
	children []*node
}

// Source es la fuente demo. Es de solo lectura después de New y segura para uso concurrente.
type Source struct {
	opts    Options
	roots   []*node
	byPath  map[string]*node
	byEntry map[string]FixtureItem
}

var _ source.MailSource = (*Source)(nil)

// New construye la fuente a partir de un fixture.
func New(f Fixture, o Options) *Source {
	s := &Source{opts: o, byPath: map[string]*node{}, byEntry: map[string]FixtureItem{}}
	for _, st := range f.Stores {
		s.roots = append(s.roots, s.build(st, st.Name, ""))
	}
	return s
}

func (s *Source) build(f FixtureFolder, store, parent string) *node {
	path := f.Name
	if parent != "" {
		path = parent + `\` + f.Name
	}
	n := &node{folder: domain.Folder{
		Ref:         domain.FolderRef{StoreID: store, EntryID: path, Path: path},
		Name:        f.Name,
		ItemCount:   len(f.Items),
		HasChildren: len(f.Children) > 0,
	}}
	for i, it := range f.Items {
		it.Mail.EntryID = fmt.Sprintf("%s#%04d", path, i+1)
		it.Mail.Attachments = nil
		for k, a := range it.Files {
			it.Mail.Attachments = append(it.Mail.Attachments, domain.Attachment{
				Index: k + 1, FileName: a.FileName, Size: a.Size, Kind: a.Kind})
		}
		n.items = append(n.items, it)
		s.byEntry[it.Mail.EntryID] = it
	}
	for _, c := range f.Children {
		n.children = append(n.children, s.build(c, store, path))
	}
	s.byPath[path] = n
	return n
}

// Name implementa source.MailSource.
func (s *Source) Name() string { return "demo" }

// ListRoots implementa source.MailSource.
func (s *Source) ListRoots(ctx context.Context) ([]domain.Folder, error) {
	out := make([]domain.Folder, 0, len(s.roots))
	for _, r := range s.roots {
		out = append(out, r.folder)
	}
	return out, ctx.Err()
}

// ListChildren implementa source.MailSource.
func (s *Source) ListChildren(ctx context.Context, ref domain.FolderRef) ([]domain.Folder, error) {
	n, ok := s.byPath[ref.EntryID]
	if !ok {
		return nil, fmt.Errorf("carpeta demo inexistente: %s", ref.Path)
	}
	out := make([]domain.Folder, 0, len(n.children))
	for _, c := range n.children {
		out = append(out, c.folder)
	}
	return out, ctx.Err()
}

// Scan implementa source.MailSource.
func (s *Source) Scan(ctx context.Context, ref domain.FolderRef, fn func(source.ScanItem) error) error {
	n, ok := s.byPath[ref.EntryID]
	if !ok {
		return fmt.Errorf("carpeta demo inexistente: %s", ref.Path)
	}
	items := append([]FixtureItem(nil), n.items...)
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Mail.ReceivedAt.After(items[j].Mail.ReceivedAt)
	})
	for _, it := range items {
		if err := wait(ctx, s.opts.ScanDelay); err != nil {
			return err
		}
		si := source.ScanItem{Mail: it.Mail}
		if it.ReadErr != "" {
			si.Err = errors.New(it.ReadErr)
		}
		if err := fn(si); err != nil {
			if errors.Is(err, source.ErrStopScan) {
				return nil
			}
			return err
		}
	}
	return nil
}

// SaveAttachment implementa source.MailSource. Escribe un archivo válido y determinista (ver fileContent).
func (s *Source) SaveAttachment(ctx context.Context, mailEntryID string, index int, destPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	it, ok := s.byEntry[mailEntryID]
	if !ok || index < 1 || index > len(it.Files) {
		return fmt.Errorf("adjunto demo inexistente: %s #%d", mailEntryID, index)
	}
	if err := wait(ctx, s.opts.SaveDelay); err != nil {
		return err
	}
	a := it.Files[index-1]
	if a.FailWith != "" {
		return errors.New(a.FailWith)
	}
	data, err := fileContent(it.Mail, a)
	if err != nil {
		return fmt.Errorf("generar adjunto demo %s: %w", a.FileName, err)
	}
	return os.WriteFile(destPath, data, 0o644)
}

func wait(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
