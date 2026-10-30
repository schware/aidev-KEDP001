// 화면 확인용 가짜 데이터. specification/40.api/README.md의 응답 예시 모양을 따른다.
// 값과 코드는 모두 지어낸 것이다. 실제 수집값이나 확인한 지표 코드가 아니다.
// 관리자 API(mock/admin.ts)가 이 목록을 바꾸면 사용자 화면에도 보인다(dev 서버를 다시 켜면 처음으로 돌아간다).
import type { Card, Cycle, Detail, Point } from '../src/api.ts'
import type { AdminIndicator, SourceCode } from '../src/admin/api.ts'

interface Series {
  last: string
  count: number
  value: (i: number) => number
}

export interface Fake extends AdminIndicator {
  /** 마지막 기간 시작일과 시계열 값을 만드는 함수. 없으면 관측값이 없다. */
  series?: Series
  /** series를 만든 주기. 주기를 바꾸면 이전 주기 값은 보이지 않는다(40.api 1.3). */
  seriesCycle?: Cycle
  last_collected_at: string | null
}

const notice = '이 정보는 신뢰할 수 없습니다. 투자 판단에 사용하지 마십시오.'
const collected = '2026-10-25T06:00:03+09:00'

// 지어낸 값이 매끄럽게 오르내리게 한다.
const wave = (base: number, amp: number, step = 0.3) => (i: number) =>
  Math.round((base + amp * Math.sin(i * step)) * 100) / 100

type Init = Pick<Fake, 'id' | 'name' | 'cycle' | 'stat_code'> &
  Partial<Omit<Fake, 'id' | 'name' | 'cycle' | 'stat_code' | 'source'>> & { source: SourceCode }

function fake(x: Init): Fake {
  return {
    item_code: x.source === 'data-go-kr' ? null : 'fake-item',
    classes: [],
    org_id: x.source === 'kosis' ? 'fake-org' : null,
    value_field: x.source === 'data-go-kr' ? 'clpr' : null,
    unit: null,
    source_unit: null,
    visible: true,
    order: null,
    chart_period: null,
    decimals: null,
    notice: null,
    related: [],
    last_collected_at: collected,
    seriesCycle: x.cycle,
    ...x,
  }
}

const initial: Fake[] = [
  // 40.api 5.4 응답 예시
  fake({ id: 'ecos-base-rate', name: '한국은행 기준금리', source: 'ecos', stat_code: '722Y001', item_code: '0101000',
    cycle: 'M', source_unit: '연%', order: 1, chart_period: '10 years', decimals: 2,
    series: { last: '2026-09-01', count: 120, value: (i) => [1.25, 1.5, 0.5, 1.75, 3.5, 2.5][Math.floor(i / 20)] },
    related: ['kosis-employment-rate', 'data-go-kr-kodex-200'] }),
  fake({ id: 'kosis-employment-rate', name: '고용률(15세 이상)', source: 'kosis', stat_code: 'DT_1DA7001S', classes: ['fake-class'],
    cycle: 'M', unit: '%', order: 2, chart_period: '10 years', decimals: 1,
    series: { last: '2026-09-01', count: 120, value: wave(61.5, 1.2) }, related: ['ecos-base-rate'] }),
  fake({ id: 'kosis-total-population', name: '총인구', source: 'kosis', stat_code: 'DT_1IN1502', cycle: 'A', source_unit: '명',
    order: 3, chart_period: '20 years', decimals: 0,
    series: { last: '2025-01-01', count: 20, value: (i) => 49_000_000 + i * 150_000 } }),
  fake({ id: 'kosis-births', name: '출생아 수', source: 'kosis', stat_code: 'INH_1B8000F_01', cycle: 'A', order: 4,
    chart_period: '20 years', series: { last: '2025-01-01', count: 20, value: (i) => 450_000 - i * 10_000 },
    related: ['kosis-total-population'] }),
  fake({ id: 'fake-quarterly', name: '분기 지표(가짜)', source: 'ecos', stat_code: 'FAKE01', cycle: 'Q', unit: '조원', order: 5,
    chart_period: '10 years', series: { last: '2026-04-01', count: 40, value: wave(500, 40, 0.5) } }),
  fake({ id: 'data-go-kr-kodex-200', name: 'KODEX 200', source: 'data-go-kr', stat_code: '069500', cycle: 'D', order: 6,
    chart_period: '1 years', series: { last: '2026-10-29', count: 250, value: wave(90000, 6000, 0.05) },
    related: ['data-go-kr-kodex-leverage', 'data-go-kr-kodex-inverse'] }),
  fake({ id: 'data-go-kr-kodex-leverage', name: 'KODEX 레버리지', source: 'data-go-kr', stat_code: '122630', cycle: 'D',
    notice, order: 7, chart_period: '1 years', series: { last: '2026-10-29', count: 250, value: wave(75000, 9000, 0.05) },
    related: ['data-go-kr-kodex-200'] }),
  fake({ id: 'data-go-kr-kodex-inverse', name: 'KODEX 인버스', source: 'data-go-kr', stat_code: '114800', cycle: 'D',
    notice, order: 8, chart_period: '1 years', series: { last: '2026-10-29', count: 250, value: wave(1200, 80, 0.05) },
    related: ['data-go-kr-kodex-200'] }),
  // 아직 수집하지 않은 지표: 값, 단위, 마지막 수집이 없다.
  fake({ id: 'fake-empty', name: '아직 수집 전(가짜)', source: 'ecos', stat_code: 'FAKE02', cycle: 'M', order: 9,
    chart_period: '10 years', last_collected_at: null }),
  // 숨긴 지표, 화면 설정이 없는 지표(관리자 화면에서만 보인다).
  fake({ id: 'fake-hidden', name: '숨긴 지표(가짜)', source: 'ecos', stat_code: 'FAKE03', cycle: 'M', visible: false, order: 10,
    chart_period: '5 years', series: { last: '2026-09-01', count: 60, value: wave(100, 5) } }),
  fake({ id: 'fake-no-display', name: '화면 설정 없음(가짜)', source: 'ecos', stat_code: 'FAKE04', cycle: 'M', visible: false,
    last_collected_at: null }),
]

/** 지금 지표 목록. 관리자 API가 바꾼다. */
export const fakes: Fake[] = initial

const unitOf = (f: Fake) => f.unit || f.source_unit || null
const sourceNames: Record<SourceCode, string> = { ecos: 'ECOS', kosis: 'KOSIS', 'data-go-kr': '공공데이터포털' }

/** last에서 주기만큼 거슬러 간 기간 시작일(YYYY-MM-DD). */
function back(last: string, cycle: Cycle, n: number): string {
  const d = new Date(`${last}T00:00:00Z`)
  if (cycle === 'D') d.setUTCDate(d.getUTCDate() - n)
  if (cycle === 'M') d.setUTCMonth(d.getUTCMonth() - n)
  if (cycle === 'Q') d.setUTCMonth(d.getUTCMonth() - 3 * n)
  if (cycle === 'A') d.setUTCFullYear(d.getUTCFullYear() - n)
  return d.toISOString().slice(0, 10)
}

/** 오래된 것부터의 시계열. 지금 주기 것만이다. */
function series(f: Fake): Point[] {
  if (!f.series || f.seriesCycle !== f.cycle) return []
  const { last, count, value } = f.series
  return Array.from({ length: count }, (_, i) => ({
    period: back(last, f.cycle, count - 1 - i),
    value: value(i),
  }))
}

/** 관리자 목록 순서(40.api 5.4): order, id. 화면 설정이 없는 지표는 맨 뒤. */
export function sorted(): Fake[] {
  return [...fakes].sort(
    (a, b) => (a.order ?? Infinity) - (b.order ?? Infinity) || a.id.localeCompare(b.id),
  )
}

export function cards(): { indicators: Card[] } {
  return {
    indicators: sorted()
      .filter((f) => f.visible && f.order !== null)
      .map((f) => ({
        id: f.id,
        name: f.name,
        cycle: f.cycle,
        unit: unitOf(f),
        decimals: f.decimals,
        latest: series(f).at(-1) ?? null,
      })),
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
    unit: unitOf(f),
    decimals: f.decimals,
    notice: f.notice,
    source: { code: f.source, name: sourceNames[f.source] },
    stat_code: f.stat_code,
    last_collected_at: s.length > 0 ? f.last_collected_at : null,
    latest: s.at(-1) ?? null,
    chart_period: f.order === null ? null : f.chart_period,
    chart: f.order === null ? [] : s,
    recent: s.slice(-10).reverse(),
    related: f.related.map((r) => ({ id: r, name: fakes.find((x) => x.id === r)?.name ?? r })),
  }
}
