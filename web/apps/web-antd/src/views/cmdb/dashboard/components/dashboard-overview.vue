<script lang="ts" setup>
import { computed } from 'vue';

import {
  AnalysisOverview,
  type AnalysisOverviewItem,
} from '@vben/common-ui';
import { Bell } from '@vben/icons';
import { ClipboardCheck, Database, FolderKanban, Server } from 'lucide-vue-next';

import type { DashboardStats } from '../composables/use-dashboard-data';

interface Props {
  stats: DashboardStats;
  loading?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
});

const overviewItems = computed<AnalysisOverviewItem[]>(() => [
  {
    icon: FolderKanban,
    title: '项目数',
    totalTitle: '应用数',
    totalValue: props.stats.applicationCount,
    value: props.stats.projectCount,
  },
  {
    icon: Server,
    title: '主机数',
    totalTitle: '总主机数',
    totalValue: props.stats.hostCount,
    value: props.stats.hostCount,
  },
  {
    icon: Database,
    title: '中间件实例',
    totalTitle: '总实例数',
    totalValue: props.stats.middlewareCount,
    value: props.stats.middlewareCount,
  },
  {
    icon: Bell,
    title: '活跃告警',
    totalTitle: '总告警数',
    totalValue: props.stats.alertCount,
    value: props.stats.alertCount,
  },
  {
    icon: ClipboardCheck,
    title: '巡检任务',
    totalTitle: '总任务数',
    totalValue: props.stats.inspectionCount,
    value: props.stats.inspectionCount,
  },
]);
</script>

<template>
  <div>
    <div v-if="props.loading" class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-5">
      <div
        v-for="index in 5"
        :key="index"
        class="h-28 rounded-lg border border-border bg-muted/40 animate-pulse"
      />
    </div>
    <AnalysisOverview v-else :items="overviewItems" />
  </div>
</template>
