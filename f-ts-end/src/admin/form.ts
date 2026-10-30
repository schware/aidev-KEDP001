// 관리자 입력칸의 검사와 요청 본문 만들기. React와 상관없이 시험할 수 있게 따로 둔다.
// 오류는 API 칸 이름(40.api 5.5)을 열쇠로 둔다. 서버의 400 fields와 같은 자리에 보인다.
import type { Cycle } from '../api.ts'
import type { AddRequest, AdminIndicator, EditRequest, SourceCode } from './api.ts'
import { chartPeriodNumberError, formatChartPeriod, parseChartPeriod, type PeriodUnit } from './chartPeriod.ts'

export type FieldErrors = Record<string, string>

export const cycles: { value: Cycle; label: string }[] = [
  { value: 'D', label: 'D (일간)' },
  { value: 'M', label: 'M (월간)' },
  { value: 'Q', label: 'Q (분기)' },
  { value: 'A', label: 'A (연간)' },
]

export const sources: { value: SourceCode; label: string }[] = [
  { value: 'ecos', label: 'ECOS' },
  { value: 'kosis', label: 'KOSIS' },
  { value: 'data-go-kr', label: '공공데이터포털' },
]

export function sourceLabel(code: string): string {
  return sources.find((s) => s.value === code)?.label ?? code
}

export type CodeField = 'org_id' | 'stat_code' | 'item_code' | 'classes' | 'value_field'

/** 제공처마다 보여 줄 코드 칸과 이름. 이 칸만 보이고 검사한다. */
export function sourceFields(source: SourceCode): { field: CodeField; label: string }[] {
  switch (source) {
    case 'ecos':
      return [
        { field: 'stat_code', label: '통계 코드' },
        { field: 'item_code', label: '항목 코드' },
      ]
    case 'kosis':
      return [
        { field: 'org_id', label: '기관 코드(org_id)' },
        { field: 'stat_code', label: '통계 코드(tblId)' },
        { field: 'item_code', label: '항목 코드(itmId)' },
        { field: 'classes', label: '분류(objL1, objL2… 쉼표로 나눈다)' },
      ]
    case 'data-go-kr':
      return [
        { field: 'stat_code', label: '종목 코드' },
        { field: 'value_field', label: 'value_field' },
      ]
  }
}

/** 화면 설정 칸(편집과 추가에 같이 쓴다). 숫자와 선택값도 입력칸 그대로 글로 둔다. */
export interface SettingsForm {
  name: string
  cycle: Cycle
  unit: string
  periodN: string
  periodUnit: PeriodUnit
  /** '' = 없음, '0'~'4' */
  decimals: string
  notice: string
  related: string[]
}

export interface AddForm extends SettingsForm {
  id: string
  source: SourceCode
  stat_code: string
  item_code: string
  org_id: string
  classes: string
  value_field: string
  visible: boolean
}

export function emptyAddForm(): AddForm {
  return {
    id: '',
    name: '',
    source: 'ecos',
    stat_code: '',
    item_code: '',
    org_id: '',
    classes: '',
    value_field: 'clpr',
    cycle: 'M',
    unit: '',
    visible: false,
    periodN: '10',
    periodUnit: 'years',
    decimals: '',
    notice: '',
    related: [],
  }
}

/** 지표에서 편집 칸을 만든다. 차트 기간이 없으면 숫자 칸은 비운다. */
export function settingsFromIndicator(ind: AdminIndicator): SettingsForm {
  const p = parseChartPeriod(ind.chart_period)
  return {
    name: ind.name,
    cycle: ind.cycle,
    unit: ind.unit ?? '',
    periodN: p ? String(p.n) : '',
    periodUnit: p?.unit ?? 'years',
    decimals: ind.decimals === null ? '' : String(ind.decimals),
    notice: ind.notice ?? '',
    related: [...ind.related],
  }
}

/** 분류 글 `00, 01` → ['00', '01']. 비어 있으면 []. */
export function parseClasses(text: string): string[] {
  return text.trim() === '' ? [] : text.split(',').map((s) => s.trim())
}

const blank = (s: string) => s.trim() === ''
const orNull = (s: string) => (blank(s) ? null : s.trim())

function validateSettings(f: SettingsForm, selfId: string): FieldErrors {
  const e: FieldErrors = {}
  if (blank(f.name)) e.name = '이름을 넣어 주세요'
  const pe = chartPeriodNumberError(f.periodN)
  if (pe) e.chart_period = pe
  if (f.decimals !== '' && !/^[0-4]$/.test(f.decimals)) e.decimals = '없음 또는 0~4 중에서 골라 주세요'
  if (f.related.includes(selfId)) e.related = '자기 자신은 고를 수 없습니다'
  else if (new Set(f.related).size !== f.related.length) e.related = '같은 지표를 두 번 고를 수 없습니다'
  return e
}

/** 지표 수정 칸 검사(40.api 5.6). 맞으면 빈 객체다. */
export function validateEdit(f: SettingsForm, selfId: string): FieldErrors {
  return validateSettings(f, selfId)
}

/** 지표 추가 칸 검사(40.api 5.5). 그 제공처에 보이는 칸만 검사한다. 맞으면 빈 객체다. */
export function validateAdd(f: AddForm, existingIds: string[]): FieldErrors {
  const e = validateSettings(f, f.id.trim())
  const id = f.id.trim()
  if (!/^[a-z0-9]+(-[a-z0-9]+)*$/.test(id)) e.id = '영문 소문자와 숫자를 -로 이어 주세요(예: kosis-kospi)'
  else if (existingIds.includes(id)) e.id = '같은 id가 이미 있습니다'
  for (const { field } of sourceFields(f.source)) {
    if (field === 'classes') {
      if (parseClasses(f.classes).some((c) => c === '')) e.classes = '빈 분류 코드가 있습니다'
    } else if (blank(f[field])) {
      e[field] = '꼭 넣어 주세요'
    }
  }
  return e
}

function settingsBody(f: SettingsForm): EditRequest {
  return {
    name: f.name.trim(),
    cycle: f.cycle,
    unit: orNull(f.unit),
    chart_period: formatChartPeriod({ n: Number(f.periodN.trim()), unit: f.periodUnit }),
    decimals: f.decimals === '' ? null : Number(f.decimals),
    notice: orNull(f.notice),
    related: f.related,
  }
}

/** 지표 수정 요청 본문(5.6). */
export function editBody(f: SettingsForm): EditRequest {
  return settingsBody(f)
}

/** 지표 추가 요청 본문(5.5). 그 제공처에 없는 코드 칸은 비운다. */
export function addBody(f: AddForm): AddRequest {
  const shown = new Set(sourceFields(f.source).map((x) => x.field))
  const code = (field: Exclude<CodeField, 'classes'>) => (shown.has(field) ? orNull(f[field]) : null)
  return {
    id: f.id.trim(),
    source: f.source,
    stat_code: f.stat_code.trim(),
    item_code: code('item_code'),
    classes: shown.has('classes') ? parseClasses(f.classes) : [],
    org_id: code('org_id'),
    value_field: code('value_field'),
    visible: f.visible,
    ...settingsBody(f),
  }
}

/** 수집 시각 검사(40.api 5.8). HH:MM(00:00~23:59)만 받는다. 맞으면 null이다. */
export function collectTimeError(text: string): string | null {
  return /^([01]\d|2[0-3]):[0-5]\d$/.test(text) ? null : 'HH:MM(00:00~23:59)으로 넣어 주세요'
}
