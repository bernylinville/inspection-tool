<template>
  <a-drawer
    :open="open"
    :width="600"
    title="Nginx 实例详情"
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

        <a-descriptions title="版本信息" :column="1" bordered>
          <a-descriptions-item label="版本">{{ instance.version || '-' }}</a-descriptions-item>
        </a-descriptions>

        <a-descriptions title="时间信息" :column="1" bordered>
          <a-descriptions-item label="创建时间">{{ formatDate(instance.created_at) }}</a-descriptions-item>
          <a-descriptions-item label="最后同步">{{ formatDate(instance.last_sync_at) }}</a-descriptions-item>
        </a-descriptions>
      </div>
      <div v-else class="py-8 text-center text-sm text-gray-500">暂无 Nginx 详情</div>
    </a-spin>
  </a-drawer>
</template>

<script lang="ts" setup>
import type { NginxInstance } from '#/api/cmdb/types';

defineProps<{
  instance: NginxInstance | null;
  open: boolean;
  loading: boolean;
}>();

const emit = defineEmits<{
  (event: 'close'): void;
}>();

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
