<script lang="ts" setup>
import type { EchartsUIType } from '@vben/plugins/echarts';

import { onMounted, ref, watch } from 'vue';

import { EchartsUI, useEcharts } from '@vben/plugins/echarts';

interface Props {
  loading?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
});

const chartRef = ref<EchartsUIType>();
const chartLoading = ref(true);
const chartOption = ref<Record<string, unknown>>({});
const { renderEcharts, showLoading } = useEcharts(chartRef);

const renderChart = () => {
  if (props.loading || chartLoading.value) {
    showLoading?.();
    return;
  }

  renderEcharts(chartOption.value);
};

onMounted(() => {
  const labels: string[] = [];
  const criticalData: number[] = [];
  const warningData: number[] = [];

  for (let index = 6; index >= 0; index -= 1) {
    const date = new Date();
    date.setDate(date.getDate() - index);
    labels.push(`${date.getMonth() + 1}/${date.getDate()}`);
    criticalData.push(Math.floor(Math.random() * 10) + 2);
    warningData.push(Math.floor(Math.random() * 15) + 4);
  }

  chartOption.value = {
    grid: {
      bottom: '8%',
      containLabel: true,
      left: '1%',
      right: '3%',
      top: '6%',
    },
    legend: {
      top: 0,
    },
    series: [
      {
        areaStyle: {
          opacity: 0.3,
        },
        data: criticalData,
        itemStyle: {
          color: '#ef4444',
        },
        lineStyle: {
          color: '#ef4444',
        },
        name: '严重',
        smooth: true,
        type: 'line',
      },
      {
        areaStyle: {
          opacity: 0.3,
        },
        data: warningData,
        itemStyle: {
          color: '#f59e0b',
        },
        lineStyle: {
          color: '#f59e0b',
        },
        name: '警告',
        smooth: true,
        type: 'line',
      },
    ],
    tooltip: {
      trigger: 'axis',
    },
    xAxis: {
      data: labels,
      type: 'category',
    },
    yAxis: {
      splitNumber: 4,
      type: 'value',
    },
  };

  chartLoading.value = false;
  renderChart();
});

watch(
  () => [props.loading, chartLoading.value],
  () => {
    renderChart();
  },
);
</script>

<template>
  <EchartsUI ref="chartRef" class="h-64" />
</template>
