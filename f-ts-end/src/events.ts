// 화면 동작을 서버 로그로 보낸다(specification/40.api/README.md 7장, KEDP001-D026 6장, Issue #94).
// React와 상관없이 시험할 수 있게 따로 둔다.
//
// - POST /api/events로 보낸다. navigator.sendBeacon을 먼저 쓰고, 안 되면 fetch + keepalive다.
// - 실패해도 다시 보내지 않고, 오류를 밖으로 던지지 않는다. 화면 동작을 막지 않는다.
// - level은 보내지 않는다(Go가 정한다). 비밀번호, 입력값, 계정 이름은 보내지 않는다.

export const eventsPath = '/api/events'

/** ui.admin_action의 action(40.api 7.1). 관리자 화면의 버튼이다. */
export type AdminAction =
  | 'login'
  | 'logout'
  | 'indicator_add'
  | 'indicator_edit'
  | 'display_order_save'
  | 'collect_time_save'
  | 'visit_base_save'
  | 'visit_reset'

/** 보내는 동작(40.api 7.1의 허용 목록). */
export type UiEvent =
  | { event: 'ui.page_view'; page: string }
  | { event: 'ui.card_open'; indicator: string }
  | { event: 'ui.related_open'; indicator: string; from: string }
  | { event: 'ui.admin_action'; action: AdminAction }
  | { event: 'ui.error'; path: string; status: number }

export interface EventDeps {
  /** navigator.sendBeacon. 없거나 false를 돌려주면 fetch를 쓴다. */
  sendBeacon?: (url: string, data: string) => boolean
  fetch?: typeof fetch
}

/** 브라우저의 sendBeacon과 fetch. 꺼내다 오류가 나면 없는 것으로 본다. */
export function browserEventDeps(): EventDeps {
  const deps: EventDeps = {}
  try {
    // Node 타입(@types/node)의 navigator에는 sendBeacon이 없어 직접 적는다.
    const nav = globalThis.navigator as { sendBeacon?: (url: string, data: string) => boolean } | undefined
    if (nav && typeof nav.sendBeacon === 'function') {
      deps.sendBeacon = nav.sendBeacon.bind(nav)
    }
  } catch {
    // sendBeacon을 쓸 수 없다: fetch만 쓴다.
  }
  if (typeof globalThis.fetch === 'function') {
    deps.fetch = (...a) => globalThis.fetch(...a)
  }
  return deps
}

/** 경로에서 query string과 hash를 뺀다. 서버는 query string이 있는 경로를 받지 않는다(40.api 7.1). */
export function cleanPath(path: string): string {
  return path.split(/[?#]/, 1)[0]
}

/** 동작 하나를 보낸다. 무슨 일이 있어도 던지지 않는다. */
export function sendEvent(e: UiEvent, deps: EventDeps = browserEventDeps()): void {
  try {
    const body = JSON.stringify(e)
    if (deps.sendBeacon?.(eventsPath, body)) return
    void deps.fetch?.(eventsPath, { method: 'POST', body, keepalive: true }).catch(() => undefined)
  } catch {
    // 보내지 못해도 화면은 그대로 둔다.
  }
}

/** 화면 열림(메인, 상세, 관리자 각 화면). */
export function pageView(pathname: string, deps?: EventDeps): void {
  sendEvent({ event: 'ui.page_view', page: cleanPath(pathname) }, deps)
}

/** 관리자 화면의 버튼을 누름. */
export function adminAction(action: AdminAction, deps?: EventDeps): void {
  sendEvent({ event: 'ui.admin_action', action }, deps)
}

/** 요청이 사용자의 취소(AbortController)로 끝났는지. 이것은 실패로 보내지 않는다. */
function isAbort(err: unknown): boolean {
  return err instanceof DOMException && err.name === 'AbortError'
}

/**
 * API 호출 실패를 보낸다. status는 응답 코드이고, 응답을 받지 못했으면(연결 실패) 0이다.
 * 정상 동작인 실패(로그인하지 않은 상태 확인 401)와 사용자의 취소는 보내지 않는다.
 */
export function apiError(path: string, status: number, deps?: EventDeps): void {
  const p = cleanPath(path)
  if (status === 401 && p === '/api/admin/session') return
  sendEvent({ event: 'ui.error', path: p, status }, deps)
}

/** fetch가 응답 없이 실패했을 때(err) 부른다. 취소면 보내지 않는다. */
export function apiNetworkError(path: string, err: unknown, deps?: EventDeps): void {
  if (isAbort(err)) return
  apiError(path, 0, deps)
}
