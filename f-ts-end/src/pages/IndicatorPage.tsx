import type { ReactNode } from 'react'
import { Link, useLocation, useNavigate, useParams } from 'react-router'
import { detailPath, type Detail } from '../api.ts'
import { ChangeLine } from '../components/ChangeLine.tsx'
import { Chart } from '../components/Chart.tsx'
import { Message, NotFound } from '../components/Layout.tsx'
import { formatCycle, formatDateTime, formatPeriod, formatValue } from '../format.ts'
import { useApi } from '../useApi.ts'

/** 지표 상세 화면(50.web 3장). */
export function IndicatorPage() {
  const { id = '' } = useParams()
  const state = useApi<Detail>(detailPath(id))

  if (state.status === 'loading') {
    return <Message>불러오는 중…</Message>
  }
  if (state.status === 'not-found') {
    return <NotFound what="지표" />
  }
  if (state.status === 'error') {
    return <Message>데이터를 불러오지 못했습니다. 잠시 뒤 다시 시도해 주세요.</Message>
  }
  return <DetailView d={state.data} />
}

function DetailView({ d }: { d: Detail }) {
  return (
    <article className="space-y-6">
      <title>{`${d.name} - KEDP`}</title>
      <BackButton top />
      {d.notice && (
        <p
          role="alert"
          className="rounded-lg border-2 border-red-500 bg-red-50 px-4 py-3 text-base font-semibold text-red-700"
        >
          {d.notice}
        </p>
      )}

      <header>
        <p className="text-sm text-slate-500">{d.category.name}</p>
        <h1 className="mt-1 text-2xl font-bold sm:text-3xl">{d.name}</h1>
      </header>

      <Section title="현재값">
        {d.latest ? (
          <>
            <p className="break-all">
              <span className="text-4xl font-bold tracking-tight tabular-nums">
                {formatValue(d.latest.value, d.decimals)}
              </span>
              {d.unit && <span className="ml-1 text-lg text-slate-500">{d.unit}</span>}
            </p>
            <p className="mt-1 text-sm text-slate-500">{formatPeriod(d.latest.period, d.cycle)} 기준</p>
            <ChangeLine
              cycle={d.cycle}
              previous={d.previous}
              change={d.change}
              unit={d.unit}
              decimals={d.decimals}
              className="mt-2 text-base"
            />
          </>
        ) : (
          <p className="text-slate-400">아직 데이터가 없습니다</p>
        )}
      </Section>

      <Section title="추이">
        {d.chart.length > 0 ? (
          <Chart points={d.chart} cycle={d.cycle} unit={d.unit} decimals={d.decimals} />
        ) : (
          <p className="text-slate-400">차트에 보여줄 데이터가 없습니다</p>
        )}
      </Section>

      <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
        <Section title="최근 데이터">
          {d.recent.length > 0 ? (
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-slate-200 text-xs text-slate-500">
                  <th scope="col" className="pb-2 text-left font-medium">
                    기간
                  </th>
                  <th scope="col" className="pb-2 text-right font-medium">
                    값{d.unit && ` (${d.unit})`}
                  </th>
                </tr>
              </thead>
              <tbody>
                {d.recent.map((p) => (
                  <tr key={p.period} className="border-b border-slate-100 last:border-0 hover:bg-slate-50">
                    <td className="py-2 pr-4 text-slate-600">{formatPeriod(p.period, d.cycle)}</td>
                    <td className="py-2 text-right font-medium tabular-nums">{formatValue(p.value, d.decimals)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <p className="text-slate-400">데이터가 없습니다</p>
          )}
        </Section>

        <div className="space-y-6">
          <Section title="지표 정보">
            <dl className="divide-y divide-slate-100 text-sm">
              <Info term="구분">{d.category.name}</Info>
              <Info term="출처">{d.source.name}</Info>
              <Info term="통계 코드">
                <span className="break-all">{d.stat_code}</span>
              </Info>
              <Info term="주기">{formatCycle(d.cycle)}</Info>
              <Info term="마지막 수집">{d.last_collected_at ? formatDateTime(d.last_collected_at) : '없음'}</Info>
            </dl>
          </Section>

          {d.related.length > 0 && (
            <Section title="관련 지표">
              <ul className="space-y-1.5">
                {d.related.map((r) => (
                  <li key={r.id}>
                    <Link to={`/indicator/${encodeURIComponent(r.id)}`} className="text-blue-600 hover:underline">
                      {r.name} &gt;
                    </Link>
                  </li>
                ))}
              </ul>
            </Section>
          )}
        </div>
      </div>

      <BackButton />
    </article>
  )
}

function Info({ term, children }: { term: string; children: ReactNode }) {
  return (
    <div className="flex justify-between gap-4 py-2">
      <dt className="shrink-0 text-slate-500">{term}</dt>
      <dd className="text-right">{children}</dd>
    </div>
  )
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm sm:p-5">
      <h2 className="mb-3 text-sm font-semibold text-slate-500">{title}</h2>
      {children}
    </section>
  )
}

/**
 * 이 사이트 안에서 들어왔으면 이전 화면으로, 주소로 바로 들어왔으면 메인 화면으로 간다.
 * top이면 화면 위쪽의 작은 글 단추다.
 */
function BackButton({ top = false }: { top?: boolean }) {
  const navigate = useNavigate()
  const location = useLocation()
  const fromApp = location.key !== 'default'
  return (
    <button
      type="button"
      onClick={() => (fromApp ? navigate(-1) : navigate('/'))}
      className={
        top
          ? 'inline-flex items-center text-sm text-slate-600 hover:text-blue-700 hover:underline'
          : 'rounded-md border border-slate-300 bg-white px-4 py-2 text-sm hover:bg-slate-100'
      }
    >
      {top ? '← 뒤로가기' : '뒤로가기'}
    </button>
  )
}
