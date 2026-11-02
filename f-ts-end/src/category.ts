// 메인 화면의 구분별 묶기(50.web 2장). React와 상관없이 시험할 수 있게 따로 둔다.
import type { Card, Category } from './api.ts'

export interface CategoryGroup {
  category: Category
  cards: Card[]
}

/**
 * 카드를 구분별로 묶는다. API가 준 순서(구분 순서 → 노출 순서)를 그대로 쓰고 다시 정렬하지 않는다.
 * 구분은 처음 나온 순서대로, 카드는 받은 순서대로 둔다. 카드가 없는 구분은 생기지 않는다.
 */
export function groupByCategory(cards: Card[]): CategoryGroup[] {
  const groups = new Map<string, CategoryGroup>()
  for (const card of cards) {
    const g = groups.get(card.category.code)
    if (g) {
      g.cards.push(card)
    } else {
      groups.set(card.category.code, { category: card.category, cards: [card] })
    }
  }
  return [...groups.values()]
}
