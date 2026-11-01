import { useState, type FormEvent } from 'react'
import { ApiError } from '../api.ts'
import { formatValue } from '../format.ts'
import { adminRequest, visitBasePath, visitResetPath, visitsPath, type AdminVisits } from './api.ts'
import { useAdminAuth } from './auth.ts'
import { parseVisitBase } from './form.ts'
import { buttonClass, Field, inputClass, LoadError, primaryButtonClass } from './ui.tsx'
import { useAdminGet } from './useAdminGet.ts'
import { VisitsChart } from './VisitsChart.tsx'

export const resetConfirmText = '지금까지의 일별 기록이 지워집니다. 시작값도 0이 됩니다. 0부터 다시 시작할까요?'

/** 접속 통계(/admin/visits, 40.api 6.4~6.6): 오늘, 전체, 시작값, 최근 30일, 시작값 바꾸기, 0부터 다시 시작. */
export function VisitsPage() {
  const [loaded, reload] = useAdminGet<AdminVisits>(visitsPath)
  // 시작값 바꾸기와 초기화는 바뀐 뒤의 숫자를 돌려준다. 그것을 다시 읽지 않고 그대로 보인다.
  const [changed, setChanged] = useState<AdminVisits | null>(null)

  return (
    <div className="space-y-6">
      <title>접속 통계 - KEDP</title>
      <h1 className="text-xl font-bold">접속 통계</h1>
      {loaded.status === 'loading' && <p className="text-slate-500">불러오는 중…</p>}
      {loaded.status === 'error' && <LoadError onRetry={reload} />}
      {loaded.status === 'ok' && <VisitsView v={changed ?? loaded.data} onChange={setChanged} />}
    </div>
  )
}

function VisitsView({ v, onChange }: { v: AdminVisits; onChange: (v: AdminVisits) => void }) {
  // 초기화하면 시작값 입력칸을 새 값(0)으로 다시 만든다.
  const [resets, setResets] = useState(0)
  return (
    <>
      <dl className="grid gap-4 sm:grid-cols-3">
        <Stat label="오늘 (Asia/Seoul)" value={v.today} />
        <Stat label="전체" value={v.total} hint="시작값 + 일별 합" />
        <Stat label="시작값" value={v.base} />
      </dl>

      <section className="rounded-lg border border-slate-200 bg-white p-4">
        <h2 className="mb-3 text-sm font-semibold text-slate-500">최근 30일</h2>
        <VisitsChart days={v.days} />
      </section>

      <section className="rounded-lg border border-slate-200 bg-white p-4">
        <h2 className="mb-3 text-sm font-semibold text-slate-500">시작값 바꾸기</h2>
        <p className="mb-3 text-sm text-slate-500">예전 숫자를 이어 쓸 때 넣는다. 전체 = 시작값 + 일별 합이다.</p>
        <BaseEditor key={resets} current={v.base} onChange={onChange} />
      </section>

      <section className="rounded-lg border border-red-200 bg-white p-4">
        <h2 className="mb-3 text-sm font-semibold text-red-700">0부터 다시 시작</h2>
        <p className="mb-3 text-sm text-slate-500">
          일별 기록을 모두 지우고 시작값을 0으로 한다. 되돌릴 수 없다. 시험 기간 숫자를 지울 때 쓴다.
        </p>
        <ResetButton
          onChange={(next) => {
            onChange(next)
            setResets((n) => n + 1)
          }}
        />
      </section>
    </>
  )
}

function Stat({ label, value, hint }: { label: string; value: number; hint?: string }) {
  return (
    <div className="rounded-lg border border-slate-200 bg-white p-4">
      <dt className="text-sm text-slate-500">{label}</dt>
      <dd className="mt-1 text-2xl font-bold tabular-nums">{formatValue(value, 0)}</dd>
      {hint && <dd className="mt-1 text-xs text-slate-400">{hint}</dd>}
    </div>
  )
}

function BaseEditor({ current, onChange }: { current: number; onChange: (v: AdminVisits) => void }) {
  const { handleAuthError } = useAdminAuth()
  const [text, setText] = useState(String(current))
  const [error, setError] = useState<string>()
  const [busy, setBusy] = useState(false)
  const [done, setDone] = useState(false)

  async function submit(e: FormEvent) {
    e.preventDefault()
    setDone(false)
    const p = parseVisitBase(text)
    if ('error' in p) {
      setError(p.error)
      return
    }
    setError(undefined)
    setBusy(true)
    try {
      const res = await adminRequest<AdminVisits>('PUT', visitBasePath, { base: p.value })
      onChange(res)
      setText(String(res.base))
      setDone(true)
    } catch (err) {
      if (handleAuthError(err)) return
      if (err instanceof ApiError && err.status === 400) {
        setError(err.fields.base ?? err.message)
      } else {
        setError('저장하지 못했습니다. 잠시 뒤 다시 시도해 주세요.')
      }
    } finally {
      setBusy(false)
    }
  }

  return (
    <form onSubmit={submit} noValidate className="flex flex-wrap items-start gap-2">
      <div className="w-48">
        <Field label="시작값(0 이상 정수)" error={error}>
          <input
            className={inputClass}
            value={text}
            inputMode="numeric"
            onChange={(e) => {
              setText(e.target.value)
              setDone(false)
            }}
          />
        </Field>
      </div>
      <div className="flex items-center gap-3 pt-6">
        <button type="submit" className={primaryButtonClass} disabled={busy}>
          저장
        </button>
        {done && (
          <span role="status" className="text-sm text-green-700">
            저장했습니다.
          </span>
        )}
      </div>
    </form>
  )
}

function ResetButton({ onChange }: { onChange: (v: AdminVisits) => void }) {
  const { handleAuthError } = useAdminAuth()
  const [busy, setBusy] = useState(false)
  const [message, setMessage] = useState<{ ok: boolean; text: string }>()

  async function reset() {
    if (!window.confirm(resetConfirmText)) return
    setBusy(true)
    setMessage(undefined)
    try {
      onChange(await adminRequest<AdminVisits>('POST', visitResetPath, {}))
      setMessage({ ok: true, text: '0부터 다시 시작합니다.' })
    } catch (err) {
      if (handleAuthError(err)) return
      setMessage({ ok: false, text: '초기화하지 못했습니다. 잠시 뒤 다시 시도해 주세요.' })
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="flex flex-wrap items-center gap-3">
      <button
        type="button"
        disabled={busy}
        className={`${buttonClass} border-red-300 text-red-700 hover:bg-red-50`}
        onClick={reset}
      >
        0부터 다시 시작
      </button>
      {message && (
        <span role={message.ok ? 'status' : 'alert'} className={`text-sm ${message.ok ? 'text-green-700' : 'text-red-600'}`}>
          {message.text}
        </span>
      )}
    </div>
  )
}
