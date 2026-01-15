<template>
  <div class="flex flex-col gap-4 p-5">
    <h1 class="text-xl font-semibold">监控查询</h1>

    <Alert
      v-if="error"
      type="error"
      :message="error?.message || '查询失败'"
      show-icon
      closable
      @close="clearError"
    />

    <Card>
      <MonitorQueryForm :loading="loading" @query="handleQuery" />
    </Card>

    <Card>
      <MonitorChart :series="chartData" :loading="loading" />
    </Card>
  </div>
</template>

<script lang="ts" setup>
import { Alert, Card } from 'ant-design-vue';

import MonitorChart from './components/monitor-chart.vue';
import MonitorQueryForm from './components/monitor-query-form.vue';
import { useMonitorQuery } from './composables/use-monitor-query';

interface QueryPayload {
  query: string;
  start: number;
  end: number;
  step: string;
}

const {
  query,
  loading,
  error,
  chartData,
  executeQuery,
  setTimeRange,
  setStep,
  clearError,
} = useMonitorQuery();

const handleQuery = ({ query: queryText, start, end, step }: QueryPayload) => {
  query.value = queryText;
  setTimeRange(start, end);
  setStep(step);
  clearError();
  executeQuery();
};
</script>
