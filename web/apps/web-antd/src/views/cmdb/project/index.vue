// OC_SANITY
<script lang="ts" setup>
import { ref, onMounted } from 'vue';
import type { FormInstance } from 'ant-design-vue';
import { message } from 'ant-design-vue';
import { useProjectList } from './composables/use-project-list';
import {
  createProjectApi,
  updateProjectApi,
} from '#/api/cmdb/asset';
import type {
  Project,
  ProjectCreateRequest,
  ProjectUpdateRequest,
} from '#/api/cmdb/types';

defineOptions({
  name: 'ProjectManage',
});

const {
  loading,
  syncLoading,
  projects,
  pagination,
  searchParams,
  fetchProjects,
  syncProjects,
  deleteProject,
  handlePageChange,
  handleSearch,
  handleReset,
} = useProjectList();

const searchFormRef = ref<FormInstance>();
const selectedProject = ref<Project | null>(null);
const detailDrawerVisible = ref(false);

const statusOptions = [
  { label: 'Active', value: 'active' },
  { label: 'Inactive', value: 'inactive' },
];

const statusMap: Record<string, { text: string; color: string }> = {
  active: { text: 'Active', color: 'green' },
  inactive: { text: 'Inactive', color: 'red' },
};

const handleViewDetail = (record: Project) => {
  selectedProject.value = record;
  detailDrawerVisible.value = true;
};

const handleTableChange = (pag: { current?: number; pageSize?: number }) => {
  handlePageChange(pag.current || 1, pag.pageSize || 20);
};

const resetForm = () => {
  searchFormRef.value?.resetFields();
  searchParams.status = undefined;
  handleReset();
};

onMounted(() => {
  fetchProjects();
});
</script>

<template>
  <div class="p-5">
    <div class="mb-4 flex items-center justify-between">
      <h1 class="text-xl font-semibold">Project Management</h1>
      <a-button type="primary" :loading="syncLoading" @click="syncProjects">
        <template #icon>
          <SyncOutlined />
        </template>
        Sync from N9E
      </a-button>
    </div>

    <a-form ref="searchFormRef" :model="searchParams" layout="inline" class="mb-4">
      <a-form-item label="Status" name="status">
        <a-select
          v-model:value="searchParams.status"
          placeholder="Select status"
          allow-clear
          style="width: 150px"
        >
          <a-select-option
            v-for="option in statusOptions"
            :key="option.value"
            :value="option.value"
          >
            {{ option.label }}
          </a-select-option>
        </a-select>
      </a-form-item>
      <a-form-item>
        <a-space>
          <a-button type="primary" @click="handleSearch">
            <template #icon>
              <SearchOutlined />
            </template>
            Search
          </a-button>
          <a-button @click="resetForm">Reset</a-button>
        </a-space>
      </a-form-item>
    </a-form>

    <a-table
      :columns="[
        {
          title: 'Name',
          dataIndex: 'name',
          key: 'name',
        },
        {
          title: 'Code',
          dataIndex: 'code',
          key: 'code',
        },
        {
          title: 'Owner',
          dataIndex: 'owner',
          key: 'owner',
        },
        {
          title: 'Hosts',
          dataIndex: 'host_count',
          key: 'host_count',
        },
        {
          title: 'Status',
          dataIndex: 'status',
          key: 'status',
        },
        {
          title: 'Updated At',
          dataIndex: 'updated_at',
          key: 'updated_at',
        },
        {
          title: 'Actions',
          key: 'actions',
        },
      ]"
      :data-source="projects"
      :loading="loading"
      :pagination="pagination"
      row-key="id"
      @change="handleTableChange"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'status'">
          <a-tag :color="statusMap[record.status]?.color">
            {{ statusMap[record.status]?.text }}
          </a-tag>
        </template>
        <template v-else-if="column.key === 'host_count'">
          {{ record.host_count ?? 0 }}
        </template>
        <template v-else-if="column.key === 'updated_at'">
          {{ record.updated_at ? new Date(record.updated_at).toLocaleString() : '-' }}
        </template>
        <template v-else-if="column.key === 'actions'">
          <a-space>
            <a-button size="small" @click="handleViewDetail(record)">
              <template #icon>
                <EyeOutlined />
              </template>
              View
            </a-button>
            <a-popconfirm
              title="Are you sure you want to delete this project?"
              ok-text="Yes"
              cancel-text="No"
              @confirm="handleDelete(record.id)"
            >
              <a-button size="small" danger>
                <template #icon>
                  <DeleteOutlined />
                </template>
                Delete
              </a-button>
            </a-popconfirm>
          </a-space>
        </template>
      </template>
    </a-table>

    <a-drawer
      v-model:open="detailDrawerVisible"
      title="Project Details"
      width="600"
      placement="right"
    >
      <a-descriptions v-if="selectedProject" :column="1" bordered>
        <a-descriptions-item label="ID">
          {{ selectedProject.id }}
        </a-descriptions-item>
        <a-descriptions-item label="Name">
          {{ selectedProject.name }}
        </a-descriptions-item>
        <a-descriptions-item label="Code">
          {{ selectedProject.code }}
        </a-descriptions-item>
        <a-descriptions-item label="Description">
          {{ selectedProject.description || '-' }}
        </a-descriptions-item>
        <a-descriptions-item label="Owner">
          {{ selectedProject.owner || '-' }}
        </a-descriptions-item>
        <a-descriptions-item label="Host Count">
          {{ selectedProject.host_count ?? 0 }}
        </a-descriptions-item>
        <a-descriptions-item label="Status">
          <a-tag :color="statusMap[selectedProject.status]?.color">
            {{ statusMap[selectedProject.status]?.text }}
          </a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="Created At">
          {{ new Date(selectedProject.created_at).toLocaleString() }}
        </a-descriptions-item>
        <a-descriptions-item label="Updated At">
          {{ selectedProject.updated_at ? new Date(selectedProject.updated_at).toLocaleString() : '-' }}
        </a-descriptions-item>
      </a-descriptions>
    </a-drawer>
  </div>
</template>
