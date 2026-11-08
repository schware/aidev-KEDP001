import { useEffect, useState } from 'react'
import { formatValue } from '../format.ts'
import { visitsOnce, type VisitSummary } from '../visits.ts'

/**
 * 사용자 화면 아래쪽의 자료 출처와 "오늘 N명 · 전체 N명"(40.api 6장). 처음 열릴 때 방문을 한 번 보내고 숫자를 읽는다.
 * 읽는 중이거나 방문자 API가 실패하면 숫자를 보이지 않는다. 출처와 화면의 다른 부분은 그대로다.
 */
export function VisitFooter() {
  const [summary, setSummary] = useState<VisitSummary | null>(null)

  useEffect(() => {
    let live = true
    visitsOnce().then((s) => {
      if (live) setSummary(s)
    })
    return () => {
      live = false
    }
  }, [])

  return (
    <footer className="border-t border-slate-200 bg-white">
      <div className="mx-auto max-w-5xl space-y-1 px-4 py-5 text-center text-xs text-slate-500">
        <p>자료: 한국은행 ECOS, 통계청 KOSIS, 공공데이터포털</p>
        {summary && (
          <p>
            오늘 <span className="tabular-nums">{formatValue(summary.today, 0)}</span>명 · 전체{' '}
            <span className="tabular-nums">{formatValue(summary.total, 0)}</span>명
          </p>
        )}
      </div>
    </footer>
  )
}
