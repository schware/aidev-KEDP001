// 차트 기간: 화면은 숫자 + 단위(년/개월/일)로 받고, API는 `N years`, `N months`, `N days`로 주고받는다(40.api 5.4, 5.5).

export type PeriodUnit = 'years' | 'months' | 'days'

export interface ChartPeriod {
  n: number
  unit: PeriodUnit
}

export const periodUnits: { value: PeriodUnit; label: string }[] = [
  { value: 'years', label: '년' },
  { value: 'months', label: '개월' },
  { value: 'days', label: '일' },
]

const unitLabels: Record<PeriodUnit, string> = { years: '년', months: '개월', days: '일' }

// PostgreSQL interval 글(`1 year`, `6 mons`)도 읽는다.
const units: Record<string, PeriodUnit> = {
  year: 'years',
  years: 'years',
  month: 'months',
  months: 'months',
  mon: 'months',
  mons: 'months',
  day: 'days',
  days: 'days',
}

/** `10 years` → {n: 10, unit: 'years'}. 읽을 수 없으면 null이다. */
export function parseChartPeriod(text: string | null): ChartPeriod | null {
  const m = /^\s*(\d+)\s+([a-z]+)\s*$/.exec(text ?? '')
  if (!m) {
    return null
  }
  const unit = units[m[2]]
  const n = Number(m[1])
  return unit && n > 0 ? { n, unit } : null
}

/** {n: 10, unit: 'years'} → `10 years`. 1이어도 복수형이다(40.api 5.4). */
export function formatChartPeriod(p: ChartPeriod): string {
  return `${p.n} ${p.unit}`
}

/** 표에 보일 글. `10 years` → `10년`. 없으면 `-`, 읽을 수 없으면 받은 글 그대로다. */
export function chartPeriodLabel(text: string | null): string {
  if (text === null) {
    return '-'
  }
  const p = parseChartPeriod(text)
  return p ? `${p.n}${unitLabels[p.unit]}` : text
}

/** 입력칸의 숫자 글을 검사한다. 1~9999 정수만 받는다. 맞으면 null이다. */
export function chartPeriodNumberError(nText: string): string | null {
  if (!/^\d+$/.test(nText.trim())) {
    return '1~9999 사이 정수를 넣어 주세요'
  }
  const n = Number(nText.trim())
  return n >= 1 && n <= 9999 ? null : '1~9999 사이 정수를 넣어 주세요'
}
