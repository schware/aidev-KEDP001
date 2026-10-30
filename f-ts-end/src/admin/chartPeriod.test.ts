import { describe, expect, it } from 'vitest'
import { chartPeriodLabel, chartPeriodNumberError, formatChartPeriod, parseChartPeriod } from './chartPeriod.ts'

describe('parseChartPeriod', () => {
  it('API 형식을 숫자와 단위로', () => {
    expect(parseChartPeriod('10 years')).toEqual({ n: 10, unit: 'years' })
    expect(parseChartPeriod('6 months')).toEqual({ n: 6, unit: 'months' })
    expect(parseChartPeriod('30 days')).toEqual({ n: 30, unit: 'days' })
    expect(parseChartPeriod('1 years')).toEqual({ n: 1, unit: 'years' })
  })

  it('PostgreSQL interval 글도 읽는다', () => {
    expect(parseChartPeriod('1 year')).toEqual({ n: 1, unit: 'years' })
    expect(parseChartPeriod('6 mons')).toEqual({ n: 6, unit: 'months' })
    expect(parseChartPeriod('1 day')).toEqual({ n: 1, unit: 'days' })
  })

  it('읽을 수 없으면 null', () => {
    expect(parseChartPeriod(null)).toBeNull()
    expect(parseChartPeriod('')).toBeNull()
    expect(parseChartPeriod('0 years')).toBeNull()
    expect(parseChartPeriod('1 year 6 mons')).toBeNull()
    expect(parseChartPeriod('10 weeks')).toBeNull()
  })
})

describe('formatChartPeriod', () => {
  it('숫자와 단위를 API 형식으로. 1이어도 복수형', () => {
    expect(formatChartPeriod({ n: 10, unit: 'years' })).toBe('10 years')
    expect(formatChartPeriod({ n: 1, unit: 'years' })).toBe('1 years')
    expect(formatChartPeriod({ n: 18, unit: 'months' })).toBe('18 months')
    expect(formatChartPeriod({ n: 90, unit: 'days' })).toBe('90 days')
  })

  it('읽고 다시 만들면 같다', () => {
    for (const s of ['10 years', '1 years', '6 months', '9999 days']) {
      expect(formatChartPeriod(parseChartPeriod(s)!)).toBe(s)
    }
  })
})

describe('chartPeriodLabel', () => {
  it('표에 보일 글', () => {
    expect(chartPeriodLabel('10 years')).toBe('10년')
    expect(chartPeriodLabel('6 months')).toBe('6개월')
    expect(chartPeriodLabel('30 days')).toBe('30일')
    expect(chartPeriodLabel(null)).toBe('-')
    expect(chartPeriodLabel('1 year 6 mons')).toBe('1 year 6 mons')
  })
})

describe('chartPeriodNumberError', () => {
  it('1~9999 정수만', () => {
    expect(chartPeriodNumberError('1')).toBeNull()
    expect(chartPeriodNumberError('9999')).toBeNull()
    expect(chartPeriodNumberError(' 10 ')).toBeNull()
    expect(chartPeriodNumberError('0')).not.toBeNull()
    expect(chartPeriodNumberError('10000')).not.toBeNull()
    expect(chartPeriodNumberError('1.5')).not.toBeNull()
    expect(chartPeriodNumberError('-1')).not.toBeNull()
    expect(chartPeriodNumberError('')).not.toBeNull()
  })
})
