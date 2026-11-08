import type { Cycle, Point } from '../api.ts'
import { formatChange } from '../format.ts'

const colors = { up: 'text-red-600', down: 'text-blue-600', same: 'text-slate-400' } as const
const spoken = { up: '상승', down: '하락', same: '변동 없음' } as const

/** 증감 한 줄(50.web 2장 카드 구성, 3장). 직전 값이 없으면 아무것도 보이지 않는다. */
export function ChangeLine(props: {
  cycle: Cycle
  previous: Point | null
  change: number | null
  unit: string | null
  decimals: number | null
  className?: string
}) {
  const v = formatChange(props.cycle, props.previous, props.change, props.unit, props.decimals)
  if (!v) {
    return null
  }
  return (
    <p className={`text-sm ${props.className ?? ''}`}>
      <span className="text-slate-500">{v.label}</span>{' '}
      <span className={`font-semibold tabular-nums ${colors[v.direction]}`}>
        <span aria-hidden="true">{v.text}</span>
        <span className="sr-only">
          {v.direction === 'same' ? spoken.same : `${v.text.slice(1)} ${spoken[v.direction]}`}
        </span>
      </span>
    </p>
  )
}
