import { useState } from 'react'
import { Message } from '../components/Layout.tsx'
import { ApiError } from '../api.ts'
import { formatCycle } from '../format.ts'
import {
  adminRequest,
  displayOrderPath,
  indicatorsPath,
  type AdminIndicator,
  type AdminIndicatorList,
  type DisplayOrderRequest,
} from './api.ts'
import { useAdminAuth } from './auth.ts'
import { chartPeriodLabel } from './chartPeriod.ts'
import { AddDialog, EditDialog } from './IndicatorDialogs.tsx'
import { buttonClass, FormErrors, primaryButtonClass } from './ui.tsx'
import { useAdminGet } from './useAdminGet.ts'

/** Dashboard 관리(/admin/dashboard, 50.web 5장). */
export function DashboardPage() {
  const [state, reload] = useAdminGet<AdminIndicatorList>(indicatorsPath)
  return (
    <>
      <title>Dashboard 관리 - KEDP</title>
      <h1 className="mb-4 text-xl font-bold">Dashboard 관리</h1>
      {state.status === 'loading' && <Message>불러오는 중…</Message>}
      {state.status === 'error' && (
        <div className="py-12 text-center text-slate-500">
          <p>지표 목록을 불러오지 못했습니다.</p>
          <button type="button" className={`${buttonClass} mt-4`} onClick={reload}>
            다시 읽기
          </button>
        </div>
      )}
      {state.status === 'ok' && <Editor initial={state.data.indicators} />}
    </>
  )
}

/** 노출과 순서. 화면 설정이 있는 지표(order가 있는 지표)만 들어 있다. */
type Draft = { id: string; visible: boolean }[]

function draftOf(list: AdminIndicator[]): Draft {
  return list
    .filter((x) => x.order !== null)
    .sort((a, b) => (a.order ?? 0) - (b.order ?? 0) || a.id.localeCompare(b.id))
    .map((x) => ({ id: x.id, visible: x.visible }))
}

function sameDraft(a: Draft, b: Draft): boolean {
  return a.length === b.length && a.every((x, i) => x.id === b[i].id && x.visible === b[i].visible)
}

function Editor({ initial }: { initial: AdminIndicator[] }) {
  const { handleAuthError } = useAdminAuth()
  const [items, setItems] = useState(initial)
  // saved: 서버에 저장된 노출과 순서, draft: 화면에서 바꾸는 중인 것. 둘이 다르면 저장하지 않은 변경이다.
  const [saved, setSaved] = useState(() => draftOf(initial))
  const [draft, setDraft] = useState(saved)
  const [editing, setEditing] = useState<AdminIndicator | null>(null)
  const [adding, setAdding] = useState(false)
  const [busy, setBusy] = useState(false)
  const [notice, setNotice] = useState<string | null>(null)
  const [saveErrors, setSaveErrors] = useState<Record<string, string>>({})
  const [saveMessage, setSaveMessage] = useState<string>()

  const dirty = !sameDraft(draft, saved)
  const byId = new Map(items.map((x) => [x.id, x]))
  const unconfigured = items.filter((x) => x.order === null)

  function move(i: number, d: -1 | 1) {
    setDraft((cur) => {
      const next = [...cur]
      ;[next[i], next[i + d]] = [next[i + d], next[i]]
      return next
    })
  }

  function toggle(i: number) {
    setDraft((cur) => cur.map((x, j) => (j === i ? { ...x, visible: !x.visible } : x)))
  }

  /** 서버가 새로 화면 설정을 만든 지표(맨 뒤)를 saved와 draft 끝에 붙인다. */
  function appendConfigured(ind: AdminIndicator) {
    const add = (d: Draft) => (ind.order !== null && !d.some((x) => x.id === ind.id) ? [...d, { id: ind.id, visible: ind.visible }] : d)
    setSaved(add)
    setDraft(add)
  }

  async function save() {
    setBusy(true)
    setNotice(null)
    setSaveErrors({})
    setSaveMessage(undefined)
    const body: DisplayOrderRequest = { indicators: draft.map((x, i) => ({ id: x.id, visible: x.visible, order: i + 1 })) }
    try {
      const res = await adminRequest<AdminIndicatorList>('PUT', displayOrderPath, body)
      setItems(res.indicators)
      const d = draftOf(res.indicators)
      setSaved(d)
      setDraft(d)
      setNotice('저장했습니다. 메인 화면에 반영됩니다.')
    } catch (err) {
      if (handleAuthError(err)) return
      if (err instanceof ApiError && err.status === 400) {
        setSaveErrors(err.fields)
        setSaveMessage('저장하지 못했습니다. 목록을 다시 읽은 뒤 해 주세요.')
      } else {
        setSaveMessage('저장하지 못했습니다. 잠시 뒤 다시 시도해 주세요.')
      }
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-2">
        <h2 className="font-semibold">메인 화면 노출 지표</h2>
        <button type="button" className={`${buttonClass} ml-auto`} onClick={() => setAdding(true)}>
          지표 추가
        </button>
      </div>

      {notice && (
        <p role="status" className="rounded-md border border-green-200 bg-green-50 px-3 py-2 text-sm text-green-800">
          {notice}
        </p>
      )}

      <div className="overflow-x-auto rounded-lg border border-slate-200 bg-white">
        <table className="w-full text-sm">
          <thead className="bg-slate-50 text-left text-slate-500">
            <tr>
              <th className="px-3 py-2 font-medium">순서</th>
              <th className="px-3 py-2 font-medium">노출</th>
              <th className="px-3 py-2 font-medium">지표명</th>
              <th className="px-3 py-2 font-medium">주기</th>
              <th className="px-3 py-2 font-medium">차트 기간</th>
              <th className="px-3 py-2 font-medium">설정</th>
            </tr>
          </thead>
          <tbody>
            {draft.map((d, i) => {
              const ind = byId.get(d.id)
              if (!ind) return null
              return (
                <tr key={d.id} className="border-t border-slate-100">
                  <td className="whitespace-nowrap px-3 py-2">
                    <span className="inline-block w-6 tabular-nums">{i + 1}</span>
                    <button
                      type="button"
                      className={`${buttonClass} px-2 py-0.5`}
                      disabled={i === 0}
                      onClick={() => move(i, -1)}
                      aria-label={`${ind.name} 위로`}
                    >
                      ↑
                    </button>
                    <button
                      type="button"
                      className={`${buttonClass} ml-1 px-2 py-0.5`}
                      disabled={i === draft.length - 1}
                      onClick={() => move(i, 1)}
                      aria-label={`${ind.name} 아래로`}
                    >
                      ↓
                    </button>
                  </td>
                  <td className="px-3 py-2">
                    <button
                      type="button"
                      aria-pressed={d.visible}
                      aria-label={`${ind.name} 노출`}
                      onClick={() => toggle(i)}
                      className={`w-14 rounded-full px-2 py-0.5 text-xs font-semibold ${
                        d.visible ? 'bg-blue-600 text-white' : 'bg-slate-200 text-slate-600'
                      }`}
                    >
                      {d.visible ? 'ON' : 'OFF'}
                    </button>
                  </td>
                  <IndicatorCells ind={ind} onEdit={() => setEditing(ind)} />
                </tr>
              )
            })}
            {unconfigured.map((ind) => (
              <tr key={ind.id} className="border-t border-slate-100 text-slate-500">
                <td className="px-3 py-2">-</td>
                <td className="px-3 py-2 text-xs">화면 설정 없음</td>
                <IndicatorCells ind={ind} onEdit={() => setEditing(ind)} />
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {unconfigured.length > 0 && (
        <p className="text-xs text-slate-500">
          화면 설정이 없는 지표는 [편집]에서 차트 기간을 넣어 저장하면 목록 맨 뒤에 들어온다(노출 안 함).
        </p>
      )}

      <div className="flex flex-wrap items-center gap-3">
        <button type="button" className={primaryButtonClass} disabled={busy || !dirty} onClick={save}>
          저장
        </button>
        {dirty && <span className="text-sm font-medium text-amber-700">저장하지 않은 변경이 있습니다</span>}
        {dirty && (
          <button type="button" className={buttonClass} disabled={busy} onClick={() => setDraft(saved)}>
            되돌리기
          </button>
        )}
      </div>
      <FormErrors errors={saveErrors} known={[]} message={saveMessage} />

      {editing && (
        <EditDialog
          indicator={editing}
          all={items}
          onClose={() => setEditing(null)}
          onSaved={(updated) => {
            setItems((cur) => cur.map((x) => (x.id === updated.id ? updated : x)))
            appendConfigured(updated)
            setEditing(null)
            setNotice(`'${updated.name}' 설정을 저장했습니다.`)
          }}
        />
      )}
      {adding && (
        <AddDialog
          all={items}
          onClose={() => setAdding(false)}
          onAdded={(added) => {
            setItems((cur) => [...cur, added])
            appendConfigured(added)
            setAdding(false)
            setNotice(`'${added.name}'을(를) 추가했습니다. 값은 다음 수집 때 들어옵니다.`)
          }}
        />
      )}
    </div>
  )
}

function IndicatorCells({ ind, onEdit }: { ind: AdminIndicator; onEdit: () => void }) {
  return (
    <>
      <td className="px-3 py-2">
        <span className="font-medium text-slate-800">{ind.name}</span>
        <span className="block text-xs text-slate-400">{ind.id}</span>
      </td>
      <td className="whitespace-nowrap px-3 py-2" title={formatCycle(ind.cycle)}>
        {ind.cycle}
      </td>
      <td className="whitespace-nowrap px-3 py-2">{chartPeriodLabel(ind.chart_period)}</td>
      <td className="px-3 py-2">
        <button type="button" className={buttonClass} onClick={onEdit}>
          편집
        </button>
      </td>
    </>
  )
}
