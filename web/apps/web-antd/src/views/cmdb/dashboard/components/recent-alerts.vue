<script lang="ts" setup>
import type { Alert } from '#/api/cmdb/types';

import { computed } from 'vue';
import { useRouter } from 'vue-router';

import { Card, CardContent, CardHeader, CardTitle } from '@vben-core/shadcn-ui';

interface Props {
  alerts: Alert[];
  loading?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
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
  <Card>
    <CardHeader class="flex items-center justify-between">
      <CardTitle>最近告警</CardTitle>
      <button class="text-sm text-primary" type="button" @click="goToAlerts">
        查看全部
      </button>
    </CardHeader>
    <CardContent>
      <div v-if="props.loading" class="space-y-3">
        <div
          v-for="index in 4"
          :key="index"
          class="h-6 rounded bg-muted/40 animate-pulse"
        />
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
    </CardContent>
  </Card>
</template>
