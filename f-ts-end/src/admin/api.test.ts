import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '../api.ts'
import { adminRequest, authFailure, checkSession } from './api.ts'

/** fetch를 한 번 응답하는 가짜로 바꾼다. */
function mockFetch(status: number, body?: unknown) {
  const fn = vi.fn(async () =>
    new Response(body === undefined ? null : JSON.stringify(body), {
      status,
      headers: { 'Content-Type': 'application/json; charset=utf-8' },
    }),
  )
  vi.stubGlobal('fetch', fn)
  return fn
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('checkSession: 관리자 화면에 들어올 때', () => {
  it('200이면 로그인 상태', async () => {
    const fn = mockFetch(200, { account: 'admin', expires_at: '2026-10-30T18:00:00+09:00' })
    expect(await checkSession()).toEqual({
      kind: 'ok',
      session: { account: 'admin', expires_at: '2026-10-30T18:00:00+09:00' },
    })
    expect(fn).toHaveBeenCalledWith('/api/admin/session', expect.objectContaining({ method: 'GET' }))
  })

  it('401이면 로그인 화면으로', async () => {
    mockFetch(401, { error: 'login required' })
    expect(await checkSession()).toEqual({ kind: 'login' })
  })

  it('403이면 허용된 주소가 아니다', async () => {
    mockFetch(403, { error: 'forbidden' })
    expect(await checkSession()).toEqual({ kind: 'forbidden' })
  })

  it('그 밖의 오류', async () => {
    mockFetch(500, { error: 'internal error' })
    expect(await checkSession()).toEqual({ kind: 'error' })
    vi.stubGlobal('fetch', vi.fn(async () => Promise.reject(new TypeError('network'))))
    expect(await checkSession()).toEqual({ kind: 'error' })
  })
})

describe('authFailure: 화면 안에서 받은 오류', () => {
  it('401은 로그인, 403은 허용 안 됨, 나머지는 null', () => {
    expect(authFailure(new ApiError(401))).toBe('login')
    expect(authFailure(new ApiError(403))).toBe('forbidden')
    expect(authFailure(new ApiError(400))).toBeNull()
    expect(authFailure(new ApiError(500))).toBeNull()
    expect(authFailure(new Error('x'))).toBeNull()
  })
})

describe('adminRequest', () => {
  it('PUT은 JSON으로 보내고 응답을 돌려준다', async () => {
    const fn = mockFetch(200, { collect_time: '07:30' })
    expect(await adminRequest('PUT', '/api/admin/collect-time', { collect_time: '07:30' })).toEqual({
      collect_time: '07:30',
    })
    const init = (fn.mock.calls[0] as unknown as [string, RequestInit])[1]
    expect(init.method).toBe('PUT')
    expect(init.body).toBe('{"collect_time":"07:30"}')
    expect((init.headers as Record<string, string>)['Content-Type']).toBe('application/json')
    expect(init.credentials).toBe('same-origin')
  })

  it('400이면 칸 오류(fields)가 붙은 ApiError', async () => {
    mockFetch(400, { error: 'invalid input', fields: { cycle: 'must be one of D, M, Q, A' } })
    const err = await adminRequest('PUT', '/api/admin/indicators/x', {}).catch((e: unknown) => e)
    expect(err).toBeInstanceOf(ApiError)
    expect((err as ApiError).status).toBe(400)
    expect((err as ApiError).fields).toEqual({ cycle: 'must be one of D, M, Q, A' })
  })

  it('본문이 JSON이 아니어도 상태는 남는다', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => new Response('oops', { status: 502 })))
    const err = await adminRequest('GET', '/api/admin/sources').catch((e: unknown) => e)
    expect((err as ApiError).status).toBe(502)
    expect((err as ApiError).fields).toEqual({})
  })

  it('204(로그아웃)는 본문 없이 끝난다', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => new Response(null, { status: 204 })))
    expect(await adminRequest('POST', '/api/admin/logout')).toBeUndefined()
  })
})
