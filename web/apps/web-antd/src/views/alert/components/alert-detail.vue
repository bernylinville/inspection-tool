<script setup lang="ts">
import { Modal, Descriptions, DescriptionsItem, Tag } from 'ant-design-vue';
import type { Alert } from '#/api/cmdb/types';

interface Props {
  visible: boolean;
  alert: Alert | null;
}

defineProps<Props>();

const emit = defineEmits<{
  'update:visible': [visible: boolean];
}>();

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

const statusColors: Record<string, string> = {
  firing: 'red',
  resolved: 'green',
};

const statusLabels: Record<string, string> = {
  firing: '触发中',
  resolved: '已恢复',
};

function formatDate(dateStr: string): string {
  if (!dateStr) return '-';
  return new Date(dateStr).toLocaleString('zh-CN');
}

function handleClose() {
  emit('update:visible', false);
}
</script>

<template>
  <Modal
    :open="visible"
    title="告警详情"
    :footer="null"
    width="600px"
    @cancel="handleClose"
  >
    <Descriptions v-if="alert" bordered :column="1">
      <DescriptionsItem label="告警ID">{{ alert.id }}</DescriptionsItem>
      <DescriptionsItem label="告警名称">{{ alert.title }}</DescriptionsItem>
      <DescriptionsItem label="级别">
        <Tag :color="severityColors[alert.severity] || 'default'">
          {{ severityLabels[alert.severity] || alert.severity }}
        </Tag>
      </DescriptionsItem>
      <DescriptionsItem label="状态">
        <Tag :color="statusColors[alert.status] || 'default'">
          {{ statusLabels[alert.status] || alert.status }}
        </Tag>
      </DescriptionsItem>
      <DescriptionsItem label="来源">{{ alert.source || '-' }}</DescriptionsItem>
      <DescriptionsItem label="触发时间">{{ formatDate(alert.created_at) }}</DescriptionsItem>
      <DescriptionsItem label="描述">{{ alert.description || '-' }}</DescriptionsItem>
    </Descriptions>
  </Modal>
</template>
