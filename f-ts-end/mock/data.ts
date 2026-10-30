// 화면 확인용 가짜 데이터. specification/40.api/README.md의 응답 예시 모양을 따른다.
// 값은 모두 지어낸 것이다. 실제 수집값이 아니다.
import type { Card, Cycle, Detail, Point } from '../src/api.ts'

interface Fake {
  id: string
  name: string
  cycle: Cycle
  unit: string | null
  notice?: string
  source: { code: string; name: string }
  stat_code: string
  /** 마지막 기간 시작일과 시계열 값을 만드는 함수. 없으면 관측값이 없다. */
  series?: { last: string; count: number; value: (i: number) => number }
  chart_period: string | null
  last_collected_at: string | null
  related: string[]
}

const ecos = { code: 'ecos', name: 'ECOS' }
const kosis = { code: 'kosis', name: 'KOSIS' }
const dataGoKr = { code: 'data-go-kr', name: '공공데이터포털' }
const notice = '이 정보는 신뢰할 수 없습니다. 투자 판단에 사용하지 마십시오.'
const collected = '2026-10-25T06:00:03+09:00'

// 지어낸 값이 매끄럽게 오르내리게 한다.
const wave = (base: number, amp: number, step = 0.3) => (i: number) =>
  Math.round((base + amp * Math.sin(i * step)) * 100) / 100

const fakes: Fake[] = [
  { id: 'ecos-base-rate', name: '한국은행 기준금리', cycle: 'M', unit: '연%', source: ecos, stat_code: '722Y001',
    series: { last: '2026-09-01', count: 120, value: (i) => [1.25, 1.5, 0.5, 1.75, 3.5, 2.5][Math.floor(i / 20)] },
    chart_period: '10 years', last_collected_at: collected, related: ['kosis-employment-rate', 'data-go-kr-kodex-200'] },
  { id: 'kosis-employment-rate', name: '고용률(15세 이상)', cycle: 'M', unit: '%', source: kosis, stat_code: 'DT_1DA7001S',
    series: { last: '2026-09-01', count: 120, value: wave(61.5, 1.2) }, chart_period: '10 years',
    last_collected_at: collected, related: ['ecos-base-rate'] },
  { id: 'kosis-total-population', name: '총인구', cycle: 'A', unit: '명', source: kosis, stat_code: 'DT_1IN1502',
    series: { last: '2025-01-01', count: 20, value: (i) => 49_000_000 + i * 150_000 }, chart_period: '20 years',
    last_collected_at: collected, related: [] },
  { id: 'kosis-births', name: '출생아 수', cycle: 'A', unit: null, source: kosis, stat_code: 'INH_1B8000F_01',
    series: { last: '2025-01-01', count: 20, value: (i) => 450_000 - i * 10_000 }, chart_period: '20 years',
    last_collected_at: collected, related: ['kosis-total-population'] },
  { id: 'fake-quarterly', name: '분기 지표(가짜)', cycle: 'Q', unit: '조원', source: ecos, stat_code: 'FAKE01',
    series: { last: '2026-04-01', count: 40, value: wave(500, 40, 0.5) }, chart_period: '10 years',
    last_collected_at: collected, related: [] },
  { id: 'data-go-kr-kodex-200', name: 'KODEX 200', cycle: 'D', unit: null, source: dataGoKr, stat_code: '069500',
    series: { last: '2026-10-29', count: 250, value: wave(90000, 6000, 0.05) }, chart_period: '1 year',
    last_collected_at: collected, related: ['data-go-kr-kodex-leverage', 'data-go-kr-kodex-inverse'] },
  { id: 'data-go-kr-kodex-leverage', name: 'KODEX 레버리지', cycle: 'D', unit: null, notice, source: dataGoKr,
    stat_code: '122630', series: { last: '2026-10-29', count: 250, value: wave(75000, 9000, 0.05) },
    chart_period: '1 year', last_collected_at: collected, related: ['data-go-kr-kodex-200'] },
  { id: 'data-go-kr-kodex-inverse', name: 'KODEX 인버스', cycle: 'D', unit: null, notice, source: dataGoKr,
    stat_code: '114800', series: { last: '2026-10-29', count: 250, value: wave(1200, 80, 0.05) },
    chart_period: '1 year', last_collected_at: collected, related: ['data-go-kr-kodex-200'] },
  // 아직 수집하지 않은 지표: 값, 단위, 차트, 마지막 수집이 없다.
  { id: 'fake-empty', name: '아직 수집 전(가짜)', cycle: 'M', unit: null, source: ecos, stat_code: 'FAKE02',
    chart_period: null, last_collected_at: null, related: [] },
]

/** last에서 주기만큼 거슬러 간 기간 시작일(YYYY-MM-DD). */
function back(last: string, cycle: Cycle, n: number): string {
  const d = new Date(`${last}T00:00:00Z`)
  if (cycle === 'D') d.setUTCDate(d.getUTCDate() - n)
  if (cycle === 'M') d.setUTCMonth(d.getUTCMonth() - n)
  if (cycle === 'Q') d.setUTCMonth(d.getUTCMonth() - 3 * n)
  if (cycle === 'A') d.setUTCFullYear(d.getUTCFullYear() - n)
  return d.toISOString().slice(0, 10)
}

/** 오래된 것부터의 시계열. */
function series(f: Fake): Point[] {
  if (!f.series) return []
  const { last, count, value } = f.series
  return Array.from({ length: count }, (_, i) => ({
    period: back(last, f.cycle, count - 1 - i),
    value: value(i),
  }))
}

export function cards(): { indicators: Card[] } {
  return {
    indicators: fakes.map((f) => {
      const s = series(f)
      return { id: f.id, name: f.name, cycle: f.cycle, unit: f.unit, latest: s.at(-1) ?? null }
    }),
  }
}

export function detail(id: string): Detail | undefined {
  const f = fakes.find((x) => x.id === id)
  if (!f) return undefined
  const s = series(f)
  return {
    id: f.id,
    name: f.name,
    cycle: f.cycle,
    unit: f.unit,
    notice: f.notice ?? null,
    source: f.source,
    stat_code: f.stat_code,
    last_collected_at: f.last_collected_at,
    latest: s.at(-1) ?? null,
    chart_period: f.chart_period,
    chart: s,
    recent: s.slice(-10).reverse(),
    related: f.related.map((r) => ({ id: r, name: fakes.find((x) => x.id === r)?.name ?? r })),
  }
}
