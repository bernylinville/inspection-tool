<template>
  <div class="flex flex-col gap-4">
    <a-table
      row-key="id"
      :columns="columns"
      :data-source="clusters"
      :loading="loading"
      :pagination="false"
      class="bg-white"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'status'">
          <a-tag :color="statusColor(record.status)">
            {{ record.status || '-' }}
          </a-tag>
        </template>
        <template v-else-if="column.key === 'last_sync_at'">
          <span>{{ formatDate(record.last_sync_at) }}</span>
        </template>
        <template v-else-if="column.key === 'actions'">
          <div class="flex items-center gap-2">
            <a-button type="link" size="small" @click="emit('view-detail', record)">查看</a-button>
            <a-button type="link" size="small" danger @click="emit('delete', record.id)">删除</a-button>
          </div>
        </template>
      </template>
    </a-table>

    <div class="flex items-center justify-end">
      <a-pagination
        :current="page"
        :page-size="pageSize"
        :total="total"
        :show-size-changer="true"
        :show-quick-jumper="true"
        :show-total="showTotal"
        @change="handlePageChange"
        @show-size-change="handlePageChange"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { TableColumnType } from 'ant-design-vue';
import type { ElasticsearchCluster } from '#/api/cmdb/types';

const props = defineProps<{
  clusters: ElasticsearchCluster[];
  loading: boolean;
  total: number;
  page: number;
  pageSize: number;
}>();

const emit = defineEmits<{
  (event: 'page-change', page: number, pageSize: number): void;
  (event: 'view-detail', cluster: ElasticsearchCluster): void;
  (event: 'delete', clusterId: number): void;
}>();

const columns: TableColumnType<ElasticsearchCluster>[] = [
  { title: '集群名称', dataIndex: 'cluster_name', key: 'cluster_name' },
  { title: '版本', dataIndex: 'version', key: 'version' },
  { title: '节点数', dataIndex: 'node_count', key: 'node_count' },
  { title: '状态', dataIndex: 'status', key: 'status' },
  { title: '最后同步', dataIndex: 'last_sync_at', key: 'last_sync_at' },
  { title: '操作', key: 'actions', fixed: 'right', width: 140 },
];

const statusColor = (value?: ElasticsearchCluster['status']) => {
  if (value === 'green') {
    return 'green';
  }
  if (value === 'yellow') {
    return 'orange';
  }
  if (value === 'red') {
    return 'red';
  }
  return 'default';
};

const formatDate = (value?: string | null) => {
  if (!value) {
    return '-';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
};

const showTotal = (totalCount: number) => `共 ${totalCount} 条`;

const handlePageChange = (nextPage: number, nextPageSize: number) => {
  emit('page-change', nextPage, nextPageSize);
};
</script>
