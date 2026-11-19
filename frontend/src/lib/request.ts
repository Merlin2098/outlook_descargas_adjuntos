import {domain, main} from '../../wailsjs/go/models'
import {es} from '../i18n/es'
import {DateRange, daysInclusive} from './dates'

export type Organize = 'flat' | 'by_month'

/** Estado del formulario de la pantalla de configuración. */
export interface FormState {
  folder: domain.FolderRef | null
  range: DateRange
  destDir: string
  organize: Organize
  subjectContains: string
  senderContains: string
  extensions: string[]
  skipInline: boolean
  includeSubfolders: boolean
}

/** Validación previa al inicio: el mensaje que se muestra, o null si se puede iniciar. */
export function validateForm(f: FormState): string | null {
  if (!f.folder) return es.config.errors.noFolder
  if (!f.destDir.trim()) return es.config.errors.noDest
  if (daysInclusive(f.range) === 0) return es.config.errors.badRange
  return null
}

/** Arma la solicitud para StartJob. */
export function buildRequest(f: FormState): main.JobRequestDTO {
  return new main.JobRequestDTO({
    folder: f.folder,
    from: f.range.from,
    to: f.range.to,
    subjectContains: f.subjectContains.trim(),
    senderContains: f.senderContains.trim(),
    extensions: f.extensions,
    skipInline: f.skipInline,
    includeSubfolders: f.includeSubfolders,
    destDir: f.destDir.trim(),
    organize: f.organize,
  })
}
