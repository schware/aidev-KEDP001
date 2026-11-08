import { describe, expect, it } from 'vitest'
import { formatChange, formatCycle, formatDateTime, formatPeriod, formatValue, formatValueUnit } from './format.ts'

describe('formatPeriod', () => {
  it('주기에 맞춰 보여준다', () => {
    expect(formatPeriod('2026-10-29', 'D')).toBe('2026-10-29')
    expect(formatPeriod('2026-09-01', 'M')).toBe('2026-09')
    expect(formatPeriod('2025-01-01', 'A')).toBe('2025')
  })

  it('분기는 시작 달로 정한다', () => {
    expect(formatPeriod('2025-01-01', 'Q')).toBe('2025 Q1')
    expect(formatPeriod('2025-04-01', 'Q')).toBe('2025 Q2')
    expect(formatPeriod('2025-07-01', 'Q')).toBe('2025 Q3')
    expect(formatPeriod('2025-10-01', 'Q')).toBe('2025 Q4')
  })

  it('형식이 다르면 그대로다', () => {
    expect(formatPeriod('202609', 'M')).toBe('202609')
    expect(formatPeriod('', 'A')).toBe('')
  })
})

describe('formatValue', () => {
  it('천 단위 쉼표', () => {
    expect(formatValue(51817499)).toBe('51,817,499')
    expect(formatValue(94465)).toBe('94,465')
  })

  it('소수는 받은 만큼, 4자리까지', () => {
    expect(formatValue(2.5)).toBe('2.5')
    expect(formatValue(3)).toBe('3')
    expect(formatValue(86756.18)).toBe('86,756.18')
    expect(formatValue(0.1 + 0.2)).toBe('0.3')
  })

  it('음수와 0', () => {
    expect(formatValue(-0.39)).toBe('-0.39')
    expect(formatValue(0)).toBe('0')
  })
})

describe('formatValue 소수 자릿수', () => {
  it('decimals가 숫자이면 그 자릿수로 맞춘다', () => {
    expect(formatValue(2.5, 2)).toBe('2.50')
    expect(formatValue(3, 2)).toBe('3.00')
    expect(formatValue(86756.186, 2)).toBe('86,756.19')
    expect(formatValue(1.23456, 4)).toBe('1.2346')
    expect(formatValue(-0.39, 1)).toBe('-0.4')
  })

  it('0이면 정수로, 쉼표는 그대로', () => {
    expect(formatValue(51817499.6, 0)).toBe('51,817,500')
    expect(formatValue(2.5, 0)).toBe('3')
  })

  it('null이면 받은 값 그대로', () => {
    expect(formatValue(2.5, null)).toBe('2.5')
    expect(formatValue(51817499, null)).toBe('51,817,499')
  })

  it('범위 밖이면 받은 값 그대로', () => {
    expect(formatValue(2.5, 7)).toBe('2.5')
  })
})

describe('formatValueUnit', () => {
  it('단위가 있으면 붙인다', () => {
    expect(formatValueUnit(2.5, '연%')).toBe('2.5 연%')
    expect(formatValueUnit(2.5, '연%', 2)).toBe('2.50 연%')
  })

  it('단위가 비어 있으면 숫자만', () => {
    expect(formatValueUnit(1134, null)).toBe('1,134')
    expect(formatValueUnit(1134, '')).toBe('1,134')
  })
})

describe('formatCycle', () => {
  it('주기 이름', () => {
    expect(formatCycle('D')).toBe('일간')
    expect(formatCycle('M')).toBe('월간')
    expect(formatCycle('Q')).toBe('분기')
    expect(formatCycle('A')).toBe('연간')
  })
})

describe('formatDateTime', () => {
  it('Asia/Seoul 기준', () => {
    expect(formatDateTime('2026-10-25T06:00:03+09:00')).toBe('2026-10-25 06:00')
    expect(formatDateTime('2026-10-24T21:00:03Z')).toBe('2026-10-25 06:00')
    expect(formatDateTime('2026-10-30T23:59:59+09:00')).toBe('2026-10-30 23:59')
  })

  it('읽을 수 없으면 그대로', () => {
    expect(formatDateTime('nope')).toBe('nope')
  })
})

describe('formatChange', () => {
  const prev = { period: '2026-08-01', value: 2.3 }

  it('오르면 ▲, 내리면 ▼, 소수 자릿수와 단위를 따른다', () => {
    expect(formatChange('M', prev, 0.2, '연%', 2)).toEqual({ label: '전월 대비', direction: 'up', text: '▲0.20%p' })
    expect(formatChange('M', prev, -0.25, '연%', 2)).toEqual({ label: '전월 대비', direction: 'down', text: '▼0.25%p' })
    expect(formatChange('A', prev, -150000, '명', 0)).toEqual({ label: '전년 대비', direction: 'down', text: '▼150,000 명' })
  })

  it('단위가 %로 끝나면 %p, 그 밖의 단위는 그대로', () => {
    // %
    expect(formatChange('M', prev, -0.1, '%', 1)?.text).toBe('▼0.1%p')
    expect(formatChange('M', prev, 1.25, '%')?.text).toBe('▲1.25%p')
    // 연%
    expect(formatChange('M', prev, 0.25, '연%', 2)?.text).toBe('▲0.25%p')
    // 단위 없음, 빈 단위
    expect(formatChange('D', prev, -0.1, null)?.text).toBe('▼0.1')
    expect(formatChange('D', prev, -0.1, '')?.text).toBe('▼0.1')
    // 다른 단위: %가 끝이 아니면 바꾸지 않는다
    expect(formatChange('M', prev, 12.5, '십억원', 1)?.text).toBe('▲12.5 십억원')
    expect(formatChange('M', prev, -0.4, '2020＝100', 1)?.text).toBe('▼0.4 2020＝100')
    expect(formatChange('M', prev, 3, '%포인트')?.text).toBe('▲3 %포인트')
    // 같으면 단위와 상관없이 "-"
    expect(formatChange('M', prev, 0, '%', 1)?.text).toBe('-')
  })

  it('단위가 없거나 소수 자릿수가 없으면 받은 값 그대로', () => {
    expect(formatChange('D', prev, 0.3, null)).toEqual({ label: '직전 대비', direction: 'up', text: '▲0.3' })
  })

  it('주기마다 다른 이름', () => {
    expect(formatChange('Q', prev, 1, null)?.label).toBe('전기 대비')
    expect(formatChange('D', prev, 1, null)?.label).toBe('직전 대비')
  })

  it('같으면 "-", 소수 자릿수로 0이 되어도 같다', () => {
    expect(formatChange('M', prev, 0, '연%', 2)).toEqual({ label: '전월 대비', direction: 'same', text: '-' })
    expect(formatChange('M', prev, 0.001, '연%', 2)?.direction).toBe('same')
    expect(formatChange('M', prev, -0.001, '연%', 2)?.text).toBe('-')
  })

  it('previous나 change가 null이면 보이지 않는다', () => {
    expect(formatChange('M', null, null, '연%', 2)).toBeNull()
    expect(formatChange('M', prev, null, '연%', 2)).toBeNull()
  })
})
