import { describe, expect, it } from 'vitest'
import type { AdminIndicator } from './api.ts'
import {
  addBody,
  collectTimeError,
  editBody,
  emptyAddForm,
  parseClasses,
  settingsFromIndicator,
  sourceFields,
  validateAdd,
  validateEdit,
  type AddForm,
} from './form.ts'

const base: AddForm = { ...emptyAddForm(), id: 'new-one', name: '새 지표' }
const fieldsOf = (s: AddForm['source']) => sourceFields(s).map((x) => x.field)

describe('sourceFields', () => {
  it('제공처마다 필요한 칸만', () => {
    expect(fieldsOf('ecos')).toEqual(['stat_code', 'item_code'])
    expect(fieldsOf('kosis')).toEqual(['org_id', 'stat_code', 'item_code', 'classes'])
    expect(fieldsOf('data-go-kr')).toEqual(['stat_code', 'value_field'])
  })
})

describe('validateAdd 제공처별 칸', () => {
  it('ECOS: 통계 코드, 항목 코드가 꼭 있어야 한다', () => {
    const e = validateAdd({ ...base, source: 'ecos' }, [])
    expect(Object.keys(e).sort()).toEqual(['item_code', 'stat_code'])
    expect(validateAdd({ ...base, source: 'ecos', stat_code: 'S1', item_code: 'I1' }, [])).toEqual({})
  })

  it('ECOS는 KOSIS 칸(org_id)을 검사하지 않는다', () => {
    expect(validateAdd({ ...base, source: 'ecos', stat_code: 'S1', item_code: 'I1', org_id: '' }, [])).toEqual({})
  })

  it('KOSIS: org_id, 통계 코드, 항목 코드가 꼭 있고, 분류는 빈 코드가 없어야 한다', () => {
    const e = validateAdd({ ...base, source: 'kosis', classes: '00,,01' }, [])
    expect(Object.keys(e).sort()).toEqual(['classes', 'item_code', 'org_id', 'stat_code'])
    expect(
      validateAdd({ ...base, source: 'kosis', org_id: 'O', stat_code: 'S', item_code: 'I', classes: '' }, []),
    ).toEqual({})
  })

  it('공공데이터포털: 종목 코드와 value_field. 항목 코드는 검사하지 않는다', () => {
    expect(emptyAddForm().value_field).toBe('clpr')
    const e = validateAdd({ ...base, source: 'data-go-kr', value_field: ' ' }, [])
    expect(Object.keys(e).sort()).toEqual(['stat_code', 'value_field'])
    expect(validateAdd({ ...base, source: 'data-go-kr', stat_code: '000000' }, [])).toEqual({})
  })
})

describe('validateAdd 공통 칸', () => {
  const ok: AddForm = { ...base, stat_code: 'S', item_code: 'I' }

  it('id 형식과 중복', () => {
    expect(validateAdd({ ...ok, id: 'Bad_ID' }, []).id).toBeDefined()
    expect(validateAdd({ ...ok, id: 'a--b' }, []).id).toBeDefined()
    expect(validateAdd({ ...ok, id: 'kosis-kospi' }, ['kosis-kospi']).id).toBe('같은 id가 이미 있습니다')
    expect(validateAdd({ ...ok, id: 'kosis-kospi2' }, ['kosis-kospi'])).toEqual({})
  })

  it('이름, 차트 기간, 소수 자릿수', () => {
    const e = validateAdd({ ...ok, name: '  ', periodN: '0', decimals: '5' }, [])
    expect(Object.keys(e).sort()).toEqual(['chart_period', 'decimals', 'name'])
  })

  it('관련 지표에 자기 자신과 중복은 안 된다', () => {
    expect(validateAdd({ ...ok, related: ['new-one'] }, []).related).toBeDefined()
    expect(validateAdd({ ...ok, related: ['a', 'a'] }, []).related).toBeDefined()
  })
})

describe('addBody', () => {
  it('그 제공처에 없는 코드 칸은 비워서 보낸다', () => {
    const f: AddForm = {
      ...base,
      source: 'data-go-kr',
      stat_code: ' 069500 ',
      item_code: '남은 값',
      org_id: '남은 값',
      classes: '00',
      unit: '',
      periodN: '1',
      periodUnit: 'years',
      decimals: '',
      notice: ' ',
    }
    expect(addBody(f)).toEqual({
      id: 'new-one',
      source: 'data-go-kr',
      stat_code: '069500',
      item_code: null,
      classes: [],
      org_id: null,
      value_field: 'clpr',
      visible: false,
      name: '새 지표',
      cycle: 'M',
      unit: null,
      chart_period: '1 years',
      decimals: null,
      notice: null,
      related: [],
    })
  })

  it('KOSIS는 분류를 배열로', () => {
    const b = addBody({ ...base, source: 'kosis', org_id: '101', stat_code: 'T', item_code: 'I', classes: '00, 01' })
    expect(b.classes).toEqual(['00', '01'])
    expect(b.org_id).toBe('101')
    expect(b.value_field).toBeNull()
  })
})

describe('편집', () => {
  const ind: AdminIndicator = {
    id: 'ecos-base-rate',
    name: '한국은행 기준금리',
    source: 'ecos',
    stat_code: '722Y001',
    item_code: '0101000',
    classes: [],
    org_id: null,
    value_field: null,
    cycle: 'M',
    unit: null,
    source_unit: '연%',
    visible: true,
    order: 1,
    chart_period: '10 years',
    decimals: 2,
    notice: null,
    related: ['kosis-employment-rate'],
  }

  it('지표에서 칸을 만들고 그대로 보내면 같은 값이다(40.api 5.6 예시)', () => {
    const f = settingsFromIndicator(ind)
    expect(validateEdit(f, ind.id)).toEqual({})
    expect(editBody(f)).toEqual({
      name: '한국은행 기준금리',
      cycle: 'M',
      unit: null,
      chart_period: '10 years',
      decimals: 2,
      notice: null,
      related: ['kosis-employment-rate'],
    })
  })

  it('소수 자릿수 0도 보낸다', () => {
    expect(editBody({ ...settingsFromIndicator(ind), decimals: '0' }).decimals).toBe(0)
  })

  it('차트 기간이 없는 지표는 넣어야 저장된다', () => {
    const f = settingsFromIndicator({ ...ind, chart_period: null, order: null })
    expect(f.periodN).toBe('')
    expect(validateEdit(f, ind.id).chart_period).toBeDefined()
  })

  it('자기 자신은 관련 지표가 될 수 없다', () => {
    expect(validateEdit({ ...settingsFromIndicator(ind), related: [ind.id] }, ind.id).related).toBeDefined()
  })
})

describe('parseClasses', () => {
  it('쉼표로 나눈다', () => {
    expect(parseClasses('')).toEqual([])
    expect(parseClasses(' 00 , 01 ')).toEqual(['00', '01'])
    expect(parseClasses('00,')).toEqual(['00', ''])
  })
})

describe('collectTimeError', () => {
  it('HH:MM만 받는다', () => {
    expect(collectTimeError('06:00')).toBeNull()
    expect(collectTimeError('00:00')).toBeNull()
    expect(collectTimeError('23:59')).toBeNull()
    expect(collectTimeError('6:00')).not.toBeNull()
    expect(collectTimeError('24:00')).not.toBeNull()
    expect(collectTimeError('07:60')).not.toBeNull()
    expect(collectTimeError('0730')).not.toBeNull()
  })
})
