// 화면에 보이는 글을 만든다. React와 상관없이 시험할 수 있게 따로 둔다.
import type { Cycle, Point } from './api.ts'

/**
 * 기간 시작일(YYYY-MM-DD)을 주기에 맞춰 보여준다(40.api 1.1).
 * D 2026-10-29, M 2026-09, Q 2025 Q2, A 2025. 형식이 다르면 받은 글 그대로다.
 */
export function formatPeriod(period: string, cycle: Cycle): string {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(period)
  if (!m) {
    return period
  }
  const [, year, month] = m
  switch (cycle) {
    case 'D':
      return period
    case 'M':
      return `${year}-${month}`
    case 'Q':
      return `${year} Q${Math.floor((Number(month) - 1) / 3) + 1}`
    case 'A':
      return year
    default:
      return period
  }
}

const numberFormat = new Intl.NumberFormat('ko-KR', { maximumFractionDigits: 4 })

// 소수 자릿수 0~4마다 하나씩 만들어 둔다.
const fixedFormats = [0, 1, 2, 3, 4].map(
  (d) => new Intl.NumberFormat('ko-KR', { minimumFractionDigits: d, maximumFractionDigits: d }),
)

/**
 * 값을 천 단위 쉼표와 함께 보여준다.
 * decimals(0~4)가 있으면 그 자릿수로 맞추고, null이면 받은 값 그대로(소수 4자리까지)다(40.api 1.1).
 */
export function formatValue(value: number, decimals: number | null = null): string {
  const fixed = decimals === null ? undefined : fixedFormats[decimals]
  return (fixed ?? numberFormat).format(value)
}

/** 값과 단위. 단위가 비어 있으면 숫자만이다. */
export function formatValueUnit(value: number, unit: string | null, decimals: number | null = null): string {
  const v = formatValue(value, decimals)
  return unit ? `${v} ${unit}` : v
}

const changeLabels: Record<Cycle, string> = { M: '전월 대비', Q: '전기 대비', A: '전년 대비', D: '직전 대비' }

/** 증감 표시(50.web 2장 카드 구성, 3장). up은 빨강 ▲, down은 파랑 ▼, same은 회색 "-"다. */
export interface ChangeView {
  label: string
  direction: 'up' | 'down' | 'same'
  text: string
}

/**
 * 증감 한 줄(예: 전월 대비 ▲0.20 연%). previous나 change가 null이면 보이지 않으므로 null이다(40.api 2.1).
 * 숫자는 지표의 소수 자릿수와 단위를 따른다. 그 자릿수로 0이 되면(예: 0.001을 2자리로) 같은 것으로 본다.
 * 단위가 %로 끝나면(%, 연% 등) 증감은 비율 변화가 아니라 차이이므로 %p로 쓴다(예: ▼0.1%p, Issue #77).
 * change는 API가 준 값을 그대로 쓰고 화면에서 다시 빼지 않는다.
 */
export function formatChange(
  cycle: Cycle,
  previous: Point | null,
  change: number | null,
  unit: string | null,
  decimals: number | null = null,
): ChangeView | null {
  if (previous === null || change === null) {
    return null
  }
  const label = changeLabels[cycle] ?? '직전 대비'
  const size = Math.abs(change)
  if (formatValue(size, decimals) === formatValue(0, decimals)) {
    return { label, direction: 'same', text: '-' }
  }
  const mark = change > 0 ? '▲' : '▼'
  const text = isPercent(unit) ? `${formatValue(size, decimals)}%p` : formatValueUnit(size, unit, decimals)
  return { label, direction: change > 0 ? 'up' : 'down', text: mark + text }
}

/** 단위가 %로 끝나는지(%, 연% 등). 증감을 %p로 쓸지 정한다. */
function isPercent(unit: string | null): boolean {
  return unit?.trim().endsWith('%') ?? false
}

const cycleNames: Record<Cycle, string> = { D: '일간', M: '월간', Q: '분기', A: '연간' }

/** 주기 이름(월간 등). 모르는 값은 그대로다. */
export function formatCycle(cycle: Cycle): string {
  return cycleNames[cycle] ?? cycle
}

const dateTimeFormat = new Intl.DateTimeFormat('en-CA', {
  timeZone: 'Asia/Seoul',
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  hourCycle: 'h23',
})

/** RFC 3339 시각을 Asia/Seoul 기준 YYYY-MM-DD HH:mm로 보여준다. 읽을 수 없으면 받은 글 그대로다. */
export function formatDateTime(value: string): string {
  const t = new Date(value)
  if (Number.isNaN(t.getTime())) {
    return value
  }
  const p = Object.fromEntries(dateTimeFormat.formatToParts(t).map((x) => [x.type, x.value]))
  return `${p.year}-${p.month}-${p.day} ${p.hour}:${p.minute}`
}
