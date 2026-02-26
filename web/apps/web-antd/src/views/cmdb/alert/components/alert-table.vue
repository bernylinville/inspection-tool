<script setup lang="ts">
import { computed } from 'vue';
import { Table, Tag, Button, Pagination } from 'ant-design-vue';
import type { TableColumnType } from 'ant-design-vue';
import type { Alert } from '#/api/cmdb/types';

interface Props {
  alerts: Alert[];
  loading: boolean;
  total: number;
  page: number;
  pageSize: number;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  'page-change': [page: number, pageSize: number];
  'view-detail': [record: Alert];
}>();

// Severity color mapping
const severityColors: Record<string, string> = {
  critical: 'red',
  warning: 'orange',
  info: 'blue',
};

const severityLabels: Record<string, string> = {
  critical: '严重',
  warning: '警告',
  info: '信息',
};

// Status color mapping
const statusColors: Record<string, string> = {
  firing: 'red',
  resolved: 'green',
};

const statusLabels: Record<string, string> = {
  firing: '触发中',
  resolved: '已恢复',
};

// Format date
function formatDate(dateStr: string): string {
  if (!dateStr) return '-';
  const date = new Date(dateStr);
  return date.toLocaleString('zh-CN');
}

const columns = computed<TableColumnType<Alert>[]>(() => [
  { title: '告警名称', dataIndex: 'title', key: 'title', ellipsis: true },
  { title: '级别', dataIndex: 'severity', key: 'severity', width: 100 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 100 },
  { title: '来源', dataIndex: 'source', key: 'source', width: 120 },
  { title: '触发时间', dataIndex: 'created_at', key: 'created_at', width: 180 },
  { title: '操作', key: 'action', width: 100, fixed: 'right' },
]);

function handlePageChange(newPage: number, newPageSize: number) {
  emit('page-change', newPage, newPageSize);
}
</script>

<template>
  <Table
    :columns="columns"
    :data-source="alerts"
    :loading="loading"
    :pagination="false"
    row-key="id"
    :scroll="{ x: 800 }"
  >
    <template #bodyCell="{ column, record }">
      <template v-if="column.key === 'severity'">
        <Tag :color="severityColors[record.severity] || 'default'">
          {{ severityLabels[record.severity] || record.severity }}
        </Tag>
      </template>
      <template v-else-if="column.key === 'status'">
        <Tag :color="statusColors[record.status] || 'default'">
          {{ statusLabels[record.status] || record.status }}
        </Tag>
      </template>
      <template v-else-if="column.key === 'created_at'">
        {{ formatDate(record.created_at) }}
      </template>
      <template v-else-if="column.key === 'action'">
        <Button type="link" size="small" @click="emit('view-detail', record as Alert)">
          详情
        </Button>
      </template>
    </template>
  </Table>
  <div class="mt-4 flex justify-end">
    <Pagination
      :current="page"
      :page-size="pageSize"
      :total="total"
      show-size-changer
      show-quick-jumper
      :show-total="(t: number) => `共 ${t} 条`"
      @change="handlePageChange"
    />
  </div>
</template>
