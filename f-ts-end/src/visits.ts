// 사용자 화면의 방문자 수(specification/40.api/README.md 6.2, 6.3). React와 상관없이 시험할 수 있게 따로 둔다.
//
// 사용자 화면이 처음 열릴 때 POST /api/visits를 한 번 보내고, 그다음 GET /api/visits/summary를 읽는다.
// localStorage의 kedp-visit-date에 Asia/Seoul 오늘 날짜(YYYY-MM-DD)가 있으면 POST를 보내지 않는다.
// localStorage에는 이 날짜만 둔다. localStorage를 쓸 수 없으면 그냥 보낸다(중복은 Go가 거른다).

/** GET /api/visits/summary의 응답(40.api 6.3). */
export interface VisitSummary {
  today: number
  total: number
}

export const visitsPath = '/api/visits'
export const visitSummaryPath = '/api/visits/summary'
export const visitDateKey = 'kedp-visit-date'

const seoulDateFormat = new Intl.DateTimeFormat('en-CA', {
  timeZone: 'Asia/Seoul',
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
})

/** now의 Asia/Seoul 날짜(YYYY-MM-DD). */
export function seoulDate(now: Date): string {
  const p = Object.fromEntries(seoulDateFormat.formatToParts(now).map((x) => [x.type, x.value]))
  return `${p.year}-${p.month}-${p.day}`
}

/** 날짜만 읽고 쓰면 된다. */
export type DateStorage = Pick<Storage, 'getItem' | 'setItem'>

/** 브라우저 localStorage. 쓸 수 없으면(사생활 보호 모드, 막힘) null이다. 꺼내기만 해도 오류가 날 수 있다. */
export function browserStorage(): DateStorage | null {
  try {
    return globalThis.localStorage ?? null
  } catch {
    return null
  }
}

/** 오늘 이미 보냈으면 false다. 저장소를 읽을 수 없으면 true(보낸다). */
export function shouldSendVisit(storage: DateStorage | null, today: string): boolean {
  if (!storage) return true
  try {
    return storage.getItem(visitDateKey) !== today
  } catch {
    return true
  }
}

/** 보낸 날짜를 적는다. 적을 수 없으면 그냥 넘어간다. */
export function markVisitSent(storage: DateStorage | null, today: string): void {
  try {
    storage?.setItem(visitDateKey, today)
  } catch {
    // 사생활 보호 모드, 저장소가 가득 참: 다음에 다시 보내도 Go가 거른다.
  }
}

export interface VisitDeps {
  fetch: typeof fetch
  storage: DateStorage | null
  now: () => Date
}

function isSummary(v: unknown): v is VisitSummary {
  const o = v as Partial<VisitSummary> | null
  return typeof o === 'object' && o !== null && Number.isFinite(o.today) && Number.isFinite(o.total)
}

/**
 * 방문을 (오늘 처음이면) 보내고 오늘·전체 방문자 수를 읽는다. 어느 하나라도 실패하면 null이다(footer 숫자를 숨긴다).
 * POST가 실패하면 날짜를 적지 않아, 다음에 화면을 열 때 다시 보낸다.
 */
export async function loadVisits({ fetch, storage, now }: VisitDeps): Promise<VisitSummary | null> {
  try {
    const today = seoulDate(now())
    if (shouldSendVisit(storage, today)) {
      const res = await fetch(visitsPath, { method: 'POST' })
      if (!res.ok) return null
      markVisitSent(storage, today)
    }
    const res = await fetch(visitSummaryPath, { headers: { Accept: 'application/json' } })
    if (!res.ok) return null
    const body: unknown = await res.json()
    return isSummary(body) ? body : null
  } catch {
    return null
  }
}

/** 한 번만 부르는 loadVisits. 화면 안에서 메인과 상세를 오가도(다시 mount되어도) 같은 결과를 쓴다. */
export function onceVisits(deps: () => VisitDeps): () => Promise<VisitSummary | null> {
  let p: Promise<VisitSummary | null> | null = null
  return () => (p ??= loadVisits(deps()))
}

/** 사용자 화면이 쓰는 것. 페이지를 새로 열 때(새로고침 포함)만 다시 부른다. */
export const visitsOnce = onceVisits(() => ({
  fetch: (...a) => globalThis.fetch(...a),
  storage: browserStorage(),
  now: () => new Date(),
}))
