<template>
  <a-modal
    :open="visible"
    title="分配角色"
    :confirm-loading="loading"
    @ok="handleOk"
    @cancel="handleCancel"
  >
    <div v-if="user" class="mb-4">
      <span class="font-semibold">用户：</span>
      <span>{{ user.display_name || user.username }}</span>
    </div>
    <a-form :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
      <a-form-item label="角色" required>
        <a-select
          v-model:value="selectedRoleIds"
          mode="multiple"
          placeholder="请选择角色"
          :loading="rolesLoading"
          :options="roleOptions"
        />
      </a-form-item>
    </a-form>
  </a-modal>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from 'vue';
import { message } from 'ant-design-vue';
import { listRolesApi } from '#/api/cmdb/role';
import type { AssignRolesRequest, Role, User } from '#/api/cmdb/types';

const props = defineProps<{
  visible: boolean;
  loading: boolean;
  user: User | null;
}>();

const emit = defineEmits<{
  (event: 'close'): void;
  (event: 'submit', userId: number, data: AssignRolesRequest): void;
}>();

const roles = ref<Role[]>([]);
const rolesLoading = ref(false);
const selectedRoleIds = ref<number[]>([]);

const roleOptions = computed(() => {
  return roles.value.map((role) => ({
    label: role.name,
    value: role.id,
  }));
});

const fetchRoles = async () => {
  rolesLoading.value = true;
  try {
    const result = await listRolesApi();
    roles.value = result.items || [];
  } catch (err) {
    message.error('获取角色列表失败');
  } finally {
    rolesLoading.value = false;
  }
};

const handleOk = () => {
  if (props.user) {
    emit('submit', props.user.id, { role_ids: selectedRoleIds.value });
  }
};

const handleCancel = () => {
  selectedRoleIds.value = [];
  emit('close');
};

watch(
  () => props.visible,
  (newVisible) => {
    if (newVisible) {
      fetchRoles();
    }
  },
);

watch(
  () => props.user,
  (newUser) => {
    if (newUser && newUser.roles) {
      selectedRoleIds.value = newUser.roles.map((role) => role.id);
    }
  },
  { immediate: true },
);
</script>
