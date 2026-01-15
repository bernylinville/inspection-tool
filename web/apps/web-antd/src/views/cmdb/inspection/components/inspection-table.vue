<script setup lang="ts">
import { Table, Tag, Button, Space } from 'ant-design-vue';
import type { InspectionJob } from '@/api/cmdb/types';
import dayjs from 'dayjs';

defineProps<{
  data: InspectionJob[];
  loading: boolean;
  total: number;
  page: number;
  pageSize: number;
}>();

const emit = defineEmits<{
  (e: 'change', page: number, pageSize: number): void;
  (e: 'view', record: InspectionJob): void;
  (e: 'delete', record: InspectionJob): void;
}>();

const statusColors: Record<string, string> = {
  pending: 'default',
  running: 'processing',
  success: 'success',
  failed: 'error',
};

const typeLabels: Record<string, string> = {
  host: '主机巡检',
  mysql: 'MySQL 巡检',
  redis: 'Redis 巡检',
  nginx: 'Nginx 巡检',
  tomcat: 'Tomcat 巡检',
  elasticsearch: 'Elasticsearch 巡检',
};

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '类型', dataIndex: 'type', key: 'type', width: 150 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 100 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 180 },
  { title: '开始时间', dataIndex: 'started_at', key: 'started_at', width: 180 },
  { title: '完成时间', dataIndex: 'completed_at', key: 'completed_at', width: 180 },
  { title: '操作', key: 'actions', width: 150, fixed: 'right' as const },
];

function formatTime(time: string | null): string {
  if (!time) return '-';
  return dayjs(time).format('YYYY-MM-DD HH:mm:ss');
}

function handleTableChange(pagination: any) {
  emit('change', pagination.current, pagination.pageSize);
}
</script>

<template>
  <Table
    :columns="columns"
    :data-source="data"
    :loading="loading"
    :pagination="{
      current: page,
      pageSize: pageSize,
      total: total,
      showSizeChanger: true,
      showQuickJumper: true,
      showTotal: (t: number) => `共 ${t} 条`,
    }"
    :row-key="(record: InspectionJob) => record.id"
    :scroll="{ x: 1000 }"
    @change="handleTableChange"
  >
    <template #bodyCell="{ column, record }">
      <template v-if="column.key === 'type'">
        {{ typeLabels[(record as InspectionJob).type] || (record as InspectionJob).type }}
      </template>
      <template v-else-if="column.key === 'status'">
        <Tag :color="statusColors[(record as InspectionJob).status]">
          {{ (record as InspectionJob).status }}
        </Tag>
      </template>
      <template v-else-if="column.key === 'created_at'">
        {{ formatTime((record as InspectionJob).created_at) }}
      </template>
      <template v-else-if="column.key === 'started_at'">
        {{ formatTime((record as InspectionJob).started_at) }}
      </template>
      <template v-else-if="column.key === 'completed_at'">
        {{ formatTime((record as InspectionJob).completed_at) }}
      </template>
      <template v-else-if="column.key === 'actions'">
        <Space>
          <Button type="link" size="small" @click="emit('view', record as InspectionJob)">
            详情
          </Button>
          <Button
            type="link"
            size="small"
            danger
            :disabled="(record as InspectionJob).status === 'running'"
            @click="emit('delete', record as InspectionJob)"
          >
            删除
          </Button>
        </Space>
      </template>
    </template>
  </Table>
</template>
