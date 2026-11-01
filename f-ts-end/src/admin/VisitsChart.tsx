import { useEffect, useRef } from 'react'
import * as echarts from 'echarts/core'
import { BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import { formatValue } from '../format.ts'
import type { VisitDay } from './api.ts'

// 쓰는 것만 넣는다. 다운로드(toolbox saveAsImage)는 넣지 않는다(KEDP001-D003).
echarts.use([BarChart, GridComponent, TooltipComponent, CanvasRenderer])

/** 최근 30일 일별 방문자 수 막대 차트. days는 오래된 날부터다(40.api 6.4). */
export function VisitsChart({ days }: { days: VisitDay[] }) {
  const el = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!el.current) {
      return
    }
    const chart = echarts.init(el.current)
    chart.setOption({
      animation: false,
      grid: { left: 8, right: 16, top: 16, bottom: 8, containLabel: true },
      tooltip: {
        trigger: 'axis',
        axisPointer: { type: 'shadow' },
        valueFormatter: (v: unknown) => (typeof v === 'number' ? `${formatValue(v, 0)}명` : String(v)),
      },
      xAxis: {
        type: 'category',
        // 축에는 MM-DD만, tooltip에는 날짜 전체를 보인다.
        data: days.map((d) => d.day),
        axisLabel: { formatter: (v: string) => v.slice(5) },
      },
      yAxis: {
        type: 'value',
        minInterval: 1,
        axisLabel: { formatter: (v: number) => formatValue(v, 0) },
      },
      series: [{ type: 'bar', name: '방문자', data: days.map((d) => d.count), color: '#2563eb' }],
    })
    const observer = new ResizeObserver(() => chart.resize())
    observer.observe(el.current)
    return () => {
      observer.disconnect()
      chart.dispose()
    }
  }, [days])

  return <div ref={el} className="h-64 w-full sm:h-72" role="img" aria-label="최근 30일 일별 방문자 수 막대 차트" />
}
