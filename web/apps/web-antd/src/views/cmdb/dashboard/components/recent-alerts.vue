<script lang="ts" setup>
import type { Alert } from '#/api/cmdb/types';

import { computed } from 'vue';
import { useRouter } from 'vue-router';

import { Card } from 'ant-design-vue';

interface Props {
  alerts: Alert[];
  loading?: boolean;
  serviceUnavailable?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  serviceUnavailable: false,
});

const router = useRouter();

const severityClass = computed(() => ({
  critical: 'bg-red-100 text-red-800',
  warning: 'bg-yellow-100 text-yellow-800',
  default: 'bg-blue-100 text-blue-800',
}));

const formatTime = (value?: string) => {
  if (!value) {
    return '-';
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  const month = String(date.getMonth() + 1);
  const day = String(date.getDate());
  const hours = String(date.getHours()).padStart(2, '0');
  const minutes = String(date.getMinutes()).padStart(2, '0');

  return `${month}/${day} ${hours}:${minutes}`;
};

const goToAlerts = () => {
  void router.push('/cmdb/alert');
};
</script>

<template>
  <Card size="small">
    <template #title>
      <div class="flex items-center justify-between">
        <span>最近告警</span>
        <button class="text-sm text-primary" type="button" @click="goToAlerts">
          查看全部
        </button>
      </div>
    </template>
      <div v-if="props.loading" class="space-y-3">
        <div
          v-for="index in 4"
          :key="index"
          class="h-6 rounded bg-muted/40 animate-pulse"
        />
      </div>
      <div
        v-else-if="props.serviceUnavailable"
        class="rounded border border-warning/50 bg-warning/10 p-4"
      >
        <div class="text-sm text-warning">告警服务暂不可用</div>
        <div class="text-xs text-muted-foreground">无法连接到告警数据源</div>
      </div>
      <div v-else-if="!props.alerts.length" class="text-sm text-muted-foreground">
        暂无告警
      </div>
      <div v-else class="space-y-4">
        <div
          v-for="alert in props.alerts"
          :key="alert.id"
          class="flex items-center justify-between gap-4"
        >
          <div class="min-w-0">
            <div class="truncate text-sm font-medium">
              {{ alert.title }}
            </div>
            <div class="text-xs text-muted-foreground">
              {{ alert.source || '-' }}
            </div>
          </div>
          <div class="flex items-center gap-3 text-xs">
            <span
              class="rounded px-2 py-0.5"
              :class="severityClass[alert.severity] || severityClass.default"
            >
              {{ alert.severity }}
            </span>
            <span class="text-muted-foreground">{{ formatTime(alert.created_at) }}</span>
          </div>
        </div>
      </div>
  </Card>
</template>
