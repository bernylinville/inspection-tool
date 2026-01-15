<template>
  <a-modal
    :open="visible"
    title="分配权限"
    :confirm-loading="loading"
    @ok="handleOk"
    @cancel="handleCancel"
  >
    <div v-if="role" class="mb-4">
      <span class="font-semibold">角色：</span>
      <span>{{ role.name }}</span>
    </div>
    <a-form :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
      <a-form-item label="权限" required>
        <a-select
          v-model:value="selectedPermissionIds"
          mode="multiple"
          placeholder="请选择权限"
          :loading="permissionsLoading"
          :options="permissionOptions"
        />
      </a-form-item>
    </a-form>
  </a-modal>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from 'vue';
import { message } from 'ant-design-vue';
import { listPermissionsApi } from '@/api/cmdb/permission';
import type { AssignPermissionsRequest, Permission, Role } from '@/api/cmdb/types';

const props = defineProps<{
  visible: boolean;
  loading: boolean;
  role: Role | null;
}>();

const emit = defineEmits<{
  (event: 'close'): void;
  (event: 'submit', roleId: number, data: AssignPermissionsRequest): void;
}>();

const permissions = ref<Permission[]>([]);
const permissionsLoading = ref(false);
const selectedPermissionIds = ref<number[]>([]);

const permissionOptions = computed(() => {
  return permissions.value.map((permission) => ({
    label: `${permission.name} (${permission.resource}:${permission.action})`,
    value: permission.id,
  }));
});

const fetchPermissions = async () => {
  permissionsLoading.value = true;
  try {
    const result = await listPermissionsApi();
    permissions.value = result.items || [];
  } catch (err) {
    message.error('获取权限列表失败');
  } finally {
    permissionsLoading.value = false;
  }
};

const handleOk = () => {
  if (props.role) {
    emit('submit', props.role.id, { permission_ids: selectedPermissionIds.value });
  }
};

const handleCancel = () => {
  selectedPermissionIds.value = [];
  emit('close');
};

watch(
  () => props.visible,
  (newVisible) => {
    if (newVisible) {
      fetchPermissions();
    }
  },
);

watch(
  () => props.role,
  (newRole) => {
    if (newRole && newRole.permissions) {
      selectedPermissionIds.value = newRole.permissions.map((permission) => permission.id);
    }
  },
  { immediate: true },
);
</script>
