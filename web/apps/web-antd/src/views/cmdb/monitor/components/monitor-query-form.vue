<template>
  <a-form layout="vertical" class="space-y-4">
    <a-form-item label="查询语句">
      <a-textarea
        v-model:value="formState.query"
        :auto-size="{ minRows: 3, maxRows: 6 }"
        placeholder="请输入 PromQL 查询"
      />
    </a-form-item>

    <a-form-item label="时间范围">
      <div class="flex flex-wrap items-center gap-3">
        <a-range-picker
          v-model:value="formState.timeRange"
          show-time
          class="min-w-[260px]"
        />
        <div class="flex flex-wrap items-center gap-2">
          <a-button size="small" @click="setQuickRange('1h')">1h</a-button>
          <a-button size="small" @click="setQuickRange('6h')">6h</a-button>
          <a-button size="small" @click="setQuickRange('24h')">24h</a-button>
          <a-button size="small" @click="setQuickRange('7d')">7d</a-button>
        </div>
      </div>
    </a-form-item>

    <a-form-item label="步长">
      <a-select v-model:value="formState.step" class="w-32">
        <a-select-option value="15">15s</a-select-option>
        <a-select-option value="30">30s</a-select-option>
        <a-select-option value="60">1m</a-select-option>
        <a-select-option value="300">5m</a-select-option>
        <a-select-option value="900">15m</a-select-option>
      </a-select>
    </a-form-item>

    <a-form-item>
      <a-button type="primary" :loading="loading" @click="handleQuery">
        <template #icon>
          <SearchOutlined />
        </template>
        查询
      </a-button>
    </a-form-item>
  </a-form>
</template>

<script lang="ts" setup>
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';

import { reactive } from 'vue';

import { SearchOutlined } from '@ant-design/icons-vue';

interface QueryPayload {
  query: string;
  start: number;
  end: number;
  step: string;
}

interface FormState {
  query: string;
  timeRange: [Dayjs, Dayjs];
  step: string;
}

interface Props {
  loading?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
});

const emit = defineEmits<{
  (event: 'query', payload: QueryPayload): void;
}>();

const formState = reactive<FormState>({
  query: '',
  timeRange: [dayjs().subtract(1, 'hour'), dayjs()],
  step: '60',
});

const setQuickRange = (range: '1h' | '6h' | '24h' | '7d') => {
  const end = dayjs();
  const start =
    range === '1h'
      ? end.subtract(1, 'hour')
      : range === '6h'
        ? end.subtract(6, 'hour')
        : range === '24h'
          ? end.subtract(24, 'hour')
          : end.subtract(7, 'day');

  formState.timeRange = [start, end];
};

const handleQuery = () => {
  const [start, end] = formState.timeRange;
  emit('query', {
    query: formState.query.trim(),
    start: start.valueOf(),
    end: end.valueOf(),
    step: formState.step,
  });
};
</script>
