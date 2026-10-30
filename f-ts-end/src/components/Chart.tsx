import { useEffect, useRef } from 'react'
import * as echarts from 'echarts/core'
import { LineChart } from 'echarts/charts'
import { DataZoomComponent, GridComponent, TooltipComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import type { Cycle, Point } from '../api.ts'
import { formatPeriod, formatValue, formatValueUnit } from '../format.ts'

// 쓰는 것만 넣는다. 다운로드(toolbox saveAsImage)는 넣지 않는다(KEDP001-D003).
echarts.use([LineChart, GridComponent, TooltipComponent, DataZoomComponent, CanvasRenderer])

interface Props {
  points: Point[]
  cycle: Cycle
  unit: string | null
}

/** 시계열 선 차트. points는 오래된 것부터다(40.api 3장 chart). */
export function Chart({ points, cycle, unit }: Props) {
  const el = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!el.current) {
      return
    }
    const chart = echarts.init(el.current)
    chart.setOption({
      animation: false,
      grid: { left: 8, right: 16, top: 16, bottom: 48, containLabel: true },
      tooltip: {
        trigger: 'axis',
        valueFormatter: (v: unknown) => (typeof v === 'number' ? formatValueUnit(v, unit) : String(v)),
      },
      xAxis: {
        type: 'category',
        boundaryGap: false,
        data: points.map((p) => formatPeriod(p.period, cycle)),
      },
      yAxis: {
        type: 'value',
        scale: true,
        axisLabel: { formatter: (v: number) => formatValue(v) },
      },
      dataZoom: [{ type: 'inside' }, { type: 'slider', height: 20, bottom: 8 }],
      series: [
        {
          type: 'line',
          data: points.map((p) => p.value),
          showSymbol: points.length <= 40,
          lineStyle: { width: 2 },
          color: '#2563eb',
        },
      ],
    })
    const observer = new ResizeObserver(() => chart.resize())
    observer.observe(el.current)
    return () => {
      observer.disconnect()
      chart.dispose()
    }
  }, [points, cycle, unit])

  return <div ref={el} className="h-72 w-full sm:h-80" role="img" aria-label="시계열 차트" />
}
