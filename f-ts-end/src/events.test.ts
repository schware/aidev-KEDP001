import { describe, expect, it, vi } from 'vitest'
import { adminAction, apiError, apiNetworkError, cleanPath, eventsPath, pageView, sendEvent, type EventDeps } from './events.ts'

function beaconDeps(ok = true) {
  const sent: { url: string; data: unknown }[] = []
  const deps: EventDeps = {
    sendBeacon: (url, data) => {
      sent.push({ url, data: JSON.parse(data) })
      return ok
    },
    fetch: vi.fn(() => Promise.resolve(new Response(null, { status: 204 }))),
  }
  return { sent, deps }
}

describe('sendEvent', () => {
  it('sendBeacon으로 동작 하나를 JSON으로 보낸다', () => {
    const { sent, deps } = beaconDeps()
    sendEvent({ event: 'ui.card_open', indicator: 'cpi' }, deps)
    expect(sent).toEqual([{ url: eventsPath, data: { event: 'ui.card_open', indicator: 'cpi' } }])
    expect(deps.fetch).not.toHaveBeenCalled()
  })

  it('sendBeacon이 없거나 false면 fetch + keepalive로 보낸다', () => {
    for (const deps of [beaconDeps(false).deps, { fetch: beaconDeps().deps.fetch }]) {
      sendEvent({ event: 'ui.page_view', page: '/' }, deps)
      expect(deps.fetch).toHaveBeenCalledWith(eventsPath, {
        method: 'POST',
        body: '{"event":"ui.page_view","page":"/"}',
        keepalive: true,
      })
    }
  })

  it('보내기가 실패해도 던지지 않는다', async () => {
    const rejected = vi.fn(() => Promise.reject(new TypeError('network')))
    expect(() => sendEvent({ event: 'ui.page_view', page: '/' }, { fetch: rejected })).not.toThrow()
    const throwing: EventDeps = {
      sendBeacon: () => {
        throw new Error('blocked')
      },
    }
    expect(() => sendEvent({ event: 'ui.page_view', page: '/' }, throwing)).not.toThrow()
    expect(() => sendEvent({ event: 'ui.page_view', page: '/' }, {})).not.toThrow()
    await Promise.resolve() // reject된 Promise가 처리되지 않은 채 남지 않는다
  })
})

describe('도우미', () => {
  it('cleanPath는 query string과 hash를 뺀다', () => {
    expect(cleanPath('/indicator/cpi?x=1#top')).toBe('/indicator/cpi')
    expect(cleanPath('/api/indicators')).toBe('/api/indicators')
  })

  it('pageView, adminAction', () => {
    const { sent, deps } = beaconDeps()
    pageView('/admin/dashboard?tab=1', deps)
    adminAction('display_order_save', deps)
    expect(sent.map((s) => s.data)).toEqual([
      { event: 'ui.page_view', page: '/admin/dashboard' },
      { event: 'ui.admin_action', action: 'display_order_save' },
    ])
  })

  it('apiError는 path와 status를 보내고, 로그인하지 않은 상태 확인(401)은 보내지 않는다', () => {
    const { sent, deps } = beaconDeps()
    apiError('/api/indicators/cpi?x=1', 500, deps)
    apiError('/api/admin/session', 401, deps)
    apiError('/api/admin/session', 500, deps)
    apiError('/api/admin/indicators', 401, deps)
    expect(sent.map((s) => s.data)).toEqual([
      { event: 'ui.error', path: '/api/indicators/cpi', status: 500 },
      { event: 'ui.error', path: '/api/admin/session', status: 500 },
      { event: 'ui.error', path: '/api/admin/indicators', status: 401 },
    ])
  })

  it('apiNetworkError는 status 0을 보내고, 취소(AbortError)는 보내지 않는다', () => {
    const { sent, deps } = beaconDeps()
    apiNetworkError('/api/indicators', new DOMException('aborted', 'AbortError'), deps)
    apiNetworkError('/api/indicators', new TypeError('Failed to fetch'), deps)
    expect(sent.map((s) => s.data)).toEqual([{ event: 'ui.error', path: '/api/indicators', status: 0 }])
  })

  it('보내는 본문에 level, 비밀번호 같은 칸이 없다', () => {
    const { sent, deps } = beaconDeps()
    adminAction('login', deps)
    expect(Object.keys(sent[0].data as object).sort()).toEqual(['action', 'event'])
  })
})
