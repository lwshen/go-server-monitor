<script setup lang="ts">
// Reusable Chart.js line chart for history series. Uses a CATEGORY x-axis with
// pre-formatted string labels (no date adapter). Reads theme colors from the
// CSS custom properties on <html> so it stays in sync with the active theme.
// Null / -1 samples render as gaps (spanGaps stays off).

import { Chart, type ChartConfiguration, type ChartDataset } from 'chart.js/auto'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { HistoryPoint, HistoryRange } from '@/types'
import { useTheme } from '@/composables/useTheme'

/** A single plotted series: a label, a color token, and a value extractor. */
export interface ChartSeries {
  key: string
  label: string
  /** CSS custom-property name to read the stroke color from (e.g. '--accent'). */
  colorVar: string
  /** Pull the numeric value (or null for a gap) out of a sample. */
  pick: (p: HistoryPoint) => number | null
  /** Optional per-point unit for tooltips (e.g. '%', ' ms', ' MB/s'). */
  unit?: string
  /** Bind to the secondary (right) y-axis instead of the primary. */
  axis?: 'y' | 'y2'
}

const props = withDefaults(
  defineProps<{
    samples: HistoryPoint[]
    series: ChartSeries[]
    range?: HistoryRange
    /** Primary y-axis label. */
    yLabel?: string
    /** Secondary y-axis label (only rendered when a series binds to 'y2'). */
    y2Label?: string
    /** Clamp the primary axis to 0..100 (percentage series). */
    yPercent?: boolean
    height?: number
  }>(),
  { range: '1h', height: 260 },
)

const canvas = ref<HTMLCanvasElement | null>(null)
let chart: Chart<'line'> | null = null

const { theme } = useTheme()

/** -1 and null are unmeasured sentinels -> render as a gap (null). */
function clean(v: number | null): number | null {
  return v === null || v === -1 || Number.isNaN(v) ? null : v
}

/**
 * Format a Unix-second timestamp into a compact category label. Short ranges
 * show HH:MM; multi-day ranges show M/D (with HH:MM for 24h/7d granularity).
 */
function labelFor(ts: number, range: HistoryRange): string {
  const d = new Date(ts * 1000)
  const p = (x: number) => String(x).padStart(2, '0')
  const hm = `${p(d.getHours())}:${p(d.getMinutes())}`
  const md = `${d.getMonth() + 1}/${d.getDate()}`
  switch (range) {
    case '1h':
    case '6h':
      return hm
    case '24h':
      return hm
    case '7d':
      return `${md} ${p(d.getHours())}h`
    case '30d':
    case '180d':
    default:
      return md
  }
}

const labels = computed(() => props.samples.map((s) => labelFor(s.ts, props.range)))

/** Read a resolved color from a CSS custom property on <html>. */
function cssVar(name: string, fallback: string): string {
  const v = getComputedStyle(document.documentElement).getPropertyValue(name).trim()
  return v || fallback
}

function withAlpha(hex: string, alpha: number): string {
  // color-mix keeps this theme-token-driven without hard-coding rgba literals.
  return `color-mix(in srgb, ${hex} ${Math.round(alpha * 100)}%, transparent)`
}

function buildDatasets(): ChartDataset<'line'>[] {
  return props.series.map((s) => {
    const color = cssVar(s.colorVar, '#22d3ee')
    return {
      label: s.label,
      data: props.samples.map((p) => clean(s.pick(p))),
      borderColor: color,
      backgroundColor: withAlpha(color, 0.12),
      borderWidth: 1.5,
      pointRadius: 0,
      pointHoverRadius: 3,
      pointHitRadius: 8,
      tension: 0.25,
      fill: props.series.length === 1,
      spanGaps: false,
      yAxisID: s.axis ?? 'y',
    }
  })
}

function buildConfig(): ChartConfiguration<'line'> {
  const text = cssVar('--text-dim', '#8b9bab')
  const faint = cssVar('--text-faint', '#5c6b7a')
  const grid = withAlpha(cssVar('--border', '#263340'), 0.6)
  const surface = cssVar('--surface', '#111820')
  const border = cssVar('--border-strong', '#33475a')
  const fontUi = cssVar('--font-ui', 'system-ui, sans-serif')

  const hasY2 = props.series.some((s) => (s.axis ?? 'y') === 'y2')
  const unitByLabel = new Map(props.series.map((s) => [s.label, s.unit ?? '']))

  return {
    type: 'line',
    data: { labels: labels.value, datasets: buildDatasets() },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      animation: { duration: 180 },
      interaction: { mode: 'index', intersect: false },
      plugins: {
        legend: {
          display: props.series.length > 1,
          position: 'top',
          align: 'end',
          labels: {
            color: text,
            boxWidth: 10,
            boxHeight: 10,
            usePointStyle: true,
            font: { family: fontUi, size: 11 },
          },
        },
        tooltip: {
          backgroundColor: surface,
          borderColor: border,
          borderWidth: 1,
          titleColor: text,
          bodyColor: cssVar('--text', '#e6edf3'),
          padding: 10,
          cornerRadius: 6,
          titleFont: { family: fontUi, size: 11 },
          bodyFont: { family: 'ui-monospace, monospace', size: 12 },
          callbacks: {
            label(ctx) {
              const raw = ctx.parsed.y
              if (raw === null || raw === undefined) return `${ctx.dataset.label}: —`
              const unit = unitByLabel.get(ctx.dataset.label ?? '') ?? ''
              const val = Number.isInteger(raw) ? raw : Number(raw.toFixed(2))
              return `${ctx.dataset.label}: ${val}${unit}`
            },
          },
        },
      },
      scales: {
        x: {
          grid: { color: grid, drawTicks: false },
          border: { color: grid },
          ticks: {
            color: faint,
            font: { family: 'ui-monospace, monospace', size: 10 },
            maxRotation: 0,
            autoSkip: true,
            maxTicksLimit: 8,
          },
        },
        y: {
          position: 'left',
          beginAtZero: true,
          min: props.yPercent ? 0 : undefined,
          max: props.yPercent ? 100 : undefined,
          grid: { color: grid, drawTicks: false },
          border: { display: false },
          ticks: {
            color: faint,
            font: { family: 'ui-monospace, monospace', size: 10 },
            maxTicksLimit: 6,
          },
          title: props.yLabel
            ? { display: true, text: props.yLabel, color: text, font: { family: fontUi, size: 11 } }
            : { display: false },
        },
        ...(hasY2
          ? {
              y2: {
                position: 'right' as const,
                beginAtZero: true,
                grid: { drawOnChartArea: false },
                border: { display: false },
                ticks: {
                  color: faint,
                  font: { family: 'ui-monospace, monospace', size: 10 },
                  maxTicksLimit: 6,
                },
                title: props.y2Label
                  ? {
                      display: true,
                      text: props.y2Label,
                      color: text,
                      font: { family: fontUi, size: 11 },
                    }
                  : { display: false },
              },
            }
          : {}),
      },
    },
  }
}

function render(): void {
  if (!canvas.value) return
  destroy()
  chart = new Chart(canvas.value, buildConfig())
}

function destroy(): void {
  if (chart) {
    chart.destroy()
    chart = null
  }
}

onMounted(render)
onBeforeUnmount(destroy)

// Re-render on any prop change (data, series set, range, or percentage clamp).
watch(
  () => [props.samples, props.series, props.range, props.yPercent],
  () => render(),
  { deep: true },
)

// Theme flip changes every resolved CSS var -> rebuild so colors track it.
watch(theme, () => render())
</script>

<template>
  <div class="metrics-chart" :style="{ height: `${height}px` }">
    <canvas ref="canvas" aria-label="metrics chart" role="img"></canvas>
  </div>
</template>

<style scoped>
.metrics-chart {
  position: relative;
  width: 100%;
}
</style>
