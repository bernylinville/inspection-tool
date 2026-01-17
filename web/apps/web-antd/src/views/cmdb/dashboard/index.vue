<script lang="ts" setup>
import { Alert } from 'ant-design-vue';

import { AnalysisChartCard } from '@vben/common-ui';

import AlertTrend from './components/alert-trend.vue';
import AssetDistribution from './components/asset-distribution.vue';
import DashboardOverview from './components/dashboard-overview.vue';
import RecentAlerts from './components/recent-alerts.vue';
import { useDashboardData } from './composables/use-dashboard-data';

const {
  stats,
  middlewareDistribution,
  recentAlerts,
  loading,
  error,
  alertServiceUnavailable,
  refresh,
} = useDashboardData();
</script>

<template>
  <div class="p-4">
    <!-- Error Banner -->
    <Alert
      v-if="error"
      :message="error?.message"
      class="mb-4"
      closable
      show-icon
      type="error"
    >
      <template #action>
        <a-button size="small" type="primary" @click="refresh">
          重试
        </a-button>
      </template>
    </Alert>

    <!-- Overview Statistics -->
    <DashboardOverview :loading="loading" :stats="stats" />

    <!-- Charts Row -->
    <div class="mt-4 grid grid-cols-1 gap-4 lg:grid-cols-2">
      <!-- Asset Distribution -->
      <AnalysisChartCard title="资产分布">
        <AssetDistribution
          :distribution="middlewareDistribution"
          :host-count="stats.hostCount"
          :loading="loading"
        />
      </AnalysisChartCard>

      <!-- Alert Trend -->
      <AnalysisChartCard title="告警趋势（近7天）">
        <AlertTrend :loading="loading" />
      </AnalysisChartCard>
    </div>

    <!-- Recent Alerts -->
    <div class="mt-4">
      <RecentAlerts
        :alerts="recentAlerts"
        :loading="loading"
        :service-unavailable="alertServiceUnavailable"
      />
    </div>
  </div>
</template>
