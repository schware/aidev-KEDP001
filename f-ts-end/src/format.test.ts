import { describe, expect, it } from 'vitest'
import { formatCycle, formatDateTime, formatPeriod, formatValue, formatValueUnit } from './format.ts'

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

describe('formatValueUnit', () => {
  it('단위가 있으면 붙인다', () => {
    expect(formatValueUnit(2.5, '연%')).toBe('2.5 연%')
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
