<template>
  <a-modal
    :open="visible"
    :title="formMode === 'create' ? '创建角色' : '编辑角色'"
    :confirm-loading="loading"
    @ok="handleOk"
    @cancel="handleCancel"
  >
    <a-form
      ref="formRef"
      :model="formState"
      :rules="rules"
      :label-col="{ span: 6 }"
      :wrapper-col="{ span: 16 }"
    >
      <a-form-item label="角色名称" name="name">
        <a-input v-model:value="formState.name" placeholder="请输入角色名称" />
      </a-form-item>

      <a-form-item label="描述" name="description">
        <a-textarea v-model:value="formState.description" placeholder="请输入描述" :rows="4" />
      </a-form-item>
    </a-form>
  </a-modal>
</template>

<script lang="ts" setup>
import { reactive, ref, watch } from 'vue';
import type { Role, RoleCreateRequest, RoleUpdateRequest } from '@/api/cmdb/types';
import type { FormInstance } from 'ant-design-vue';

const props = defineProps<{
  visible: boolean;
  formMode: 'create' | 'edit';
  loading: boolean;
  role?: Role;
}>();

const emit = defineEmits<{
  (event: 'close'): void;
  (event: 'submit', data: RoleCreateRequest | RoleUpdateRequest): void;
}>();

const formRef = ref<FormInstance>();
const formState = reactive<{
  name?: string;
  description?: string;
}>({
  name: '',
  description: '',
});

const rules = {
  name: [{ required: true, message: '请输入角色名称', trigger: 'blur' }],
};

const handleOk = async () => {
  try {
    await formRef.value?.validate();
    if (props.formMode === 'create') {
      emit('submit', {
        name: formState.name!,
        description: formState.description,
      } as RoleCreateRequest);
    } else {
      emit('submit', {
        name: formState.name,
        description: formState.description,
      } as RoleUpdateRequest);
    }
  } catch (error) {
    console.error('Validation failed:', error);
  }
};

const handleCancel = () => {
  formRef.value?.resetFields();
  emit('close');
};

watch(
  () => props.visible,
  (newVisible) => {
    if (!newVisible) {
      formRef.value?.resetFields();
    }
  },
);

watch(
  () => props.role,
  (newRole) => {
    if (newRole) {
      formState.name = newRole.name;
      formState.description = newRole.description;
    }
  },
  { immediate: true },
);
</script>
