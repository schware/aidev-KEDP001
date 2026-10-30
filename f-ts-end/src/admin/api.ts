// Go 관리자 API의 요청·응답 형식이다. 기준: specification/40.api/README.md 5장
// session은 서버가 준 cookie(HttpOnly)만 쓴다. 비밀번호와 session을 브라우저 저장소에 두지 않는다.
import { ApiError, type Cycle, type ErrorBody } from '../api.ts'

export type SourceCode = 'ecos' | 'kosis' | 'data-go-kr'

/** 로그인 상태(5.3). */
export interface Session {
  account: string
  expires_at: string
}

/** 관리자 지표 하나(5.4). */
export interface AdminIndicator {
  id: string
  name: string
  source: SourceCode
  stat_code: string
  item_code: string | null
  classes: string[]
  org_id: string | null
  value_field: string | null
  cycle: Cycle
  unit: string | null
  source_unit: string | null
  visible: boolean
  /** 메인 화면 순서. 화면 설정이 없으면 null이다. */
  order: number | null
  chart_period: string | null
  decimals: number | null
  notice: string | null
  related: string[]
}

export interface AdminIndicatorList {
  indicators: AdminIndicator[]
}

/** 지표 수정 요청(5.6). 바꿀 수 있는 칸만, 모두 보낸다. */
export interface EditRequest {
  name: string
  cycle: Cycle
  unit: string | null
  chart_period: string
  decimals: number | null
  notice: string | null
  related: string[]
}

/** 지표 추가 요청(5.5). */
export interface AddRequest extends EditRequest {
  id: string
  source: SourceCode
  stat_code: string
  item_code: string | null
  classes: string[]
  org_id: string | null
  value_field: string | null
  visible: boolean
}

/** 노출과 순서 저장 요청(5.7). */
export interface DisplayOrderRequest {
  indicators: { id: string; visible: boolean; order: number }[]
}

export interface CollectTime {
  collect_time: string
}

export type SourceStatus = 'connected' | 'error' | 'unchecked' | 'no_key'

/** 제공처 상태(5.9). */
export interface SourceInfo {
  code: SourceCode
  name: string
  key_set: boolean
  last_collected_at: string | null
  status: SourceStatus
}

export interface SourceList {
  sources: SourceInfo[]
}

const base = '/api/admin'
export const loginPath = `${base}/login`
export const logoutPath = `${base}/logout`
export const sessionPath = `${base}/session`
export const indicatorsPath = `${base}/indicators`
export const displayOrderPath = `${base}/display-order`
export const collectTimePath = `${base}/collect-time`
export const sourcesPath = `${base}/sources`

export function indicatorPath(id: string): string {
  return `${indicatorsPath}/${encodeURIComponent(id)}`
}

/**
 * 관리자 API를 부른다. 200대가 아니면 ApiError를 던진다(400이면 fields가 붙는다).
 * POST, PUT은 Content-Type: application/json으로 보낸다(5.1). 204는 undefined다.
 */
export async function adminRequest<T>(
  method: 'GET' | 'POST' | 'PUT',
  path: string,
  body?: unknown,
  signal?: AbortSignal,
): Promise<T> {
  const headers: Record<string, string> = { Accept: 'application/json' }
  if (method !== 'GET') {
    headers['Content-Type'] = 'application/json'
  }
  const res = await fetch(path, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
    credentials: 'same-origin',
    signal,
  })
  if (!res.ok) {
    let errBody: ErrorBody | undefined
    try {
      errBody = (await res.json()) as ErrorBody
    } catch {
      errBody = undefined
    }
    throw new ApiError(res.status, errBody)
  }
  if (res.status === 204) {
    return undefined as T
  }
  return (await res.json()) as T
}

/** 401이면 로그인 화면으로, 403이면 "허용된 주소가 아닙니다"로 간다. 그 밖의 오류는 null이다. */
export function authFailure(err: unknown): 'login' | 'forbidden' | null {
  if (err instanceof ApiError) {
    if (err.status === 401) return 'login'
    if (err.status === 403) return 'forbidden'
  }
  return null
}

export type SessionCheck = { kind: 'ok'; session: Session } | { kind: 'login' } | { kind: 'forbidden' } | { kind: 'error' }

/** 관리자 화면에 들어올 때 로그인 상태를 확인한다(GET /api/admin/session). */
export async function checkSession(signal?: AbortSignal): Promise<SessionCheck> {
  try {
    const session = await adminRequest<Session>('GET', sessionPath, undefined, signal)
    return { kind: 'ok', session }
  } catch (err) {
    return { kind: authFailure(err) ?? 'error' }
  }
}
