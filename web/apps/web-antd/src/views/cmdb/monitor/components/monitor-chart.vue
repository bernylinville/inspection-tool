<script lang="ts" setup>
import type { EchartsUIType } from '@vben/plugins/echarts';

import { ref, watch } from 'vue';

import { EchartsUI, useEcharts } from '@vben/plugins/echarts';

import type { EChartsSeries } from '#/api/cmdb/types';

interface Props {
  series: EChartsSeries[];
  loading?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
});

const chartRef = ref<EchartsUIType>();
const { renderEcharts, showLoading } = useEcharts(chartRef);

const formatMetricLabel = (metric: Record<string, string>) => {
  const label = Object.entries(metric)
    .map(([key, value]) => `${key}=${value}`)
    .join(', ');

  return label || 'value';
};

const renderChart = () => {
  if (props.loading) {
    showLoading?.();
    return;
  }

  const series = props.series.map((item) => ({
    name: formatMetricLabel(item.metric),
    type: 'line',
    smooth: true,
    data: item.values.map((point) => [point.timestamp, point.value]),
  }));

  renderEcharts({
    grid: {
      bottom: '8%',
      containLabel: true,
      left: '2%',
      right: '2%',
      top: '8%',
    },
    legend: {
      top: 0,
    },
    tooltip: {
      trigger: 'axis',
    },
    xAxis: {
      type: 'time',
    },
    yAxis: {
      type: 'value',
    },
    series,
  });
};

watch(
  () => [props.series, props.loading],
  () => {
    renderChart();
  },
  { deep: true, immediate: true },
);
</script>

<template>
  <EchartsUI ref="chartRef" class="h-96" />
</template>
