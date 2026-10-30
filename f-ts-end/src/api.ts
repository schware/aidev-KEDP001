// Go 조회 API의 응답 형식이다. 기준: specification/40.api/README.md

export type Cycle = 'D' | 'M' | 'Q' | 'A'

/** 관측값 하나. period는 기간 시작일(YYYY-MM-DD)이다(40.api 1.1). */
export interface Point {
  period: string
  value: number
}

/** GET /api/indicators의 카드 하나(40.api 2장). */
export interface Card {
  id: string
  name: string
  cycle: Cycle
  unit: string | null
  /** 소수 자릿수 0~4. null이면 받은 값 그대로 보여준다(40.api 1.1). */
  decimals: number | null
  latest: Point | null
}

export interface CardList {
  indicators: Card[]
}

/** GET /api/indicators/{id}(40.api 3장). */
export interface Detail {
  id: string
  name: string
  cycle: Cycle
  unit: string | null
  decimals: number | null
  notice: string | null
  source: { code: string; name: string }
  stat_code: string
  last_collected_at: string | null
  latest: Point | null
  chart_period: string | null
  chart: Point[]
  recent: Point[]
  related: { id: string; name: string }[]
}

/** 오류 응답 본문(40.api 1.4, 5.2). 400 입력 오류는 fields에 칸 이름별 이유가 붙는다. */
export interface ErrorBody {
  error?: string
  fields?: Record<string, string>
}

/** API가 200이 아닌 상태를 돌려줬다. */
export class ApiError extends Error {
  readonly status: number
  readonly fields: Record<string, string>

  constructor(status: number, body?: ErrorBody) {
    super(body?.error ?? `API status ${status}`)
    this.status = status
    this.fields = body?.fields ?? {}
  }
}

export async function getJSON<T>(path: string, signal?: AbortSignal): Promise<T> {
  const res = await fetch(path, { signal, headers: { Accept: 'application/json' } })
  if (!res.ok) {
    throw new ApiError(res.status)
  }
  return (await res.json()) as T
}

export const cardsPath = '/api/indicators'

export function detailPath(id: string): string {
  return `/api/indicators/${encodeURIComponent(id)}`
}
