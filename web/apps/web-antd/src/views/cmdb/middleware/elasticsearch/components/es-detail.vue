<template>
  <a-drawer
    :open="open"
    :width="600"
    title="Elasticsearch 集群详情"
    @close="emit('close')"
  >
    <a-spin :spinning="loading">
      <div v-if="cluster" class="space-y-6">
        <a-descriptions title="基本信息" :column="1" bordered>
          <a-descriptions-item label="集群名称">
            {{ cluster.cluster_name || '-' }}
          </a-descriptions-item>
          <a-descriptions-item label="状态">
            <a-tag :color="statusColor(cluster.status)">
              {{ cluster.status || '-' }}
            </a-tag>
          </a-descriptions-item>
        </a-descriptions>

        <a-descriptions title="集群信息" :column="1" bordered>
          <a-descriptions-item label="版本">{{ cluster.version || '-' }}</a-descriptions-item>
          <a-descriptions-item label="节点数">{{ cluster.node_count ?? '-' }}</a-descriptions-item>
        </a-descriptions>

        <a-descriptions title="时间信息" :column="1" bordered>
          <a-descriptions-item label="创建时间">{{ formatDate(cluster.created_at) }}</a-descriptions-item>
          <a-descriptions-item label="最后同步">{{ formatDate(cluster.last_sync_at) }}</a-descriptions-item>
        </a-descriptions>
      </div>
      <div v-else class="py-8 text-center text-sm text-gray-500">暂无 Elasticsearch 详情</div>
    </a-spin>
  </a-drawer>
</template>

<script lang="ts" setup>
import type { ElasticsearchCluster } from '#/api/cmdb/types';

defineProps<{
  cluster: ElasticsearchCluster | null;
  open: boolean;
  loading: boolean;
}>();

const emit = defineEmits<{
  (event: 'close'): void;
}>();

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
</script>
