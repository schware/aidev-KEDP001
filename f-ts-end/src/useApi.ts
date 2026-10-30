import { useEffect, useState } from 'react'
import { ApiError, getJSON } from './api.ts'

export type ApiState<T> =
  | { status: 'loading' }
  | { status: 'ok'; data: T }
  | { status: 'not-found' }
  | { status: 'error' }

type Settled<T> = Exclude<ApiState<T>, { status: 'loading' }>

/** path를 GET한다. path가 바뀌면 다시 읽고, 이전 요청은 취소한다. */
export function useApi<T>(path: string): ApiState<T> {
  // 어느 path의 결과인지 함께 둔다. path가 바뀌면 결과가 오기 전까지 loading이다.
  const [result, setResult] = useState<{ path: string; state: Settled<T> } | null>(null)

  useEffect(() => {
    const ctrl = new AbortController()
    getJSON<T>(path, ctrl.signal).then(
      (data) => setResult({ path, state: { status: 'ok', data } }),
      (err: unknown) => {
        if (ctrl.signal.aborted) {
          return
        }
        const notFound = err instanceof ApiError && err.status === 404
        setResult({ path, state: { status: notFound ? 'not-found' : 'error' } })
      },
    )
    return () => ctrl.abort()
  }, [path])

  return result?.path === path ? result.state : { status: 'loading' }
}
