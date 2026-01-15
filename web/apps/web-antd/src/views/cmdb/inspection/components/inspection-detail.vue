<script setup lang="ts">
import { Modal, Descriptions, DescriptionsItem, Tag, Button } from 'ant-design-vue';
import type { InspectionJob } from '@/api/cmdb/types';
import dayjs from 'dayjs';

const props = defineProps<{
  visible: boolean;
  job: InspectionJob | null;
}>();

const emit = defineEmits<{
  (e: 'close'): void;
  (e: 'download', job: InspectionJob): void;
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

function formatTime(time: string | null): string {
  if (!time) return '-';
  return dayjs(time).format('YYYY-MM-DD HH:mm:ss');
}
</script>

<template>
  <Modal
    :open="visible"
    title="巡检任务详情"
    :footer="null"
    width="700px"
    @cancel="emit('close')"
  >
    <Descriptions v-if="job" :column="2" bordered>
      <DescriptionsItem label="任务ID">{{ job.id }}</DescriptionsItem>
      <DescriptionsItem label="类型">
        {{ typeLabels[job.type] || job.type }}
      </DescriptionsItem>
      <DescriptionsItem label="状态">
        <Tag :color="statusColors[job.status]">{{ job.status }}</Tag>
      </DescriptionsItem>
      <DescriptionsItem label="创建人">{{ job.created_by || '-' }}</DescriptionsItem>
      <DescriptionsItem label="创建时间" :span="2">
        {{ formatTime(job.created_at) }}
      </DescriptionsItem>
      <DescriptionsItem label="开始时间">
        {{ formatTime(job.started_at) }}
      </DescriptionsItem>
      <DescriptionsItem label="完成时间">
        {{ formatTime(job.completed_at) }}
      </DescriptionsItem>
      <DescriptionsItem v-if="job.report_path" label="报告路径" :span="2">
        {{ job.report_path }}
        <Button
          type="link"
          size="small"
          :disabled="job.status !== 'success'"
          @click="emit('download', job)"
        >
          下载
        </Button>
      </DescriptionsItem>
      <DescriptionsItem v-if="job.error" label="错误信息" :span="2">
        <span style="color: #ff4d4f">{{ job.error }}</span>
      </DescriptionsItem>
    </Descriptions>
  </Modal>
</template>
