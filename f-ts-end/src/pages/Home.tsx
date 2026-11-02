import { Link } from 'react-router'
import { cardsPath, type Card, type CardList } from '../api.ts'
import { groupByCategory } from '../category.ts'
import { Message } from '../components/Layout.tsx'
import { formatPeriod, formatValue } from '../format.ts'
import { useApi } from '../useApi.ts'

/**
 * 메인 화면: 노출된 지표 카드를 구분별로 묶어 보여준다(50.web 2장).
 * 구분 이름 아래에 그 구분의 카드를 놓는다. 순서는 API가 준 순서(구분 순서 → 노출 순서) 그대로다.
 */
export function Home() {
  const state = useApi<CardList>(cardsPath)

  return (
    <>
      <title>KEDP</title>
      <h1 className="mb-6 text-center text-lg font-semibold sm:text-xl">대한민국 주요 경제지표</h1>
      {state.status === 'loading' && <Message>불러오는 중…</Message>}
      {(state.status === 'error' || state.status === 'not-found') && (
        <Message>데이터를 불러오지 못했습니다. 잠시 뒤 다시 시도해 주세요.</Message>
      )}
      {state.status === 'ok' &&
        (state.data.indicators.length === 0 ? (
          <Message>보여줄 지표가 없습니다.</Message>
        ) : (
          <div className="space-y-8">
            {groupByCategory(state.data.indicators).map(({ category, cards }) => (
              <section key={category.code} aria-labelledby={`category-${category.code}`}>
                <h2 id={`category-${category.code}`} className="mb-3 border-b border-slate-200 pb-1 font-semibold text-slate-700">
                  {category.name}
                </h2>
                <ul className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
                  {cards.map((c) => (
                    <li key={c.id}>
                      <IndicatorCard card={c} />
                    </li>
                  ))}
                </ul>
              </section>
            ))}
          </div>
        ))}
    </>
  )
}

function IndicatorCard({ card }: { card: Card }) {
  return (
    <Link
      to={`/indicator/${encodeURIComponent(card.id)}`}
      className="flex h-full flex-col rounded-lg border border-slate-200 bg-white p-5 shadow-sm transition hover:border-blue-400 hover:shadow focus-visible:outline-2 focus-visible:outline-blue-500"
    >
      <h3 className="font-medium text-slate-700">{card.name}</h3>
      {card.latest ? (
        <>
          <p className="mt-2 break-all">
            <span className="text-2xl font-bold tabular-nums">{formatValue(card.latest.value, card.decimals)}</span>
            {card.unit && <span className="ml-1 text-sm text-slate-500">{card.unit}</span>}
          </p>
          <p className="mt-1 text-xs text-slate-500">{formatPeriod(card.latest.period, card.cycle)}</p>
        </>
      ) : (
        <p className="mt-2 text-slate-400">아직 데이터가 없습니다</p>
      )}
      <span className="mt-auto pt-4 text-sm text-blue-600">상세보기 &gt;</span>
    </Link>
  )
}
