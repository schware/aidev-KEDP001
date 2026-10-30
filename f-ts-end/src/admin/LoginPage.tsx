import { useEffect, useState, type FormEvent } from 'react'
import { Navigate, useNavigate } from 'react-router'
import { Message } from '../components/Layout.tsx'
import { ApiError } from '../api.ts'
import { adminRequest, authFailure, checkSession, loginPath, type Session, type SessionCheck } from './api.ts'
import { Field, Forbidden, inputClass, primaryButtonClass } from './ui.tsx'

/**
 * 관리자 로그인(/admin/login). 이미 로그인했으면 Dashboard 관리로 간다.
 * 비밀번호는 이 화면의 state에만 잠깐 둔다. 브라우저 저장소에 두지 않는다.
 */
export function LoginPage() {
  const [check, setCheck] = useState<SessionCheck | null>(null)

  useEffect(() => {
    const ctrl = new AbortController()
    checkSession(ctrl.signal).then((c) => {
      if (!ctrl.signal.aborted) setCheck(c)
    })
    return () => ctrl.abort()
  }, [])

  if (check === null) {
    return <Message>불러오는 중…</Message>
  }
  if (check.kind === 'ok') {
    return <Navigate to="/admin/dashboard" replace />
  }
  if (check.kind === 'forbidden') {
    return <Forbidden />
  }
  return <LoginForm onForbidden={() => setCheck({ kind: 'forbidden' })} />
}

function LoginForm({ onForbidden }: { onForbidden: () => void }) {
  const navigate = useNavigate()
  const [account, setAccount] = useState('')
  const [password, setPassword] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function submit(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      await adminRequest<Session>('POST', loginPath, { account, password })
      setPassword('')
      navigate('/admin/dashboard', { replace: true })
    } catch (err) {
      setPassword('')
      setBusy(false)
      if (authFailure(err) === 'forbidden') {
        onForbidden()
      } else if (err instanceof ApiError && (err.status === 401 || err.status === 400)) {
        setError('계정 또는 비밀번호가 맞지 않습니다')
      } else {
        setError('로그인하지 못했습니다. 잠시 뒤 다시 시도해 주세요.')
      }
    }
  }

  return (
    <div className="min-h-screen bg-slate-50 px-4 py-16 text-slate-900">
      <title>관리자 로그인 - KEDP</title>
      <form
        onSubmit={submit}
        className="mx-auto max-w-sm space-y-4 rounded-lg border border-slate-200 bg-white p-6 shadow-sm"
      >
        <h1 className="text-lg font-bold">KEDP 관리자 로그인</h1>
        <Field label="계정">
          <input
            className={inputClass}
            name="account"
            autoComplete="username"
            required
            value={account}
            onChange={(e) => setAccount(e.target.value)}
          />
        </Field>
        <Field label="비밀번호">
          <input
            className={inputClass}
            type="password"
            name="password"
            autoComplete="current-password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </Field>
        {error && (
          <p role="alert" className="text-sm text-red-600">
            {error}
          </p>
        )}
        <button type="submit" disabled={busy} className={`${primaryButtonClass} w-full`}>
          로그인
        </button>
      </form>
    </div>
  )
}
