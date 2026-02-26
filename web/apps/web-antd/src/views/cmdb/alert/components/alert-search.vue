<script setup lang="ts">
import { ref, watch } from 'vue';
import { Form, FormItem, Select, Button, RangePicker } from 'ant-design-vue';
import type { AlertStatus } from '#/api/cmdb/types';
import type { Dayjs } from 'dayjs';

interface Filters {
  status?: AlertStatus;
  startTime?: number;
  endTime?: number;
}

interface Props {
  filters: Filters;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  search: [];
  reset: [];
  'update:filters': [filters: Filters];
}>();

const localStatus = ref<AlertStatus | undefined>(props.filters.status);
const timeRange = ref<[Dayjs, Dayjs] | undefined>(undefined);

const statusOptions = [
  { value: undefined, label: '全部状态' },
  { value: 'firing', label: '触发中' },
  { value: 'resolved', label: '已恢复' },
];

watch(() => props.filters, (newFilters) => {
  localStatus.value = newFilters.status;
}, { deep: true });

function handleSearch() {
  const newFilters: Filters = {
    status: localStatus.value,
  };
  if (timeRange.value && timeRange.value[0] && timeRange.value[1]) {
    newFilters.startTime = Math.floor(timeRange.value[0].valueOf() / 1000);
    newFilters.endTime = Math.floor(timeRange.value[1].valueOf() / 1000);
  }
  emit('update:filters', newFilters);
  emit('search');
}

function handleReset() {
  localStatus.value = undefined;
  timeRange.value = undefined;
  emit('update:filters', {});
  emit('reset');
}
</script>

<template>
  <Form layout="inline">
    <FormItem label="时间范围">
      <RangePicker
        v-model:value="timeRange"
        show-time
        format="YYYY-MM-DD HH:mm"
        :placeholder="['开始时间', '结束时间']"
        style="width: 340px"
      />
    </FormItem>
    <FormItem label="状态">
      <Select
        v-model:value="localStatus"
        :options="statusOptions"
        placeholder="选择状态"
        style="width: 120px"
        allow-clear
      />
    </FormItem>
    <FormItem>
      <Button type="primary" @click="handleSearch">查询</Button>
    </FormItem>
    <FormItem>
      <Button @click="handleReset">重置</Button>
    </FormItem>
  </Form>
</template>
