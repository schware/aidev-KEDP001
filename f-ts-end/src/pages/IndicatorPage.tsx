import type { ReactNode } from 'react'
import { Link, useLocation, useNavigate, useParams } from 'react-router'
import { detailPath, type Detail } from '../api.ts'
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
      {d.notice && (
        <p
          role="alert"
          className="rounded-lg border-2 border-red-500 bg-red-50 px-4 py-3 text-base font-semibold text-red-700"
        >
          {d.notice}
        </p>
      )}

      <h1 className="text-2xl font-bold">{d.name}</h1>

      <Section title="현재값">
        {d.latest ? (
          <>
            <p className="break-all">
              <span className="text-3xl font-bold tabular-nums">{formatValue(d.latest.value, d.decimals)}</span>
              {d.unit && <span className="ml-1 text-slate-500">{d.unit}</span>}
            </p>
            <p className="mt-1 text-sm text-slate-500">{formatPeriod(d.latest.period, d.cycle)}</p>
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

      <Section title="최근 데이터">
        {d.recent.length > 0 ? (
          <table className="w-full max-w-sm text-sm">
            <thead className="sr-only">
              <tr>
                <th>기간</th>
                <th>값</th>
              </tr>
            </thead>
            <tbody>
              {d.recent.map((p) => (
                <tr key={p.period} className="border-b border-slate-100 last:border-0">
                  <td className="py-1.5 pr-4 text-slate-600">{formatPeriod(p.period, d.cycle)}</td>
                  <td className="py-1.5 text-right tabular-nums">{formatValue(p.value, d.decimals)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <p className="text-slate-400">데이터가 없습니다</p>
        )}
      </Section>

      <Section title="지표 정보">
        <dl className="grid grid-cols-[auto_1fr] gap-x-6 gap-y-1.5 text-sm">
          <dt className="text-slate-500">출처</dt>
          <dd>{d.source.name}</dd>
          <dt className="text-slate-500">통계 코드</dt>
          <dd className="break-all">{d.stat_code}</dd>
          <dt className="text-slate-500">주기</dt>
          <dd>{formatCycle(d.cycle)}</dd>
          <dt className="text-slate-500">마지막 수집</dt>
          <dd>{d.last_collected_at ? formatDateTime(d.last_collected_at) : '없음'}</dd>
        </dl>
      </Section>

      {d.related.length > 0 && (
        <Section title="관련 지표">
          <ul className="space-y-1.5">
            {d.related.map((r) => (
              <li key={r.id}>
                <Link to={`/indicator/${encodeURIComponent(r.id)}`} className="text-blue-600 hover:underline">
                  {r.name}
                </Link>
              </li>
            ))}
          </ul>
        </Section>
      )}

      <BackButton />
    </article>
  )
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="rounded-lg border border-slate-200 bg-white p-4 sm:p-5">
      <h2 className="mb-3 text-sm font-semibold text-slate-500">{title}</h2>
      {children}
    </section>
  )
}

/** 이 사이트 안에서 들어왔으면 이전 화면으로, 주소로 바로 들어왔으면 메인 화면으로 간다. */
function BackButton() {
  const navigate = useNavigate()
  const location = useLocation()
  const fromApp = location.key !== 'default'
  return (
    <button
      type="button"
      onClick={() => (fromApp ? navigate(-1) : navigate('/'))}
      className="rounded-md border border-slate-300 bg-white px-4 py-2 text-sm hover:bg-slate-100"
    >
      뒤로가기
    </button>
  )
}
