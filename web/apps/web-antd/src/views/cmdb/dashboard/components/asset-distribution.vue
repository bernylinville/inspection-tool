<script lang="ts" setup>
import type { EchartsUIType } from '@vben/plugins/echarts';

import { computed, ref, watch } from 'vue';

import { EchartsUI, useEcharts } from '@vben/plugins/echarts';

import type { MiddlewareDistribution } from '../composables/use-dashboard-data';

interface Props {
  distribution: MiddlewareDistribution;
  hostCount: number;
  loading?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
});

const chartRef = ref<EchartsUIType>();
const { renderEcharts, showLoading } = useEcharts(chartRef);

const chartData = computed(() => {
  const items = [
    { name: '主机', value: props.hostCount },
    { name: 'MySQL', value: props.distribution.mysql },
    { name: 'Redis', value: props.distribution.redis },
    { name: 'Nginx', value: props.distribution.nginx },
    { name: 'Tomcat', value: props.distribution.tomcat },
    { name: 'Elasticsearch', value: props.distribution.elasticsearch },
  ];

  return items.filter((item) => item.value > 0);
});

const renderChart = () => {
  if (props.loading) {
    showLoading?.();
    return;
  }

  renderEcharts({
    legend: {
      bottom: '2%',
      left: 'center',
    },
    series: [
      {
        animationDelay() {
          return Math.random() * 100;
        },
        animationEasing: 'exponentialInOut',
        animationType: 'scale',
        avoidLabelOverlap: false,
        data: chartData.value,
        emphasis: {
          label: {
            fontSize: '12',
            fontWeight: 'bold',
            show: true,
          },
        },
        itemStyle: {
          borderRadius: 10,
          borderWidth: 2,
        },
        label: {
          position: 'center',
          show: false,
        },
        labelLine: {
          show: false,
        },
        radius: ['40%', '65%'],
        type: 'pie',
      },
    ],
    tooltip: {
      trigger: 'item',
    },
  });
};

watch(
  () => [props.loading, chartData.value],
  () => {
    renderChart();
  },
  { immediate: true },
);
</script>

<template>
  <EchartsUI ref="chartRef" class="h-64" />
</template>
