import { Link } from 'react-router'
import { cardsPath, type Card, type CardList } from '../api.ts'
import { groupByCategory } from '../category.ts'
import { ChangeLine } from '../components/ChangeLine.tsx'
import { Message } from '../components/Layout.tsx'
import { formatPeriod, formatValue } from '../format.ts'
import { useApi } from '../useApi.ts'

/**
 * 메인 화면: 노출된 지표 카드를 구분별로 묶어 보여준다(50.web 2장).
 * 구분 이름 아래에 그 구분의 카드를 놓는다. 순서는 API가 준 순서(구분 순서 → 노출 순서) 그대로다.
 * 휴대폰 1열, 태블릿 2열, PC 3열이다. 카드가 적은 구분은 옆 구분과 한 줄에 놓아 오른쪽이 비지 않게 한다.
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
          <div className="grid grid-cols-1 gap-x-4 gap-y-8 sm:grid-cols-2 lg:grid-cols-3">
            {groupByCategory(state.data.indicators).map(({ category, cards }) => {
              const layout = sectionLayout(cards.length)
              return (
                <section
                  key={category.code}
                  aria-labelledby={`category-${category.code}`}
                  className={`flex flex-col ${layout.section}`}
                >
                  <h2
                    id={`category-${category.code}`}
                    className="mb-3 border-b border-slate-200 pb-1 font-semibold text-slate-700"
                  >
                    {category.name}
                  </h2>
                  <ul className={`grid flex-1 auto-rows-fr grid-cols-1 gap-4 ${layout.list}`}>
                    {cards.map((c) => (
                      <li key={c.id}>
                        <IndicatorCard card={c} />
                      </li>
                    ))}
                  </ul>
                </section>
              )
            })}
          </div>
        ))}
    </>
  )
}

/**
 * 구분 하나가 차지하는 칸 수와 그 안의 열 수. 카드 수만큼 칸을 쓴다(태블릿 2칸, PC 3칸까지).
 * Tailwind가 class 이름을 찾을 수 있게 글 전체를 적는다.
 */
function sectionLayout(count: number): { section: string; list: string } {
  if (count >= 3) {
    return { section: 'sm:col-span-2 lg:col-span-3', list: 'sm:grid-cols-2 lg:grid-cols-3' }
  }
  if (count === 2) {
    return { section: 'sm:col-span-2', list: 'sm:grid-cols-2' }
  }
  return { section: '', list: '' }
}

function IndicatorCard({ card }: { card: Card }) {
  return (
    <Link
      to={`/indicator/${encodeURIComponent(card.id)}`}
      className="group flex h-full min-h-44 flex-col rounded-xl border border-slate-200 bg-white p-5 shadow-sm transition duration-150 hover:-translate-y-0.5 hover:border-blue-500 hover:shadow-md focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-500"
    >
      <h3 className="font-medium text-slate-700 group-hover:text-blue-700">{card.name}</h3>
      {card.latest ? (
        <>
          <p className="mt-3 break-all">
            <span className="text-3xl font-bold tracking-tight tabular-nums">
              {formatValue(card.latest.value, card.decimals)}
            </span>
            {card.unit && <span className="ml-1 text-base text-slate-500">{card.unit}</span>}
          </p>
          <p className="mt-1 text-xs text-slate-500">{formatPeriod(card.latest.period, card.cycle)} 기준</p>
          <ChangeLine
            cycle={card.cycle}
            previous={card.previous}
            change={card.change}
            unit={card.unit}
            decimals={card.decimals}
            className="mt-2"
          />
        </>
      ) : (
        <p className="mt-3 text-slate-400">아직 데이터가 없습니다</p>
      )}
      <span className="mt-auto pt-4 text-sm text-blue-600 group-hover:underline">상세보기 &gt;</span>
    </Link>
  )
}
