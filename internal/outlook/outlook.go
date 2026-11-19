//go:build windows

package outlook

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"

	"outlook_descargas_adjuntos/internal/domain"
	"outlook_descargas_adjuntos/internal/source"
)

// Propiedades MAPI usadas para detectar adjuntos inline.
const (
	propAttachmentHidden = "http://schemas.microsoft.com/mapi/proptag/0x7FFE000B" // PR_ATTACHMENT_HIDDEN
	propAttachContentID  = "http://schemas.microsoft.com/mapi/proptag/0x3712001F" // PR_ATTACH_CONTENT_ID
)

// Códigos HRESULT que indican que Outlook se cerró o dejó de responder: se reconecta en la próxima llamada.
var disconnectedCodes = map[uintptr]bool{
	0x80010108: true, // RPC_E_DISCONNECTED
	0x800706BA: true, // RPC_S_SERVER_UNAVAILABLE
	0x800706BE: true, // RPC_S_CALL_FAILED
}

// Options configura la fuente. ProgID vacío = "Outlook.Application".
type Options struct {
	ProgID string
}

// Source implementa source.MailSource sobre Outlook Classic. Todos los campos COM se usan solo
// dentro del STA; los métodos públicos son seguros para llamar desde cualquier goroutine.
type Source struct {
	sta    *sta
	progID string

	// Solo dentro del STA.
	app, ns *ole.IDispatch
	stores  map[string]string // EntryID de correo → StoreID, para GetItemFromID
}

var _ source.MailSource = (*Source)(nil)

// New arranca el STA. No conecta con Outlook hasta la primera llamada, así la app abre rápido.
func New(o Options) (*Source, error) {
	st, err := newSTA()
	if err != nil {
		return nil, err
	}
	if o.ProgID == "" {
		o.ProgID = "Outlook.Application"
	}
	return &Source{sta: st, progID: o.ProgID, stores: map[string]string{}}, nil
}

// Name implementa source.MailSource.
func (s *Source) Name() string { return "outlook" }

// closeTimeout limita cuánto espera Close a que el STA quede libre (variable para los tests).
var closeTimeout = 2 * time.Second

// Close libera Outlook y termina el STA. Es idempotente. Si Outlook no responde (el STA sigue
// ocupado en una llamada COM, por ejemplo con un diálogo modal abierto), no bloquea el cierre de
// la app: abandona el STA, que termina solo cuando esa llamada vuelva o el proceso salga.
func (s *Source) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), closeTimeout)
	defer cancel()
	err := s.sta.do(ctx, func() error {
		s.disconnect()
		return nil
	})
	switch {
	case errors.Is(err, errSTAClosed):
		return nil
	case ctx.Err() != nil:
		s.sta.abandon()
		return fmt.Errorf("Outlook no respondió al cerrar la conexión: %w", ctx.Err()) //nolint:staticcheck // texto visible
	}
	s.sta.close()
	return err
}

// namespace devuelve el MAPI Namespace, conectando si hace falta (en el STA).
func (s *Source) namespace() (*ole.IDispatch, error) {
	if s.ns != nil {
		return s.ns, nil
	}
	unk, err := oleutil.GetActiveObject(s.progID)
	if err != nil {
		unk, err = oleutil.CreateObject(s.progID)
		if err != nil {
			return nil, connectError(err)
		}
	}
	defer unk.Release()
	app, err := unk.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return nil, connectError(err)
	}
	ns, err := callObj(app, "GetNamespace", "MAPI")
	if err != nil {
		release(app)
		return nil, connectError(err)
	}
	s.app, s.ns = app, ns
	return ns, nil
}

func (s *Source) disconnect() {
	release(s.ns, s.app)
	s.ns, s.app = nil, nil
}

// checkConn suelta la conexión si el error indica que Outlook se cerró, para reconectar después.
func (s *Source) checkConn(err error) error {
	var oe *ole.OleError
	if err != nil && errors.As(err, &oe) && disconnectedCodes[oe.Code()] {
		s.disconnect()
		return connectError(err)
	}
	return err
}

// ListRoots implementa source.MailSource: las cuentas y archivos de datos (PST) del perfil.
func (s *Source) ListRoots(ctx context.Context) ([]domain.Folder, error) {
	var out []domain.Folder
	err := s.sta.do(ctx, func() error {
		ns, err := s.namespace()
		if err != nil {
			return err
		}
		roots, err := getObj(ns, "Folders")
		if err != nil {
			return s.checkConn(err)
		}
		defer release(roots)
		out, err = listFolders(roots)
		return s.checkConn(err)
	})
	return out, err
}

// ListChildren implementa source.MailSource.
func (s *Source) ListChildren(ctx context.Context, ref domain.FolderRef) ([]domain.Folder, error) {
	var out []domain.Folder
	err := s.sta.do(ctx, func() error {
		f, err := s.folder(ref)
		if err != nil {
			return err
		}
		defer release(f)
		kids, err := getObj(f, "Folders")
		if err != nil {
			return s.checkConn(err)
		}
		defer release(kids)
		out, err = listFolders(kids)
		return s.checkConn(err)
	})
	return out, err
}

func (s *Source) folder(ref domain.FolderRef) (*ole.IDispatch, error) {
	ns, err := s.namespace()
	if err != nil {
		return nil, err
	}
	f, err := callObj(ns, "GetFolderFromID", ref.EntryID, ref.StoreID)
	if err != nil {
		return nil, s.checkConn(fmt.Errorf("abrir carpeta %s: %w", ref.Path, err))
	}
	return f, nil
}

// listFolders lee una colección Folders. Una carpeta ilegible (sin permisos, buzón compartido
// desconectado) se omite del árbol en lugar de vaciarlo.
func listFolders(coll *ole.IDispatch) ([]domain.Folder, error) {
	n, err := getInt(coll, "Count")
	if err != nil {
		return nil, err
	}
	out := make([]domain.Folder, 0, n)
	for i := 1; i <= n; i++ {
		f, err := getObj(coll, "Item", i)
		if err != nil {
			continue
		}
		info, err := folderInfo(f)
		release(f)
		if err == nil {
			out = append(out, info)
		}
	}
	return out, nil
}

func folderInfo(f *ole.IDispatch) (domain.Folder, error) {
	var info domain.Folder
	var err error
	if info.Ref.EntryID, err = getString(f, "EntryID"); err != nil {
		return info, err
	}
	if info.Ref.StoreID, err = getString(f, "StoreID"); err != nil {
		return info, err
	}
	if info.Name, err = getString(f, "Name"); err != nil {
		return info, err
	}
	if p, err := getString(f, "FolderPath"); err == nil {
		info.Ref.Path = folderPath(p)
	} else {
		info.Ref.Path = info.Name
	}
	info.ItemCount = countOf(f, "Items")
	info.HasChildren = countOf(f, "Folders") > 0
	return info, nil
}

// countOf devuelve Count de una colección hija, o 0 si no se puede leer.
func countOf(d *ole.IDispatch, coll string) int {
	c, err := getObj(d, coll)
	if err != nil {
		return 0
	}
	defer release(c)
	n, _ := getInt(c, "Count")
	return n
}

// Scan implementa source.MailSource: recorre la carpeta por ReceivedTime descendente. Los items se
// leen en el STA y se entregan uno a uno a fn en el goroutine que llama.
func (s *Source) Scan(ctx context.Context, ref domain.FolderRef, fn func(source.ScanItem) error) error {
	return stream(ctx, s.sta, func(yield func(source.ScanItem) bool) error {
		f, err := s.folder(ref)
		if err != nil {
			return err
		}
		defer release(f)
		items, err := getObj(f, "Items")
		if err != nil {
			return s.checkConn(err)
		}
		defer release(items)
		if err := call(items, "Sort", "[ReceivedTime]", true); err != nil {
			return s.checkConn(err)
		}
		it, err := callMaybeObj(items, "GetFirst")
		for err == nil && it != nil {
			si := s.readItem(it, ref.StoreID)
			release(it)
			if !yield(si) {
				return nil
			}
			it, err = callMaybeObj(items, "GetNext")
		}
		return s.checkConn(err)
	}, fn)
}

// readItem lee un item de la carpeta. Cualquier error queda en ScanItem.Err y el job sigue.
func (s *Source) readItem(it *ole.IDispatch, storeID string) source.ScanItem {
	class, err := getInt(it, "Class")
	if err != nil {
		return source.ScanItem{Err: s.checkConn(err)}
	}
	var m domain.Mail
	m.EntryID, _ = getString(it, "EntryID")
	m.Subject, _ = getString(it, "Subject")
	if m.ReceivedAt, err = getDate(it, "ReceivedTime"); err != nil {
		// Citas, tareas o informes pueden no tener ReceivedTime.
		if m.ReceivedAt, err = getDate(it, "CreationTime"); err != nil {
			return source.ScanItem{Mail: m, Err: s.checkConn(err)}
		}
	}
	if !isMailClass(class) {
		return source.ScanItem{Mail: m}
	}
	m.IsMail = true
	name, _ := getString(it, "SenderName")
	addr, _ := getString(it, "SenderEmailAddress")
	m.Sender = senderText(name, addr)
	if m.Attachments, err = readAttachments(it); err != nil {
		return source.ScanItem{Mail: m, Err: s.checkConn(err)}
	}
	if m.EntryID != "" {
		s.stores[m.EntryID] = storeID
	}
	return source.ScanItem{Mail: m}
}

func readAttachments(it *ole.IDispatch) ([]domain.Attachment, error) {
	atts, err := getObj(it, "Attachments")
	if err != nil {
		return nil, err
	}
	defer release(atts)
	n, err := getInt(atts, "Count")
	if err != nil || n == 0 {
		return nil, err
	}
	html := lazyHTML{it: it}
	out := make([]domain.Attachment, 0, n)
	for i := 1; i <= n; i++ {
		a, err := getObj(atts, "Item", i)
		if err != nil {
			return nil, err
		}
		att, err := readAttachment(a, i, &html)
		release(a)
		if err != nil {
			return nil, err
		}
		out = append(out, att)
	}
	return out, nil
}

// lazyHTML lee HTMLBody una sola vez y solo si algún adjunto tiene Content-ID.
type lazyHTML struct {
	it   *ole.IDispatch
	done bool
	body string
}

func (h *lazyHTML) contains(s string) bool {
	if !h.done {
		h.body, _ = getString(h.it, "HTMLBody")
		h.done = true
	}
	return strings.Contains(h.body, s)
}

func readAttachment(a *ole.IDispatch, index int, html *lazyHTML) (domain.Attachment, error) {
	typ, err := getInt(a, "Type")
	if err != nil {
		return domain.Attachment{}, err
	}
	name, err := getString(a, "FileName")
	if err != nil || name == "" {
		name, _ = getString(a, "DisplayName")
	}
	size, _ := getInt(a, "Size")
	var hidden bool
	var cid string
	if typ == olByValue {
		hidden, cid = inlineProps(a)
	}
	kind := classifyAttachment(typ, hidden, cid, cid != "" && html.contains("cid:"+cid))
	return domain.Attachment{Index: index, FileName: attachmentFileName(name, kind, index), Size: int64(size), Kind: kind}, nil
}

// inlineProps lee PR_ATTACHMENT_HIDDEN y PR_ATTACH_CONTENT_ID. Si no existen, Outlook lanza
// un error que aquí significa "no oculto" y "sin Content-ID".
func inlineProps(a *ole.IDispatch) (hidden bool, contentID string) {
	pa, err := getObj(a, "PropertyAccessor")
	if err != nil {
		return false, ""
	}
	defer release(pa)
	if v, err := oleutil.CallMethod(pa, "GetProperty", propAttachmentHidden); err == nil {
		hidden, _ = v.Value().(bool)
		_ = v.Clear()
	}
	if v, err := oleutil.CallMethod(pa, "GetProperty", propAttachContentID); err == nil {
		if v.VT == ole.VT_BSTR {
			contentID = strings.Trim(v.ToString(), "<>")
		}
		_ = v.Clear()
	}
	return hidden, contentID
}

// callMaybeObj llama un método que devuelve un objeto o nada (GetFirst/GetNext al final).
func callMaybeObj(d *ole.IDispatch, method string) (*ole.IDispatch, error) {
	v, err := oleutil.CallMethod(d, method)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", method, err)
	}
	if v.VT == ole.VT_EMPTY || v.VT == ole.VT_NULL || (v.VT == ole.VT_DISPATCH && v.Val == 0) {
		_ = v.Clear()
		return nil, nil
	}
	return dispatchOf(v, method)
}

// SaveAttachment implementa source.MailSource: relee el correo por EntryID (y su StoreID, si se
// conoce del Scan) y guarda el adjunto index en destPath.
func (s *Source) SaveAttachment(ctx context.Context, mailEntryID string, index int, destPath string) error {
	return s.sta.do(ctx, func() error {
		ns, err := s.namespace()
		if err != nil {
			return err
		}
		args := []any{mailEntryID}
		if store := s.stores[mailEntryID]; store != "" {
			args = append(args, store)
		}
		mail, err := callObj(ns, "GetItemFromID", args...)
		if err != nil {
			return s.checkConn(err)
		}
		defer release(mail)
		atts, err := getObj(mail, "Attachments")
		if err != nil {
			return s.checkConn(err)
		}
		defer release(atts)
		a, err := getObj(atts, "Item", index)
		if err != nil {
			return s.checkConn(err)
		}
		defer release(a)
		return s.checkConn(call(a, "SaveAsFile", destPath))
	})
}
