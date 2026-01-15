<script setup lang="ts">
import { ref } from 'vue';
import { Card, Button, Modal } from 'ant-design-vue';
import { PlusOutlined } from '@ant-design/icons-vue';
import RoleTable from './components/role-table.vue';
import RoleSearch from './components/role-search.vue';
import RoleForm from './components/role-form.vue';
import AssignPermissionsModal from './components/assign-permissions-modal.vue';
import { useRoleList } from './composables/use-role-list';
import type { AssignPermissionsRequest, RoleCreateRequest, RoleUpdateRequest } from '#/api/cmdb/types';

const {
  roles,
  total,
  page,
  pageSize,
  loading,
  filters,
  selectedRole,
  formVisible,
  formMode,
  formLoading,
  assignPermissionsVisible,
  assignPermissionsLoading,
  assignPermissionsSelectedRole,
  deleteRole,
  assignPermissions,
  closeDetail,
  openForm,
  closeForm,
  openAssignPermissions,
  closeAssignPermissions,
  applyFilters,
  resetFilters,
  changePage,
  createRole,
  updateRole,
} = useRoleList();

const deleteModalVisible = ref(false);
const roleToDelete = ref<number | null>(null);

function handleCreateClick() {
  openForm('create');
}

function handleEdit(role: any) {
  openForm('edit', role);
}

function handleAssignPermissions(role: any) {
  openAssignPermissions(role);
}

function handleDeleteClick(roleId: number) {
  roleToDelete.value = roleId;
  deleteModalVisible.value = true;
}

async function confirmDelete() {
  if (roleToDelete.value) {
    await deleteRole(roleToDelete.value);
  }
  deleteModalVisible.value = false;
  roleToDelete.value = null;
}

function cancelDelete() {
  deleteModalVisible.value = false;
  roleToDelete.value = null;
}

async function handleFormSubmit(data: RoleCreateRequest | RoleUpdateRequest) {
  let success = false;
  if (formMode.value === 'create') {
    success = await createRole(data as RoleCreateRequest);
  } else if (selectedRole.value) {
    success = await updateRole(selectedRole.value.id, data as RoleUpdateRequest);
  }
  if (success) {
    closeForm();
  }
}

async function handleAssignPermissionsSubmit(roleId: number, data: AssignPermissionsRequest) {
  const success = await assignPermissions(roleId, data);
  if (success) {
    closeAssignPermissions();
  }
}
</script>

<template>
  <div class="p-4">
    <div class="mb-4 flex items-center justify-between">
      <h2 class="text-xl font-semibold">角色管理</h2>
      <Button type="primary" @click="handleCreateClick">
        <template #icon><PlusOutlined /></template>
        创建角色
      </Button>
    </div>

    <Card class="mb-4">
      <RoleSearch @search="applyFilters" @reset="resetFilters" />
    </Card>

    <Card>
      <RoleTable
        :roles="roles"
        :loading="loading"
        :total="total"
        :page="page"
        :page-size="pageSize"
        @page-change="changePage"
        @edit="handleEdit"
        @assign-permissions="handleAssignPermissions"
        @delete="handleDeleteClick"
      />
    </Card>

    <RoleForm
      :visible="formVisible"
      :form-mode="formMode"
      :loading="formLoading"
      :role="selectedRole || undefined"
      @close="closeForm"
      @submit="handleFormSubmit"
    />

    <AssignPermissionsModal
      :visible="assignPermissionsVisible"
      :loading="assignPermissionsLoading"
      :role="assignPermissionsSelectedRole"
      @close="closeAssignPermissions"
      @submit="handleAssignPermissionsSubmit"
    />

    <Modal
      :open="deleteModalVisible"
      title="确认删除"
      @ok="confirmDelete"
      @cancel="cancelDelete"
    >
      <p>确定要删除该角色吗？此操作不可恢复。</p>
    </Modal>
  </div>
</template>
