import { describe, expect, it } from 'vitest'
import type { Card } from './api.ts'
import { groupByCategory } from './category.ts'

const money = { code: 'money', name: '금리·통화' }
const labor = { code: 'labor', name: '고용·소득' }
const market = { code: 'market', name: '금융시장' }

function card(id: string, category: Card['category']): Card {
  return { id, name: id, category, cycle: 'M', unit: null, decimals: null, latest: null }
}

const ids = (g: ReturnType<typeof groupByCategory>) => g.map((x) => [x.category.name, x.cards.map((c) => c.id)])

describe('groupByCategory', () => {
  it('받은 순서대로 구분을 묶는다(다시 정렬하지 않는다)', () => {
    const cards = [card('b', labor), card('a', labor), card('z', money), card('k1', market), card('k2', market)]
    expect(ids(groupByCategory(cards))).toEqual([
      ['고용·소득', ['b', 'a']],
      ['금리·통화', ['z']],
      ['금융시장', ['k1', 'k2']],
    ])
  })

  it('카드가 없으면 구분도 없다', () => {
    expect(groupByCategory([])).toEqual([])
  })

  it('같은 구분이 떨어져 와도 처음 나온 자리에 하나로 묶는다', () => {
    const cards = [card('a', money), card('b', market), card('c', money)]
    expect(ids(groupByCategory(cards))).toEqual([
      ['금리·통화', ['a', 'c']],
      ['금융시장', ['b']],
    ])
  })
})
