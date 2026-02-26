<script lang="ts" setup>
import { ref, onMounted, computed } from 'vue';
import type { FormInstance } from 'ant-design-vue';
import { message, Modal } from 'ant-design-vue';
import { PlusOutlined, SearchOutlined, ReloadOutlined, DeleteOutlined, EyeOutlined, EditOutlined } from '@ant-design/icons-vue';
import { useApplicationList } from './composables/use-application-list';
import {
  createApplicationApi,
  updateApplicationApi,
  getApplicationApi,
} from '#/api/cmdb/asset';
import { listProjectsApi } from '#/api/cmdb/asset';
import type {
  Application,
  ApplicationCreateRequest,
  ApplicationUpdateRequest,
  Project,
  ApplicationStatus,
} from '#/api/cmdb/types';

defineOptions({
  name: 'ApplicationManage',
});

const {
  loading,
  applications,
  pagination,
  searchParams,
  fetchApplications,
  deleteApplication,
  handlePageChange,
  handleSearch,
  handleReset,
} = useApplicationList();

const searchFormRef = ref<FormInstance>();
const projects = ref<Project[]>([]);
const projectOptions = computed(() => projects.value.map(p => ({ label: p.name, value: p.id })));
const projectMap = computed(() => {
  const map: Record<number, string> = {};
  projects.value.forEach(p => {
    map[p.id] = p.name;
  });
  return map;
});

const statusOptions = [
  { label: 'Active', value: 'active' as ApplicationStatus },
  { label: 'Inactive', value: 'inactive' as ApplicationStatus },
];

const statusMap: Record<ApplicationStatus, { text: string; color: string }> = {
  active: { text: 'Active', color: 'green' },
  inactive: { text: 'Inactive', color: 'red' },
};

const selectedApplication = ref<Application | null>(null);
const detailDrawerVisible = ref(false);
const modalVisible = ref(false);
const modalMode = ref<'create' | 'edit'>('create');
const modalLoading = ref(false);
const modalFormRef = ref<FormInstance>();

const handleFetchProjects = async () => {
  try {
    const res = await listProjectsApi({ page: 1, page_size: 1000 });
    projects.value = res.items || [];
  } catch (error) {
    message.error('Failed to load projects');
  }
};

const handleViewDetail = (record: Application) => {
  selectedApplication.value = record;
  detailDrawerVisible.value = true;
};

const handleTableChange = (pag: { current?: number; pageSize?: number }) => {
  handlePageChange(pag.current || 1, pag.pageSize || 20);
};

const resetForm = () => {
  searchFormRef.value?.resetFields();
  searchParams.name = undefined;
  searchParams.project_id = undefined;
  searchParams.status = undefined;
  handleReset();
};

const handleDelete = async (id: number) => {
  Modal.confirm({
    title: '确认删除',
    content: '确定删除此应用吗?',
    okText: '删除',
    cancelText: '取消',
    onOk: async () => {
      await deleteApplication(id);
    },
  });
};

const handleCreate = () => {
  modalMode.value = 'create';
  selectedApplication.value = {
    project_id: undefined,
    name: '',
    code: '',
    description: '',
    owner: '',
    status: 'active',
  } as any;
  modalVisible.value = true;
};

const handleEdit = async (record: Application) => {
  try {
    modalLoading.value = true;
    modalMode.value = 'edit';
    const res = await getApplicationApi(record.id);
    selectedApplication.value = res;
    modalVisible.value = true;
  } catch (error) {
    message.error('Failed to load application');
  } finally {
    modalLoading.value = false;
  }
};

const handleModalOk = async () => {
  try {
    await modalFormRef.value?.validate();
    modalLoading.value = true;

    const formData = modalFormRef.value?.getFieldsValue();

    if (modalMode.value === 'create') {
      const data: ApplicationCreateRequest = {
        project_id: formData.project_id,
        name: formData.name,
        code: formData.code,
        description: formData.description,
        owner: formData.owner,
      };
      await createApplicationApi(data);
      message.success('Application created successfully');
    } else if (selectedApplication.value) {
      const data: ApplicationUpdateRequest = {
        name: formData.name,
        description: formData.description,
        owner: formData.owner,
        status: formData.status,
      };
      await updateApplicationApi(selectedApplication.value.id, data);
      message.success('Application updated successfully');
    }

    modalVisible.value = false;
    await fetchApplications();
  } catch (error) {
    if (error && typeof error === 'object' && 'errorFields' in error) {
      return;
    }
    message.error(modalMode.value === 'create' ? 'Failed to create application' : 'Failed to update application');
  } finally {
    modalLoading.value = false;
  }
};

const handleModalCancel = () => {
  modalVisible.value = false;
  modalFormRef.value?.resetFields();
};

onMounted(() => {
  handleFetchProjects();
  fetchApplications();
});
</script>

<template>
  <div class="p-5">
    <div class="mb-4 flex items-center justify-between">
      <h1 class="text-xl font-semibold">应用管理</h1>
      <a-button type="primary" @click="handleCreate">
        <template #icon>
          <PlusOutlined />
        </template>
        新建应用
      </a-button>
    </div>

    <a-form ref="searchFormRef" :model="searchParams" layout="inline" class="mb-4">
      <a-form-item label="Name" name="name">
        <a-input
          v-model:value="searchParams.name"
          placeholder="Enter name"
          allow-clear
          style="width: 200px"
        />
      </a-form-item>
      <a-form-item label="Project" name="project_id">
        <a-select
          v-model:value="searchParams.project_id"
          placeholder="Select project"
          allow-clear
          style="width: 200px"
        >
          <a-select-option
            v-for="option in projectOptions"
            :key="option.value"
            :value="option.value"
          >
            {{ option.label }}
          </a-select-option>
        </a-select>
      </a-form-item>
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
            搜索
          </a-button>
          <a-button @click="resetForm">重置</a-button>
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
          title: 'Project',
          dataIndex: 'project_name',
          key: 'project_name',
        },
        {
          title: 'Owner',
          dataIndex: 'owner',
          key: 'owner',
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
      :data-source="applications"
      :loading="loading"
      :pagination="pagination"
      row-key="id"
      @change="handleTableChange"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'project_name'">
          {{ projectMap[record.project_id] || '-' }}
        </template>
        <template v-else-if="column.key === 'status'">
          <a-tag :color="statusMap[record.status]?.color">
            {{ statusMap[record.status]?.text }}
          </a-tag>
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
              查看
            </a-button>
            <a-button size="small" @click="handleEdit(record)">
              <template #icon>
                <EditOutlined />
              </template>
              编辑
            </a-button>
            <a-button size="small" danger @click="handleDelete(record.id)">
              <template #icon>
                <DeleteOutlined />
              </template>
              删除
            </a-button>
          </a-space>
        </template>
      </template>
    </a-table>

    <a-modal
      v-model:open="modalVisible"
      :title="modalMode === 'create' ? '新建应用' : '编辑应用'"
      :confirm-loading="modalLoading"
      @ok="handleModalOk"
      @cancel="handleModalCancel"
    >
      <a-form
        ref="modalFormRef"
        :model="selectedApplication || {}"
        :label-col="{ span: 6 }"
        :wrapper-col="{ span: 16 }"
        class="mt-4"
      >
        <a-form-item
          v-if="modalMode === 'create'"
          label="Project"
          name="project_id"
          :rules="[{ required: true, message: 'Please select project' }]"
        >
          <a-select
            v-model:value="(selectedApplication as any).project_id"
            placeholder="Select project"
            allow-clear
          >
            <a-select-option
              v-for="option in projectOptions"
              :key="option.value"
              :value="option.value"
            >
              {{ option.label }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item
          label="Name"
          name="name"
          :rules="[{ required: true, message: 'Please enter name' }]"
        >
          <a-input v-model:value="(selectedApplication as any).name" placeholder="Enter name" />
        </a-form-item>
        <a-form-item
          label="Code"
          name="code"
          :rules="[{ required: true, message: 'Please enter code' }]"
        >
          <a-input v-model:value="(selectedApplication as any).code" placeholder="Enter code" />
        </a-form-item>
        <a-form-item label="Description" name="description">
          <a-textarea v-model:value="(selectedApplication as any).description" placeholder="Enter description" :rows="3" />
        </a-form-item>
        <a-form-item label="Owner" name="owner">
          <a-input v-model:value="(selectedApplication as any).owner" placeholder="Enter owner" />
        </a-form-item>
        <a-form-item
          v-if="modalMode === 'edit'"
          label="Status"
          name="status"
          :rules="[{ required: true, message: 'Please select status' }]"
        >
          <a-select
            v-model:value="(selectedApplication as any).status"
            placeholder="Select status"
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
      </a-form>
    </a-modal>

    <a-drawer
      v-model:open="detailDrawerVisible"
      title="Application Details"
      width="600"
      placement="right"
    >
      <a-descriptions v-if="selectedApplication" :column="1" bordered>
        <a-descriptions-item label="ID">
          {{ selectedApplication.id }}
        </a-descriptions-item>
        <a-descriptions-item label="Project">
          {{ projectMap[selectedApplication.project_id] || '-' }}
        </a-descriptions-item>
        <a-descriptions-item label="Name">
          {{ selectedApplication.name }}
        </a-descriptions-item>
        <a-descriptions-item label="Code">
          {{ selectedApplication.code }}
        </a-descriptions-item>
        <a-descriptions-item label="Description">
          {{ selectedApplication.description || '-' }}
        </a-descriptions-item>
        <a-descriptions-item label="Owner">
          {{ selectedApplication.owner || '-' }}
        </a-descriptions-item>
        <a-descriptions-item label="Status">
          <a-tag :color="statusMap[selectedApplication.status]?.color">
            {{ statusMap[selectedApplication.status]?.text }}
          </a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="Created At">
          {{ new Date(selectedApplication.created_at).toLocaleString() }}
        </a-descriptions-item>
        <a-descriptions-item label="Updated At">
          {{ selectedApplication.updated_at ? new Date(selectedApplication.updated_at).toLocaleString() : '-' }}
        </a-descriptions-item>
      </a-descriptions>
    </a-drawer>
  </div>
</template>
