import {JobState} from '../lib/job'

interface Props {
  job: JobState
  onNew: () => void
}

export function SummaryScreen(_props: Props) {
  return <main className="screen-main"/>
}
