import { useState, type FormEvent } from 'react'
import { Message } from '../components/Layout.tsx'
import { ApiError } from '../api.ts'
import { formatDateTime } from '../format.ts'
import {
  adminRequest,
  collectTimePath,
  sourcesPath,
  type CollectTime,
  type SourceInfo,
  type SourceList,
  type SourceStatus,
} from './api.ts'
import { useAdminAuth } from './auth.ts'
import { collectTimeError } from './form.ts'
import { buttonClass, Field, inputClass, primaryButtonClass } from './ui.tsx'
import { useAdminGet } from './useAdminGet.ts'

/** API 관리(/admin/api, 50.web 6장). 키 값을 넣거나 보여 주는 칸은 없다. */
export function ApiPage() {
  const [time, reloadTime] = useAdminGet<CollectTime>(collectTimePath)
  const [sources, reloadSources] = useAdminGet<SourceList>(sourcesPath)

  return (
    <div className="space-y-6">
      <title>API 관리 - KEDP</title>
      <h1 className="text-xl font-bold">API 관리</h1>

      <section className="rounded-lg border border-slate-200 bg-white p-4">
        <h2 className="mb-3 text-sm font-semibold text-slate-500">수집 시각</h2>
        {time.status === 'loading' && <p className="text-slate-500">불러오는 중…</p>}
        {time.status === 'error' && <LoadError onRetry={reloadTime} />}
        {time.status === 'ok' && <CollectTimeEditor initial={time.data.collect_time} />}
      </section>

      <section>
        <h2 className="mb-3 text-sm font-semibold text-slate-500">제공처</h2>
        {sources.status === 'loading' && <Message>불러오는 중…</Message>}
        {sources.status === 'error' && <LoadError onRetry={reloadSources} />}
        {sources.status === 'ok' && (
          <ul className="grid gap-4 md:grid-cols-3">
            {sources.data.sources.map((s) => (
              <li key={s.code}>
                <SourceCard s={s} />
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  )
}

function LoadError({ onRetry }: { onRetry: () => void }) {
  return (
    <p className="text-slate-500">
      불러오지 못했습니다.{' '}
      <button type="button" className={buttonClass} onClick={onRetry}>
        다시 읽기
      </button>
    </p>
  )
}

/** 지금 수집 시각과 [변경]. HH:MM으로 받는다(40.api 5.8). */
function CollectTimeEditor({ initial }: { initial: string }) {
  const { handleAuthError } = useAdminAuth()
  const [current, setCurrent] = useState(initial)
  const [editing, setEditing] = useState(false)
  const [text, setText] = useState(initial)
  const [error, setError] = useState<string>()
  const [busy, setBusy] = useState(false)
  const [done, setDone] = useState(false)

  async function submit(e: FormEvent) {
    e.preventDefault()
    const v = collectTimeError(text)
    setError(v ?? undefined)
    if (v) return
    setBusy(true)
    try {
      const res = await adminRequest<CollectTime>('PUT', collectTimePath, { collect_time: text })
      setCurrent(res.collect_time)
      setEditing(false)
      setDone(true)
    } catch (err) {
      if (handleAuthError(err)) return
      if (err instanceof ApiError && err.status === 400) {
        setError(err.fields.collect_time ?? err.message)
      } else {
        setError('저장하지 못했습니다. 잠시 뒤 다시 시도해 주세요.')
      }
    } finally {
      setBusy(false)
    }
  }

  if (!editing) {
    return (
      <div className="flex flex-wrap items-center gap-3">
        <span>
          매일 <strong className="tabular-nums">{current}</strong> (Asia/Seoul)
        </span>
        <button
          type="button"
          className={buttonClass}
          onClick={() => {
            setText(current)
            setError(undefined)
            setDone(false)
            setEditing(true)
          }}
        >
          변경
        </button>
        {done && (
          <span role="status" className="text-sm text-green-700">
            저장했습니다. 다음에 오는 그 시각부터 따른다.
          </span>
        )}
      </div>
    )
  }

  return (
    <form onSubmit={submit} noValidate className="flex flex-wrap items-start gap-2">
      <div className="w-32">
        <Field label="수집 시각(HH:MM)" error={error}>
          <input
            className={inputClass}
            value={text}
            placeholder="06:00"
            inputMode="numeric"
            onChange={(e) => setText(e.target.value)}
          />
        </Field>
      </div>
      <div className="flex gap-2 pt-6">
        <button type="submit" className={primaryButtonClass} disabled={busy}>
          저장
        </button>
        <button type="button" className={buttonClass} onClick={() => setEditing(false)}>
          취소
        </button>
      </div>
    </form>
  )
}

const statusView: Record<SourceStatus, { label: string; dot: string; text: string }> = {
  connected: { label: '연결', dot: 'bg-green-500', text: 'text-green-700' },
  error: { label: '오류', dot: 'bg-red-500', text: 'text-red-700' },
  unchecked: { label: '확인 전', dot: 'bg-slate-400', text: 'text-slate-600' },
  no_key: { label: '연결 안 됨', dot: 'bg-amber-500', text: 'text-amber-700' },
}

function SourceCard({ s }: { s: SourceInfo }) {
  const st = statusView[s.status] ?? { label: s.status, dot: 'bg-slate-400', text: 'text-slate-600' }
  return (
    <div className="h-full rounded-lg border border-slate-200 bg-white p-4">
      <h3 className="mb-3 font-semibold">{s.name}</h3>
      <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1.5 text-sm">
        <dt className="text-slate-500">상태</dt>
        <dd className={`flex items-center gap-1.5 font-medium ${st.text}`}>
          <span aria-hidden="true" className={`inline-block h-2.5 w-2.5 rounded-full ${st.dot}`} />
          {st.label}
        </dd>
        <dt className="text-slate-500">API 키</dt>
        <dd>{s.key_set ? '설정됨 (.env)' : '설정 안 됨'}</dd>
        <dt className="text-slate-500">마지막 수집</dt>
        <dd>{s.last_collected_at ? formatDateTime(s.last_collected_at) : '없음'}</dd>
      </dl>
    </div>
  )
}
