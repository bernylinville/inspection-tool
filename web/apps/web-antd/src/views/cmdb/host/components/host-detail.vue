<template>
  <a-drawer
    :open="open"
    :width="600"
    title="主机详情"
    @close="emit('close')"
  >
    <a-spin :spinning="loading">
      <div v-if="host" class="space-y-6">
        <a-descriptions title="基本信息" :column="1" bordered>
          <a-descriptions-item label="主机名">{{ host.hostname || '-' }}</a-descriptions-item>
          <a-descriptions-item label="IP地址">{{ host.ip || '-' }}</a-descriptions-item>
          <a-descriptions-item label="标识">{{ host.ident || '-' }}</a-descriptions-item>
          <a-descriptions-item label="状态">
            <a-tag :color="host.status === 'online' ? 'green' : 'default'">
              {{ host.status === 'online' ? '在线' : '离线' }}
            </a-tag>
          </a-descriptions-item>
        </a-descriptions>

        <a-descriptions title="系统信息" :column="1" bordered>
          <a-descriptions-item label="操作系统">{{ host.os || '-' }}</a-descriptions-item>
          <a-descriptions-item label="系统版本">{{ host.os_version || '-' }}</a-descriptions-item>
          <a-descriptions-item label="内核版本">{{ host.kernel_version || '-' }}</a-descriptions-item>
        </a-descriptions>

        <a-descriptions title="硬件信息" :column="1" bordered>
          <a-descriptions-item label="CPU型号">{{ host.cpu_model || '-' }}</a-descriptions-item>
          <a-descriptions-item label="CPU核心">{{ host.cpu_cores ?? '-' }}</a-descriptions-item>
          <a-descriptions-item label="内存">{{ formatMemory(host.memory_total) }}</a-descriptions-item>
        </a-descriptions>

        <a-descriptions title="元数据" :column="1" bordered>
          <a-descriptions-item label="业务组">{{ host.business_group || '-' }}</a-descriptions-item>
          <a-descriptions-item label="环境">{{ host.env || '-' }}</a-descriptions-item>
          <a-descriptions-item label="标签">
            <pre class="whitespace-pre-wrap text-xs text-gray-500">{{ formatTags(host.tags) }}</pre>
          </a-descriptions-item>
        </a-descriptions>

        <a-descriptions title="时间信息" :column="1" bordered>
          <a-descriptions-item label="创建时间">{{ formatDate(host.created_at) }}</a-descriptions-item>
          <a-descriptions-item label="最后同步">{{ formatDate(host.last_sync_at) }}</a-descriptions-item>
        </a-descriptions>
      </div>
      <div v-else class="py-8 text-center text-sm text-gray-500">暂无主机详情</div>
    </a-spin>
  </a-drawer>
</template>

<script lang="ts" setup>
import type { Host } from '#/api/cmdb/types';

const props = defineProps<{
  host: Host | null;
  open: boolean;
  loading: boolean;
}>();

const emit = defineEmits<{
  (event: 'close'): void;
}>();

const formatMemory = (value?: number | null) => {
  if (!value || value <= 0) {
    return '-';
  }
  const gb = value / 1024 / 1024 / 1024;
  return `${gb.toFixed(1)} GB`;
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

const formatTags = (value: Host['tags']) => {
  if (!value) {
    return '-';
  }
  try {
    return JSON.stringify(value, null, 2);
  } catch (err) {
    return String(value);
  }
};
</script>
