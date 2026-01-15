<template>
  <a-modal
    :open="visible"
    :title="formMode === 'create' ? '创建用户' : '编辑用户'"
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
      <a-form-item v-if="formMode === 'create'" label="用户名" name="username">
        <a-input v-model:value="formState.username" placeholder="请输入用户名" />
      </a-form-item>

      <a-form-item v-if="formMode === 'create'" label="密码" name="password">
        <a-input-password v-model:value="formState.password" placeholder="请输入密码" />
      </a-form-item>

      <a-form-item label="邮箱" name="email">
        <a-input v-model:value="formState.email" placeholder="请输入邮箱" />
      </a-form-item>

      <a-form-item label="显示名称" name="display_name">
        <a-input v-model:value="formState.display_name" placeholder="请输入显示名称" />
      </a-form-item>
    </a-form>
  </a-modal>
</template>

<script lang="ts" setup>
import { reactive, ref, watch } from 'vue';
import type { User, UserCreateRequest, UserUpdateRequest } from '#/api/cmdb/types';
import type { FormInstance } from 'ant-design-vue';

const props = defineProps<{
  visible: boolean;
  formMode: 'create' | 'edit';
  loading: boolean;
  user?: User;
}>();

const emit = defineEmits<{
  (event: 'close'): void;
  (event: 'submit', data: UserCreateRequest | UserUpdateRequest): void;
}>();

const formRef = ref<FormInstance>();
const formState = reactive<{
  username?: string;
  password?: string;
  email?: string;
  display_name?: string;
}>({
  username: '',
  password: '',
  email: '',
  display_name: '',
});

const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
};

const handleOk = async () => {
  try {
    await formRef.value?.validate();
    if (props.formMode === 'create') {
      emit('submit', {
        username: formState.username!,
        password: formState.password!,
        email: formState.email,
        display_name: formState.display_name,
      } as UserCreateRequest);
    } else {
      emit('submit', {
        email: formState.email,
        display_name: formState.display_name,
      } as UserUpdateRequest);
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
  () => props.user,
  (newUser) => {
    if (newUser) {
      formState.username = newUser.username;
      formState.email = newUser.email;
      formState.display_name = newUser.display_name;
    }
  },
  { immediate: true },
);
</script>
