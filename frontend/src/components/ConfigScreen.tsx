import {FormState} from '../lib/request'

interface Props {
  form: FormState
  onChange: (f: FormState) => void
  onStarted: (jobId: string) => void
}

export function ConfigScreen(_props: Props) {
  return <main className="screen-main"/>
}
