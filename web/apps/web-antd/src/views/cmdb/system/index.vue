<script setup lang="ts">
import { ref } from 'vue';
import { Card, Button, Modal } from 'ant-design-vue';
import { PlusOutlined } from '@ant-design/icons-vue';
import UserTable from './components/user-table.vue';
import UserSearch from './components/user-search.vue';
import UserDetail from './components/user-detail.vue';
import UserForm from './components/user-form.vue';
import AssignRolesModal from './components/assign-roles-modal.vue';
import { useUserList } from './composables/use-user-list';
import type { AssignRolesRequest, UserCreateRequest, UserUpdateRequest } from '@/api/cmdb/types';

const {
  users,
  total,
  page,
  pageSize,
  loading,
  filters,
  selectedUser,
  detailVisible,
  formVisible,
  formMode,
  formLoading,
  assignRolesVisible,
  assignRolesLoading,
  assignRolesSelectedUser,
  deleteUser,
  assignRoles,
  viewDetail,
  closeDetail,
  openForm,
  closeForm,
  openAssignRoles,
  closeAssignRoles,
  applyFilters,
  resetFilters,
  changePage,
  createUser,
  updateUser,
} = useUserList();

const deleteModalVisible = ref(false);
const userToDelete = ref<number | null>(null);

function handleCreateClick() {
  openForm('create');
}

function handleEdit(user: any) {
  openForm('edit', user);
}

function handleAssignRoles(user: any) {
  openAssignRoles(user);
}

function handleDeleteClick(userId: number) {
  userToDelete.value = userId;
  deleteModalVisible.value = true;
}

async function confirmDelete() {
  if (userToDelete.value) {
    await deleteUser(userToDelete.value);
  }
  deleteModalVisible.value = false;
  userToDelete.value = null;
}

function cancelDelete() {
  deleteModalVisible.value = false;
  userToDelete.value = null;
}

async function handleFormSubmit(data: UserCreateRequest | UserUpdateRequest) {
  let success = false;
  if (formMode.value === 'create') {
    success = await createUser(data as UserCreateRequest);
  } else if (selectedUser.value) {
    success = await updateUser(selectedUser.value.id, data as UserUpdateRequest);
  }
  if (success) {
    closeForm();
  }
}

async function handleAssignRolesSubmit(userId: number, data: AssignRolesRequest) {
  const success = await assignRoles(userId, data);
  if (success) {
    closeAssignRoles();
  }
}
</script>

<template>
  <div class="p-4">
    <div class="mb-4 flex items-center justify-between">
      <h2 class="text-xl font-semibold">用户管理</h2>
      <Button type="primary" @click="handleCreateClick">
        <template #icon><PlusOutlined /></template>
        创建用户
      </Button>
    </div>

    <Card class="mb-4">
      <UserSearch @search="applyFilters" @reset="resetFilters" />
    </Card>

    <Card>
      <UserTable
        :users="users"
        :loading="loading"
        :total="total"
        :page="page"
        :page-size="pageSize"
        @page-change="changePage"
        @view-detail="viewDetail"
        @edit="handleEdit"
        @assign-roles="handleAssignRoles"
        @delete="handleDeleteClick"
      />
    </Card>

    <UserDetail
      :visible="detailVisible"
      :user="selectedUser"
      @close="closeDetail"
    />

    <UserForm
      :visible="formVisible"
      :form-mode="formMode"
      :loading="formLoading"
      :user="selectedUser || undefined"
      @close="closeForm"
      @submit="handleFormSubmit"
    />

    <AssignRolesModal
      :visible="assignRolesVisible"
      :loading="assignRolesLoading"
      :user="assignRolesSelectedUser"
      @close="closeAssignRoles"
      @submit="handleAssignRolesSubmit"
    />

    <Modal
      :open="deleteModalVisible"
      title="确认删除"
      @ok="confirmDelete"
      @cancel="cancelDelete"
    >
      <p>确定要删除该用户吗？此操作不可恢复。</p>
    </Modal>
  </div>
</template>
