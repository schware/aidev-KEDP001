import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  browserStorage,
  loadVisits,
  onceVisits,
  seoulDate,
  shouldSendVisit,
  visitDateKey,
  type DateStorage,
  type VisitDeps,
} from './visits.ts'

/** 날짜만 담는 가짜 localStorage. */
function memoryStorage(init: Record<string, string> = {}): DateStorage & { data: Record<string, string> } {
  const data = { ...init }
  return {
    data,
    getItem: (k) => data[k] ?? null,
    setItem: (k, v) => {
      data[k] = v
    },
  }
}

/** 읽기·쓰기가 모두 오류인 저장소(사생활 보호 모드 등). */
const brokenStorage: DateStorage = {
  getItem: () => {
    throw new DOMException('blocked', 'SecurityError')
  },
  setItem: () => {
    throw new DOMException('full', 'QuotaExceededError')
  },
}

/** 주소별 응답을 정하는 가짜 fetch. 부른 순서를 남긴다. */
function fakeFetch(post = 204, summary: number | 'network' = 200, body: unknown = { today: 1234, total: 56789 }) {
  return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input)
    if (url === '/api/visits' && init?.method === 'POST') {
      return new Response(null, { status: post })
    }
    if (url === '/api/visits/summary') {
      if (summary === 'network') throw new TypeError('Failed to fetch')
      return new Response(JSON.stringify(summary === 200 ? body : { error: 'internal error' }), { status: summary })
    }
    return new Response(null, { status: 404 })
  })
}

const calls = (f: ReturnType<typeof fakeFetch>) =>
  f.mock.calls.map(([u, init]) => `${(init as RequestInit | undefined)?.method ?? 'GET'} ${String(u)}`)

// 2026-11-01 23:59:59 Asia/Seoul = 2026-11-01T14:59:59Z, 다음 1초가 11-02 00:00:00
const beforeMidnight = new Date('2026-11-01T14:59:59Z')
const afterMidnight = new Date('2026-11-01T15:00:00Z')

function deps(over: Partial<VisitDeps>): VisitDeps {
  return { fetch: fakeFetch(), storage: memoryStorage(), now: () => afterMidnight, ...over }
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('seoulDate: Asia/Seoul 날짜', () => {
  it('자정 경계', () => {
    expect(seoulDate(beforeMidnight)).toBe('2026-11-01')
    expect(seoulDate(afterMidnight)).toBe('2026-11-02')
  })
  it('UTC로는 전날이어도 서울이 다음 날이면 다음 날', () => {
    expect(seoulDate(new Date('2026-12-31T15:00:00Z'))).toBe('2027-01-01')
    expect(seoulDate(new Date('2026-12-31T14:59:59.999Z'))).toBe('2026-12-31')
  })
})

describe('shouldSendVisit: localStorage 날짜 판단', () => {
  it('오늘 날짜가 있으면 보내지 않는다', () => {
    expect(shouldSendVisit(memoryStorage({ [visitDateKey]: '2026-11-02' }), '2026-11-02')).toBe(false)
  })
  it('없거나 다른 날이면 보낸다', () => {
    expect(shouldSendVisit(memoryStorage(), '2026-11-02')).toBe(true)
    expect(shouldSendVisit(memoryStorage({ [visitDateKey]: '2026-11-01' }), '2026-11-02')).toBe(true)
  })
  it('저장소를 쓸 수 없으면 보낸다', () => {
    expect(shouldSendVisit(null, '2026-11-02')).toBe(true)
    expect(shouldSendVisit(brokenStorage, '2026-11-02')).toBe(true)
  })
})

describe('loadVisits', () => {
  it('처음: POST 다음 summary를 읽고, 날짜만 적는다', async () => {
    const f = fakeFetch()
    const storage = memoryStorage()
    expect(await loadVisits(deps({ fetch: f, storage }))).toEqual({ today: 1234, total: 56789 })
    expect(calls(f)).toEqual(['POST /api/visits', 'GET /api/visits/summary'])
    expect(storage.data).toEqual({ [visitDateKey]: '2026-11-02' })
  })

  it('같은 날 다시 열면 POST 없이 summary만', async () => {
    const f = fakeFetch()
    const storage = memoryStorage({ [visitDateKey]: '2026-11-02' })
    await loadVisits(deps({ fetch: f, storage }))
    expect(calls(f)).toEqual(['GET /api/visits/summary'])
  })

  it('23:59:59에 보냈어도 00:00(Asia/Seoul)이 지나면 다시 보낸다', async () => {
    const storage = memoryStorage()
    const f1 = fakeFetch()
    await loadVisits(deps({ fetch: f1, storage, now: () => beforeMidnight }))
    expect(storage.data[visitDateKey]).toBe('2026-11-01')
    const f2 = fakeFetch()
    await loadVisits(deps({ fetch: f2, storage, now: () => afterMidnight }))
    expect(calls(f2)).toEqual(['POST /api/visits', 'GET /api/visits/summary'])
    expect(storage.data[visitDateKey]).toBe('2026-11-02')
  })

  it('localStorage를 쓸 수 없으면 그냥 보내고 숫자를 보인다', async () => {
    for (const storage of [null, brokenStorage]) {
      const f = fakeFetch()
      expect(await loadVisits(deps({ fetch: f, storage }))).toEqual({ today: 1234, total: 56789 })
      expect(calls(f)).toEqual(['POST /api/visits', 'GET /api/visits/summary'])
    }
  })

  it('localStorage를 꺼내기만 해도 오류인 브라우저', () => {
    vi.stubGlobal('localStorage', undefined)
    Object.defineProperty(globalThis, 'localStorage', {
      configurable: true,
      get() {
        throw new DOMException('blocked', 'SecurityError')
      },
    })
    expect(browserStorage()).toBeNull()
    Reflect.deleteProperty(globalThis, 'localStorage')
  })

  it('POST가 실패하면 숫자를 숨기고(null) 날짜를 적지 않는다', async () => {
    const storage = memoryStorage()
    const f = fakeFetch(500)
    expect(await loadVisits(deps({ fetch: f, storage }))).toBeNull()
    expect(calls(f)).toEqual(['POST /api/visits'])
    expect(storage.data).toEqual({})
  })

  it('summary가 실패하면 숨긴다: 500, 연결 실패, 형식이 틀린 응답', async () => {
    expect(await loadVisits(deps({ fetch: fakeFetch(204, 500) }))).toBeNull()
    expect(await loadVisits(deps({ fetch: fakeFetch(204, 'network') }))).toBeNull()
    expect(await loadVisits(deps({ fetch: fakeFetch(204, 200, { today: '1' }) }))).toBeNull()
    expect(await loadVisits(deps({ fetch: fakeFetch(204, 200, null) }))).toBeNull()
  })

  it('POST 연결 실패도 오류를 던지지 않고 숨긴다', async () => {
    const f = vi.fn(async () => {
      throw new TypeError('Failed to fetch')
    })
    expect(await loadVisits(deps({ fetch: f }))).toBeNull()
  })
})

describe('onceVisits: 화면 안에서 오가도 한 번만', () => {
  it('여러 번 불러도 POST·summary는 한 번', async () => {
    const f = fakeFetch()
    const load = onceVisits(() => deps({ fetch: f }))
    const [a, b] = await Promise.all([load(), load()])
    await load()
    expect(a).toEqual(b)
    expect(calls(f)).toEqual(['POST /api/visits', 'GET /api/visits/summary'])
  })
})
