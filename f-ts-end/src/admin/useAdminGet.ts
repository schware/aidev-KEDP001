import { useCallback, useEffect, useState } from 'react'
import { adminRequest } from './api.ts'
import { useAdminAuth } from './auth.ts'

export type AdminGetState<T> = { status: 'loading' } | { status: 'ok'; data: T } | { status: 'error' }

/**
 * 관리자 API를 GET한다. reload()로 다시 읽는다.
 * 401/403은 useAdminAuth가 로그인 화면이나 "허용된 주소가 아닙니다"로 바꾼다.
 */
export function useAdminGet<T>(path: string): [AdminGetState<T>, () => void] {
  const { handleAuthError } = useAdminAuth()
  const [version, setVersion] = useState(0)
  const key = `${path}#${version}`
  const [result, setResult] = useState<{ key: string; state: AdminGetState<T> } | null>(null)

  useEffect(() => {
    const ctrl = new AbortController()
    adminRequest<T>('GET', path, undefined, ctrl.signal).then(
      (data) => setResult({ key, state: { status: 'ok', data } }),
      (err: unknown) => {
        if (ctrl.signal.aborted || handleAuthError(err)) {
          return
        }
        setResult({ key, state: { status: 'error' } })
      },
    )
    return () => ctrl.abort()
  }, [path, key, handleAuthError])

  const reload = useCallback(() => setVersion((v) => v + 1), [])
  return [result?.key === key ? result.state : { status: 'loading' }, reload]
}
