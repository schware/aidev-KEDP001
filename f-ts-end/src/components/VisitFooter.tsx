import { useEffect, useState } from 'react'
import { formatValue } from '../format.ts'
import { visitsOnce, type VisitSummary } from '../visits.ts'

/**
 * 사용자 화면 아래쪽의 "오늘 N명 · 전체 N명"(40.api 6장). 처음 열릴 때 방문을 한 번 보내고 숫자를 읽는다.
 * 읽는 중이거나 방문자 API가 실패하면 숫자를 보이지 않는다. 화면의 다른 부분은 그대로다.
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
    <footer className="border-t border-slate-200">
      <div className="mx-auto min-h-12 max-w-5xl px-4 py-4 text-center text-xs text-slate-500">
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
