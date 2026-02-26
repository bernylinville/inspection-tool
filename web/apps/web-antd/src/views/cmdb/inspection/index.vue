<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue';
import { Card, Button, Modal, message } from 'ant-design-vue';
import { PlusOutlined } from '@ant-design/icons-vue';
import InspectionTable from './components/inspection-table.vue';
import InspectionSearch from './components/inspection-search.vue';
import InspectionDetail from './components/inspection-detail.vue';
import CreateJobModal from './components/create-job-modal.vue';
import { useInspectionList } from './composables/use-inspection-list';
import type { InspectionJob, CreateInspectionJobRequest } from '#/api/cmdb/types';

const {
  jobs,
  total,
  page,
  pageSize,
  loading,
  filters,
  selectedJob,
  detailVisible,
  createLoading,
  fetchJobs,
  changePage,
  applyFilters,
  resetFilters,
  createJob,
  deleteJob,
  viewDetail,
  closeDetail,
  stopPolling,
} = useInspectionList();

const createModalVisible = ref(false);
const deleteModalVisible = ref(false);
const jobToDelete = ref<InspectionJob | null>(null);

onMounted(() => {
  fetchJobs();
});

onUnmounted(() => {
  stopPolling();
});

function handleView(job: InspectionJob) {
  viewDetail(job.id);
}

function handleDeleteClick(job: InspectionJob) {
  jobToDelete.value = job;
  deleteModalVisible.value = true;
}

async function confirmDelete() {
  if (jobToDelete.value) {
    const success = await deleteJob(jobToDelete.value.id);
    if (success) {
      message.success('删除成功');
    }
  }
  deleteModalVisible.value = false;
  jobToDelete.value = null;
}

function cancelDelete() {
  deleteModalVisible.value = false;
  jobToDelete.value = null;
}

async function handleCreate(request: CreateInspectionJobRequest) {
  const success = await createJob(request);
  if (success) {
    message.success('巡检任务创建成功');
    createModalVisible.value = false;
  }
}
</script>

<template>
  <div class="p-4">
    <div class="mb-4 flex items-center justify-between">
      <h2 class="text-xl font-semibold">巡检管理</h2>
      <Button type="primary" @click="createModalVisible = true">
        <template #icon><PlusOutlined /></template>
        创建巡检
      </Button>
    </div>

    <Card class="mb-4">
      <InspectionSearch
        v-model:status="filters.status"
        v-model:type="filters.type"
        @search="applyFilters"
        @reset="resetFilters"
      />
    </Card>

    <Card>
      <InspectionTable
        :data="jobs"
        :loading="loading"
        :total="total"
        :page="page"
        :page-size="pageSize"
        @change="changePage"
        @view="handleView"
        @delete="handleDeleteClick"
      />
    </Card>

    <InspectionDetail
      :visible="detailVisible"
      :job="selectedJob"
      @close="closeDetail"
    />

    <CreateJobModal
      :visible="createModalVisible"
      :loading="createLoading"
      @close="createModalVisible = false"
      @create="handleCreate"
    />

    <Modal
      :open="deleteModalVisible"
      title="确认删除"
      @ok="confirmDelete"
      @cancel="cancelDelete"
    >
      <p>确定要删除该巡检任务吗？此操作不可恢复。</p>
    </Modal>
  </div>
</template>
