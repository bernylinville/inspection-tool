<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { Modal, Form, FormItem, Select } from 'ant-design-vue';
import { listProjectsApi } from '#/api/cmdb/asset';
import type { CreateInspectionJobRequest, InspectionType, Project } from '#/api/cmdb/types';

const props = defineProps<{
  visible: boolean;
  loading: boolean;
}>();

const emit = defineEmits<{
  (e: 'close'): void;
  (e: 'create', request: CreateInspectionJobRequest): void;
}>();

const jobType = ref<InspectionType>('host');
const projectId = ref<number>(0);
const projects = ref<Project[]>([]);

const projectOptions = computed(() =>
  projects.value.map((p) => ({
    value: p.id,
    label: `${p.name} (${p.code})`,
  })),
);

const typeOptions = [
  { value: 'host', label: '主机巡检' },
  { value: 'mysql', label: 'MySQL 巡检' },
  { value: 'redis', label: 'Redis 巡检' },
  { value: 'nginx', label: 'Nginx 巡检' },
  { value: 'tomcat', label: 'Tomcat 巡检' },
  { value: 'elasticsearch', label: 'Elasticsearch 巡检' },
];

function handleOk() {
  emit('create', { type: jobType.value, project_id: projectId.value });
}

function handleCancel() {
  projectId.value = 0;
  jobType.value = 'host';
  emit('close');
}

onMounted(async () => {
  try {
    const res = await listProjectsApi({ page_size: 100 });
    projects.value = res.items || [];
  } catch {
    projects.value = [];
  }
});
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
      <FormItem label="选择项目" required>
        <Select
          v-model:value="projectId"
          :options="projectOptions"
          show-search
          style="width: 100%"
        />
      </FormItem>
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
