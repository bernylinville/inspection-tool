import { computed, ref } from 'vue';

import { queryMetricsRangeApi } from '#/api/cmdb/monitor';
import type {
  EChartsResponse,
  EChartsSeries,
  MonitorRangeQueryParams,
} from '#/api/cmdb/types';

export function useMonitorQuery() {
  const query = ref('');
  const now = Date.now();
  const startTime = ref(now - 60 * 60 * 1000);
  const endTime = ref(now);
  const step = ref('60');
  const loading = ref(false);
  const error = ref<Error | null>(null);
  const chartData = ref<EChartsSeries[]>([]);

  const timeRange = computed(() => ({
    start: startTime.value,
    end: endTime.value,
  }));

  const executeQuery = async () => {
    loading.value = true;
    error.value = null;

    const params: MonitorRangeQueryParams = {
      query: query.value,
      start: String(startTime.value),
      end: String(endTime.value),
      step: step.value,
      format: 'echarts',
    };

    try {
      const response: EChartsResponse = await queryMetricsRangeApi(params);
      chartData.value = response.series ?? [];
    } catch (err) {
      error.value = err instanceof Error ? err : new Error('查询失败');
      chartData.value = [];
    } finally {
      loading.value = false;
    }
  };

  const setTimeRange = (start: number, end: number) => {
    startTime.value = start;
    endTime.value = end;
  };

  const setStep = (nextStep: string) => {
    step.value = nextStep;
  };

  const clearError = () => {
    error.value = null;
  };

  return {
    query,
    startTime,
    endTime,
    step,
    loading,
    error,
    chartData,
    timeRange,
    executeQuery,
    setTimeRange,
    setStep,
    clearError,
  };
}
