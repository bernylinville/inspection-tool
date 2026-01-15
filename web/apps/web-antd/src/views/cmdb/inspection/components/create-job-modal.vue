<script setup lang="ts">
import { ref } from 'vue';
import { Modal, Form, FormItem, Select, Button } from 'ant-design-vue';
import type { CreateInspectionJobRequest } from '@/api/cmdb/types';

const props = defineProps<{
  visible: boolean;
  loading: boolean;
}>();

const emit = defineEmits<{
  (e: 'close'): void;
  (e: 'create', request: CreateInspectionJobRequest): void;
}>();

const jobType = ref('host');

const typeOptions = [
  { value: 'host', label: '主机巡检' },
  { value: 'mysql', label: 'MySQL 巡检' },
  { value: 'redis', label: 'Redis 巡检' },
  { value: 'nginx', label: 'Nginx 巡检' },
  { value: 'tomcat', label: 'Tomcat 巡检' },
  { value: 'elasticsearch', label: 'Elasticsearch 巡检' },
];

function handleOk() {
  emit('create', { type: jobType.value });
}

function handleCancel() {
  jobType.value = 'host';
  emit('close');
}
</script>

<template>
  <Modal
    :open="visible"
    title="创建巡检任务"
    :confirm-loading="loading"
    @ok="handleOk"
    @cancel="handleCancel"
  >
    <Form layout="vertical">
      <FormItem label="巡检类型" required>
        <Select
          v-model:value="jobType"
          :options="typeOptions"
          style="width: 100%"
        />
      </FormItem>
    </Form>
  </Modal>
</template>
