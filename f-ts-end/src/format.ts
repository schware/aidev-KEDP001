// 화면에 보이는 글을 만든다. React와 상관없이 시험할 수 있게 따로 둔다.
import type { Cycle } from './api.ts'

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

/** 값을 천 단위 쉼표와 함께 보여준다. 소수는 4자리까지다. */
export function formatValue(value: number): string {
  return numberFormat.format(value)
}

/** 값과 단위. 단위가 비어 있으면 숫자만이다. */
export function formatValueUnit(value: number, unit: string | null): string {
  const v = formatValue(value)
  return unit ? `${v} ${unit}` : v
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
