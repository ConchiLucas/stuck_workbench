import ReactECharts from 'echarts-for-react'
import { useTrend } from '../../api/dashboard'
import { useChildStore } from '../../store/childStore'

const MUTE = '#c9a8b8'
const INK = '#f6eaf1'

export function TrendChart({ days = 30 }: { days?: number }) {
  const childId = useChildStore((s) => s.childId)
  const { data } = useTrend(childId, days)
  if (!data) return null

  const option = {
    backgroundColor: 'transparent',
    grid: { left: 40, right: 40, top: 30, bottom: 30 },
    tooltip: { trigger: 'axis' },
    legend: { data: ['练习时长(分钟)', '累计已掌握'], top: 0, textStyle: { fontSize: 11, color: MUTE } },
    xAxis: {
      type: 'category',
      data: data.map((p) => p.date.slice(5)),
      axisLabel: { fontSize: 10, interval: Math.floor(days / 10), color: MUTE },
      axisLine: { lineStyle: { color: 'rgba(255,255,255,0.12)' } },
    },
    yAxis: [
      { type: 'value', name: '分钟', axisLabel: { fontSize: 10, color: MUTE }, nameTextStyle: { color: MUTE }, splitLine: { lineStyle: { color: 'rgba(255,255,255,0.06)' } } },
      { type: 'value', name: '已掌握', axisLabel: { fontSize: 10, color: MUTE }, nameTextStyle: { color: MUTE }, splitLine: { show: false } },
    ],
    series: [
      {
        name: '练习时长(分钟)', type: 'bar', yAxisIndex: 0,
        data: data.map((p) => p.practice_min),
        itemStyle: { color: '#FBA9C5', borderRadius: [4, 4, 0, 0] },
      },
      {
        name: '累计已掌握', type: 'line', yAxisIndex: 1, smooth: true, showSymbol: false,
        data: data.map((p) => p.cumulative_mastered),
        lineStyle: { color: '#4ade80', width: 3 },
        areaStyle: { color: 'rgba(74,222,128,0.12)' },
      },
    ],
    textStyle: { color: INK },
  }

  return (
    <div className="dash-card p-4">
      <div className="mb-2 font-semibold text-white">近 {days} 天学习趋势</div>
      <ReactECharts option={option} style={{ height: 260 }} />
    </div>
  )
}
