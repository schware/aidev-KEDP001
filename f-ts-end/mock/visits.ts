// npm run dev:mock에서만 쓰는 가짜 방문자 수(specification/40.api/README.md 6장). 같은 사람을 거르지 않는다:
// POST /api/visits가 올 때마다 오늘 숫자를 1 늘린다(화면의 localStorage 판단을 보려면 그것으로 충분하다).
// dev 서버를 다시 켜면 처음으로 돌아간다.
import type { AdminVisits } from '../src/admin/api.ts'

let base = 3000
const days = new Map<string, number>()

function seoulDay(offset = 0): string {
  return new Date(Date.now() + 9 * 3600_000 - offset * 86400_000).toISOString().slice(0, 10)
}

// 지난 날 숫자를 조금 넣어 둔다(차트가 비지 않게).
for (let i = 1; i < 30; i++) {
  if (i % 4 !== 0) days.set(seoulDay(i), (i * 37) % 90)
}

export function addVisit(): void {
  const d = seoulDay()
  days.set(d, (days.get(d) ?? 0) + 1)
}

export function visitSummary() {
  let total = base
  for (const n of days.values()) total += n
  return { today: days.get(seoulDay()) ?? 0, total }
}

export function adminVisits(): AdminVisits {
  const list = Array.from({ length: 30 }, (_, i) => {
    const day = seoulDay(29 - i)
    return { day, count: days.get(day) ?? 0 }
  })
  return { ...visitSummary(), base, days: list }
}

export function setVisitBase(v: number): void {
  base = v
}

export function resetVisits(): void {
  days.clear()
  base = 0
}
