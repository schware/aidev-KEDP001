// Dashboard 관리의 [편집]과 [지표 추가] 창(50.web 5장, 40.api 5.5, 5.6).
import { useState, type FormEvent, type ReactNode } from 'react'
import { ApiError } from '../api.ts'
import { adminRequest, indicatorPath, indicatorsPath, type AdminIndicator } from './api.ts'
import { useAdminAuth } from './auth.ts'
import { periodUnits, type PeriodUnit } from './chartPeriod.ts'
import {
  addBody,
  cycles,
  editBody,
  emptyAddForm,
  settingsFromIndicator,
  sourceFields,
  sourceLabel,
  sources,
  validateAdd,
  validateEdit,
  type AddForm,
  type FieldErrors,
  type SettingsForm,
} from './form.ts'
import { buttonClass, Field, FormErrors, inputClass, primaryButtonClass } from './ui.tsx'

const settingsKeys = ['name', 'cycle', 'unit', 'chart_period', 'decimals', 'notice', 'related']
const addKeys = [...settingsKeys, 'id', 'source', 'stat_code', 'item_code', 'org_id', 'classes', 'value_field', 'visible']

/** 저장 요청의 오류를 칸 오류나 일반 오류로 바꾼다. 401/403은 useAdminAuth가 처리한다. */
function useSaveError() {
  const { handleAuthError } = useAdminAuth()
  return (err: unknown, notFound = '지표를 찾을 수 없습니다'): { fields: FieldErrors; message?: string } | null => {
    if (handleAuthError(err)) return null
    if (err instanceof ApiError) {
      if (err.status === 400) {
        const fields = err.fields
        return Object.keys(fields).length > 0 ? { fields } : { fields: {}, message: `입력이 맞지 않습니다(${err.message})` }
      }
      if (err.status === 404) return { fields: {}, message: notFound }
      if (err.status === 409) return { fields: { id: '같은 id가 이미 있습니다' } }
    }
    return { fields: {}, message: '저장하지 못했습니다. 잠시 뒤 다시 시도해 주세요.' }
  }
}

/** [편집]: 이름, 주기, 단위, 차트 기간, 소수 자릿수, 안내 문구, 관련 지표를 바꾼다. id, 제공처, 코드는 보기만 한다. */
export function EditDialog({
  indicator,
  all,
  onClose,
  onSaved,
}: {
  indicator: AdminIndicator
  all: AdminIndicator[]
  onClose: () => void
  onSaved: (updated: AdminIndicator) => void
}) {
  const toSaveError = useSaveError()
  const [form, setForm] = useState<SettingsForm>(() => settingsFromIndicator(indicator))
  const [errors, setErrors] = useState<FieldErrors>({})
  const [message, setMessage] = useState<string>()
  const [busy, setBusy] = useState(false)

  async function submit(e: FormEvent) {
    e.preventDefault()
    const v = validateEdit(form, indicator.id)
    setErrors(v)
    setMessage(undefined)
    if (Object.keys(v).length > 0) return
    if (form.cycle !== indicator.cycle && !window.confirm('주기를 바꾸면 이전 주기 값은 화면에 보이지 않게 됩니다. 바꿀까요?')) {
      return
    }
    setBusy(true)
    try {
      onSaved(await adminRequest<AdminIndicator>('PUT', indicatorPath(indicator.id), editBody(form)))
    } catch (err) {
      setBusy(false)
      const r = toSaveError(err)
      if (r) {
        setErrors(r.fields)
        setMessage(r.message)
      }
    }
  }

  const codes = [
    ['id', indicator.id],
    ['제공처', sourceLabel(indicator.source)],
    ...sourceFields(indicator.source).map(({ field, label }): [string, string] => [
      label,
      field === 'classes' ? indicator.classes.join(', ') || '-' : (indicator[field] ?? '-'),
    ]),
  ]

  return (
    <Dialog title={`지표 편집: ${indicator.name}`} onClose={onClose}>
      <form onSubmit={submit} noValidate className="space-y-4">
        <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 rounded-md bg-slate-50 p-3 text-sm">
          {codes.map(([k, v]) => (
            <div key={k} className="contents">
              <dt className="text-slate-500">{k}</dt>
              <dd className="break-all">{v}</dd>
            </div>
          ))}
        </dl>
        <SettingsFields
          form={form}
          onChange={(p) => setForm((f) => ({ ...f, ...p }))}
          errors={errors}
          all={all}
          selfId={indicator.id}
          sourceUnit={indicator.source_unit}
        />
        <FormErrors errors={errors} known={settingsKeys} message={message} />
        <DialogButtons busy={busy} onClose={onClose} />
      </form>
    </Dialog>
  )
}

/** [지표 추가]: 제공처를 고르면 그 제공처에 필요한 칸만 보인다. */
export function AddDialog({
  all,
  onClose,
  onAdded,
}: {
  all: AdminIndicator[]
  onClose: () => void
  onAdded: (added: AdminIndicator) => void
}) {
  const toSaveError = useSaveError()
  const [form, setForm] = useState<AddForm>(emptyAddForm)
  const [errors, setErrors] = useState<FieldErrors>({})
  const [message, setMessage] = useState<string>()
  const [busy, setBusy] = useState(false)
  const set = (p: Partial<AddForm>) => setForm((f) => ({ ...f, ...p }))

  async function submit(e: FormEvent) {
    e.preventDefault()
    const v = validateAdd(
      form,
      all.map((x) => x.id),
    )
    setErrors(v)
    setMessage(undefined)
    if (Object.keys(v).length > 0) return
    setBusy(true)
    try {
      onAdded(await adminRequest<AdminIndicator>('POST', indicatorsPath, addBody(form)))
    } catch (err) {
      setBusy(false)
      const r = toSaveError(err)
      if (r) {
        setErrors(r.fields)
        setMessage(r.message)
      }
    }
  }

  return (
    <Dialog title="지표 추가" onClose={onClose}>
      <form onSubmit={submit} noValidate className="space-y-4">
        <Field label="제공처" error={errors.source}>
          <select
            className={inputClass}
            value={form.source}
            onChange={(e) => set({ source: e.target.value as AddForm['source'] })}
          >
            {sources.map((s) => (
              <option key={s.value} value={s.value}>
                {s.label}
              </option>
            ))}
          </select>
        </Field>
        <div className="grid gap-4 sm:grid-cols-2">
          {sourceFields(form.source).map(({ field, label }) => (
            <Field key={field} label={label} error={errors[field]}>
              <input className={inputClass} value={form[field]} onChange={(e) => set({ [field]: e.target.value })} />
            </Field>
          ))}
        </div>
        <Field label="id" error={errors.id} hint="영문 소문자와 숫자를 -로 잇는다(예: kosis-kospi). 나중에 바꿀 수 없다">
          <input className={inputClass} value={form.id} onChange={(e) => set({ id: e.target.value })} />
        </Field>
        <SettingsFields form={form} onChange={set} errors={errors} all={all} selfId={form.id.trim()} sourceUnit={null} />
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" checked={form.visible} onChange={(e) => set({ visible: e.target.checked })} />
          메인 화면에 보이기(순서는 맨 뒤)
        </label>
        <FormErrors errors={errors} known={addKeys} message={message} />
        <DialogButtons busy={busy} onClose={onClose} />
      </form>
    </Dialog>
  )
}

/** 편집과 추가에 같이 쓰는 화면 설정 칸. */
function SettingsFields({
  form,
  onChange,
  errors,
  all,
  selfId,
  sourceUnit,
}: {
  form: SettingsForm
  onChange: (p: Partial<SettingsForm>) => void
  errors: FieldErrors
  all: AdminIndicator[]
  selfId: string
  sourceUnit: string | null
}) {
  return (
    <>
      <Field label="이름" error={errors.name}>
        <input className={inputClass} value={form.name} onChange={(e) => onChange({ name: e.target.value })} />
      </Field>
      <div className="grid gap-4 sm:grid-cols-2">
        <Field label="주기" error={errors.cycle}>
          <select
            className={inputClass}
            value={form.cycle}
            onChange={(e) => onChange({ cycle: e.target.value as SettingsForm['cycle'] })}
          >
            {cycles.map((c) => (
              <option key={c.value} value={c.value}>
                {c.label}
              </option>
            ))}
          </select>
        </Field>
        <Field
          label="단위"
          error={errors.unit}
          hint={sourceUnit ? `비우면 응답 단위(${sourceUnit})를 쓴다` : '비우면 단위 없음(응답 단위가 있으면 그것)'}
        >
          <input className={inputClass} value={form.unit} onChange={(e) => onChange({ unit: e.target.value })} />
        </Field>
        <Field label="차트 기간" error={errors.chart_period}>
          <span className="flex gap-2">
            <input
              className={inputClass}
              inputMode="numeric"
              aria-label="차트 기간 숫자"
              value={form.periodN}
              onChange={(e) => onChange({ periodN: e.target.value })}
            />
            <select
              className={`${inputClass} w-28`}
              aria-label="차트 기간 단위"
              value={form.periodUnit}
              onChange={(e) => onChange({ periodUnit: e.target.value as PeriodUnit })}
            >
              {periodUnits.map((u) => (
                <option key={u.value} value={u.value}>
                  {u.label}
                </option>
              ))}
            </select>
          </span>
        </Field>
        <Field label="소수 자릿수" error={errors.decimals}>
          <select className={inputClass} value={form.decimals} onChange={(e) => onChange({ decimals: e.target.value })}>
            <option value="">없음(받은 값 그대로)</option>
            {['0', '1', '2', '3', '4'].map((d) => (
              <option key={d} value={d}>
                {d}
              </option>
            ))}
          </select>
        </Field>
      </div>
      <Field label="안내 문구" error={errors.notice} hint="상세 화면 맨 위에 보인다. 비우면 없음">
        <textarea
          className={inputClass}
          rows={2}
          value={form.notice}
          onChange={(e) => onChange({ notice: e.target.value })}
        />
      </Field>
      <RelatedPicker
        value={form.related}
        onChange={(related) => onChange({ related })}
        all={all}
        selfId={selfId}
        error={errors.related}
      />
    </>
  )
}

/** 관련 지표: 전체 지표에서 고르고 순서를 정한다. 자기 자신은 고를 수 없다. */
function RelatedPicker({
  value,
  onChange,
  all,
  selfId,
  error,
}: {
  value: string[]
  onChange: (ids: string[]) => void
  all: AdminIndicator[]
  selfId: string
  error?: string
}) {
  const [pick, setPick] = useState('')
  const nameOf = (id: string) => all.find((x) => x.id === id)?.name ?? id
  const choices = all.filter((x) => x.id !== selfId && !value.includes(x.id))
  const move = (i: number, d: -1 | 1) => {
    const next = [...value]
    ;[next[i], next[i + d]] = [next[i + d], next[i]]
    onChange(next)
  }

  return (
    <fieldset className="text-sm">
      <legend className="mb-1 font-medium text-slate-700">관련 지표(상세 화면에 이 순서로 보인다)</legend>
      {value.length === 0 ? (
        <p className="text-slate-400">없음</p>
      ) : (
        <ol className="space-y-1">
          {value.map((id, i) => (
            <li key={id} className="flex items-center gap-2">
              <span className="w-5 text-right text-slate-400">{i + 1}</span>
              <span className="flex-1">
                {nameOf(id)} <span className="text-xs text-slate-400">{id}</span>
              </span>
              <button type="button" className={buttonClass} disabled={i === 0} onClick={() => move(i, -1)} aria-label="위로">
                ↑
              </button>
              <button
                type="button"
                className={buttonClass}
                disabled={i === value.length - 1}
                onClick={() => move(i, 1)}
                aria-label="아래로"
              >
                ↓
              </button>
              <button type="button" className={buttonClass} onClick={() => onChange(value.filter((x) => x !== id))}>
                빼기
              </button>
            </li>
          ))}
        </ol>
      )}
      <span className="mt-2 flex gap-2">
        <select className={inputClass} aria-label="더할 관련 지표" value={pick} onChange={(e) => setPick(e.target.value)}>
          <option value="">지표 고르기</option>
          {choices.map((x) => (
            <option key={x.id} value={x.id}>
              {x.name} ({x.id})
            </option>
          ))}
        </select>
        <button
          type="button"
          className={buttonClass}
          disabled={pick === '' || !choices.some((x) => x.id === pick)}
          onClick={() => {
            onChange([...value, pick])
            setPick('')
          }}
        >
          더하기
        </button>
      </span>
      {error && (
        <span role="alert" className="mt-1 block text-xs text-red-600">
          {error}
        </span>
      )}
    </fieldset>
  )
}

function Dialog({ title, onClose, children }: { title: string; onClose: () => void; children: ReactNode }) {
  return (
    <div
      className="fixed inset-0 z-10 overflow-y-auto bg-slate-900/40 px-4 py-8"
      onKeyDown={(e) => e.key === 'Escape' && onClose()}
    >
      <section
        role="dialog"
        aria-modal="true"
        aria-label={title}
        className="mx-auto max-w-2xl rounded-lg bg-white p-5 shadow-lg"
      >
        <h2 className="mb-4 text-lg font-bold">{title}</h2>
        {children}
      </section>
    </div>
  )
}

function DialogButtons({ busy, onClose }: { busy: boolean; onClose: () => void }) {
  return (
    <div className="flex justify-end gap-2 pt-2">
      <button type="button" className={buttonClass} onClick={onClose}>
        취소
      </button>
      <button type="submit" className={primaryButtonClass} disabled={busy}>
        저장
      </button>
    </div>
  )
}
