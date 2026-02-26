<script setup lang="ts">
import { Modal, Descriptions, DescriptionsItem, Tag, Button } from 'ant-design-vue';
import { downloadInspectionReportApi } from '#/api/cmdb/inspection';
import type { InspectionJob } from '#/api/cmdb/types';
import dayjs from 'dayjs';

const props = defineProps<{
  visible: boolean;
  job: InspectionJob | null;
}>();

const emit = defineEmits<{
  (e: 'close'): void;
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

async function handleDownload(format: 'excel' | 'html') {
  if (!props.job) {
    return;
  }

  const res = await downloadInspectionReportApi(props.job!.id, format);
  const blob = new Blob([res as any]);
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  const ext = format === 'excel' ? 'xlsx' : 'html';
  const filename = `inspection_report_${props.job!.id}.${ext}`;
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
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
      <DescriptionsItem label="项目">
        {{ job.project_code || '-' }}
      </DescriptionsItem>
      <DescriptionsItem label="状态">
        <Tag :color="statusColors[job.status]">{{ job.status }}</Tag>
      </DescriptionsItem>
      <DescriptionsItem label="创建人">{{ job.created_by || '-' }}</DescriptionsItem>
      <DescriptionsItem label="创建时间" :span="2">
        {{ formatTime(job.created_at) }}
      </DescriptionsItem>
      <DescriptionsItem label="开始时间">
        {{ formatTime(job.start_time || null) }}
      </DescriptionsItem>
      <DescriptionsItem label="完成时间">
        {{ formatTime(job.end_time || null) }}
      </DescriptionsItem>
      <DescriptionsItem v-if="job.status === 'success'" label="下载报告" :span="2">
        <Button type="primary" size="small" class="mr-2" @click="handleDownload('excel')">
          下载 Excel
        </Button>
        <Button size="small" @click="handleDownload('html')">
          下载 HTML
        </Button>
      </DescriptionsItem>
      <DescriptionsItem v-if="job.error_message" label="错误信息" :span="2">
        <span style="color: #ff4d4f">{{ job.error_message }}</span>
      </DescriptionsItem>
    </Descriptions>
  </Modal>
</template>
