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

/** API가 200이 아닌 상태를 돌려줬다. */
export class ApiError extends Error {
  readonly status: number

  constructor(status: number) {
    super(`API status ${status}`)
    this.status = status
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
