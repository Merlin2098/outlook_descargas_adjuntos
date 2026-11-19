import {JobState} from '../lib/job'

interface Props {
  job: JobState
  onCancel: () => Promise<void>
}

export function ProgressScreen(_props: Props) {
  return <main className="screen-main"/>
}
