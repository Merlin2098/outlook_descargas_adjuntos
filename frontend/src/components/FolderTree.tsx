import {useCallback, useEffect, useRef, useState} from 'react'
import {ListChildren, ListRoots} from '../../wailsjs/go/main/App'
import {domain} from '../../wailsjs/go/models'
import {es} from '../i18n/es'
import {formatInt} from '../lib/format'
import {Icon} from './Icon'

type Children = {state: 'loading'} | {state: 'error'; message: string} | {state: 'ok'; items: domain.Folder[]}

const keyOf = (r: domain.FolderRef) => `${r.storeId}|${r.entryId}`
const errText = (e: unknown) => (e instanceof Error ? e.message : String(e))

interface Props {
  selected: domain.FolderRef | null
  onSelect: (f: domain.Folder) => void
}

/** Árbol de cuentas y carpetas. Los hijos se piden al expandir, una sola vez por carpeta. */
export function FolderTree({selected, onSelect}: Props) {
  const [roots, setRoots] = useState<Children>({state: 'loading'})
  const [children, setChildren] = useState<Record<string, Children>>({})
  const [expanded, setExpanded] = useState<Set<string>>(new Set())
  const childrenRef = useRef(children)
  childrenRef.current = children

  const loadChildren = useCallback((f: domain.Folder) => {
    const k = keyOf(f.ref)
    setChildren((c) => ({...c, [k]: {state: 'loading'}}))
    ListChildren(f.ref)
      .then((items) => setChildren((c) => ({...c, [k]: {state: 'ok', items: items ?? []}})))
      .catch((e: unknown) => setChildren((c) => ({...c, [k]: {state: 'error', message: errText(e)}})))
  }, [])

  const toggle = useCallback((f: domain.Folder) => {
    const k = keyOf(f.ref)
    setExpanded((s) => {
      const n = new Set(s)
      if (n.has(k)) n.delete(k)
      else n.add(k)
      return n
    })
    const c = childrenRef.current[k]
    if (!c || c.state === 'error') loadChildren(f)
  }, [loadChildren])

  const reload = useCallback(() => {
    setRoots({state: 'loading'})
    setChildren({})
    setExpanded(new Set())
    ListRoots()
      .then((items) => {
        const list = items ?? []
        setRoots({state: 'ok', items: list})
        if (list[0]?.hasChildren) {
          setExpanded(new Set([keyOf(list[0].ref)]))
          loadChildren(list[0])
        }
      })
      .catch((e: unknown) => setRoots({state: 'error', message: errText(e)}))
  }, [loadChildren])

  useEffect(reload, [reload])

  const selectedKey = selected ? keyOf(selected) : null

  const renderList = (list: Children, depth: number) => {
    if (list.state === 'loading') return <div className="tree-note" style={{paddingLeft: 20 + depth * 20}}>{es.config.tree.loading}</div>
    if (list.state === 'error') return <div className="tree-note tree-error" style={{paddingLeft: 20 + depth * 20}}>{list.message}</div>
    if (depth > 0 && list.items.length === 0) return null
    return list.items.map((f) => renderNode(f, depth))
  }

  const renderNode = (f: domain.Folder, depth: number) => {
    const k = keyOf(f.ref)
    const open = expanded.has(k)
    const chevron = f.hasChildren
      ? <button className="tree-chevron" aria-label={open ? es.config.tree.collapse : es.config.tree.expand}
                aria-expanded={open} onClick={(e) => { e.stopPropagation(); toggle(f) }}>
          <Icon name={open ? 'chevronDown' : 'chevronRight'} size={depth === 0 ? 14 : 12}/>
        </button>
      : <span className="tree-chevron"/>
    const sub = open && children[k] ? renderList(children[k], depth + 1) : null
    if (depth === 0) {
      return (
        <div key={k} role="group">
          <div className="tree-root" onClick={() => f.hasChildren && toggle(f)}>
            {chevron}
            <Icon name={f.ref.storeId === roots0Store(roots) ? 'mail' : 'database'} size={16} strokeWidth={1.8}
                  className={f.ref.storeId === roots0Store(roots) ? 'tree-icon-primary' : 'tree-icon'}/>
            <span className="tree-label">{f.name}</span>
          </div>
          {sub}
        </div>
      )
    }
    const isSel = k === selectedKey
    return (
      <div key={k} role="group">
        <div role="treeitem" aria-selected={isSel} tabIndex={0}
             className={`tree-row${isSel ? ' selected' : ''}`} style={{paddingLeft: 20 + (depth - 1) * 20}}
             onClick={() => onSelect(f)}
             onDoubleClick={() => f.hasChildren && toggle(f)}
             onKeyDown={(e) => {
               if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onSelect(f) }
               if (e.key === 'ArrowRight' && f.hasChildren && !open) toggle(f)
               if (e.key === 'ArrowLeft' && open) toggle(f)
             }}>
          {chevron}
          <span className="tree-label">{f.name}</span>
          <span className="mono tree-count">{formatInt(f.itemCount)}</span>
        </div>
        {sub}
      </div>
    )
  }

  return (
    <>
      <div className="aside-head">
        <span className="section-title">{es.config.tree.title}</span>
        <button className="icon-btn" aria-label={es.config.tree.reload} onClick={reload}>
          <Icon name="refresh" size={14} strokeWidth={2.2}/>
        </button>
      </div>
      <div className="tree" role="tree">{renderList(roots, 0)}</div>
    </>
  )
}

/** Store de la primera raíz: se dibuja como buzón; las demás, como archivo de datos (PST). */
function roots0Store(roots: Children): string | null {
  return roots.state === 'ok' && roots.items[0] ? roots.items[0].ref.storeId : null
}
