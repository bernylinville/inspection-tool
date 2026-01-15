<template>
  <a-drawer
    :open="open"
    :width="600"
    title="Redis 实例详情"
    @close="emit('close')"
  >
    <a-spin :spinning="loading">
      <div v-if="instance" class="space-y-6">
        <a-descriptions title="基本信息" :column="1" bordered>
          <a-descriptions-item label="地址">{{ instance.address || '-' }}</a-descriptions-item>
          <a-descriptions-item label="IP">{{ instance.ip || '-' }}</a-descriptions-item>
          <a-descriptions-item label="端口">{{ instance.port ?? '-' }}</a-descriptions-item>
          <a-descriptions-item label="状态">
            <a-tag :color="instance.status === 'online' ? 'green' : 'default'">
              {{ instance.status === 'online' ? '在线' : '离线' }}
            </a-tag>
          </a-descriptions-item>
        </a-descriptions>

        <a-descriptions title="集群信息" :column="1" bordered>
          <a-descriptions-item label="版本">{{ instance.version || '-' }}</a-descriptions-item>
          <a-descriptions-item label="集群模式">{{ instance.cluster_mode || '-' }}</a-descriptions-item>
          <a-descriptions-item label="角色">{{ formatRole(instance.role) }}</a-descriptions-item>
        </a-descriptions>

        <a-descriptions title="时间信息" :column="1" bordered>
          <a-descriptions-item label="创建时间">{{ formatDate(instance.created_at) }}</a-descriptions-item>
          <a-descriptions-item label="最后同步">{{ formatDate(instance.last_sync_at) }}</a-descriptions-item>
        </a-descriptions>
      </div>
      <div v-else class="py-8 text-center text-sm text-gray-500">暂无 Redis 详情</div>
    </a-spin>
  </a-drawer>
</template>

<script lang="ts" setup>
import type { RedisInstance } from '#/api/cmdb/types';

defineProps<{
  instance: RedisInstance | null;
  open: boolean;
  loading: boolean;
}>();

const emit = defineEmits<{
  (event: 'close'): void;
}>();

const formatRole = (value?: RedisInstance['role']) => {
  if (!value) {
    return '-';
  }
  if (value === 'master') {
    return '主节点';
  }
  if (value === 'slave') {
    return '从节点';
  }
  return value;
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
