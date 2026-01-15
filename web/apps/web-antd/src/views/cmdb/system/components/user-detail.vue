<template>
  <a-modal
    :open="visible"
    title="用户详情"
    :footer="null"
    @cancel="emit('close')"
  >
    <a-descriptions v-if="user" bordered :column="1">
      <a-descriptions-item label="用户名">
        {{ user.username }}
      </a-descriptions-item>
      <a-descriptions-item label="显示名称">
        {{ user.display_name || '-' }}
      </a-descriptions-item>
      <a-descriptions-item label="邮箱">
        {{ user.email || '-' }}
      </a-descriptions-item>
      <a-descriptions-item label="状态">
        <a-tag :color="user.status === 'active' ? 'green' : 'default'">
          {{ user.status === 'active' ? '启用' : '禁用' }}
        </a-tag>
      </a-descriptions-item>
      <a-descriptions-item label="角色">
        <a-tag v-for="role in user.roles" :key="role.id">
          {{ role.name }}
        </a-tag>
        <span v-if="!user.roles || user.roles.length === 0">-</span>
      </a-descriptions-item>
      <a-descriptions-item label="最后登录">
        {{ formatDate(user.last_login_at) }}
      </a-descriptions-item>
      <a-descriptions-item label="创建时间">
        {{ formatDate(user.created_at) }}
      </a-descriptions-item>
      <a-descriptions-item label="更新时间">
        {{ formatDate(user.updated_at) }}
      </a-descriptions-item>
    </a-descriptions>
  </a-modal>
</template>

<script lang="ts" setup>
import type { User } from '@/api/cmdb/types';

const props = defineProps<{
  visible: boolean;
  user: User | null;
}>();

const emit = defineEmits<{
  (event: 'close'): void;
}>();

const formatDate = (value?: string | null) => {
  if (!value) {
    return '-';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
};
</script>
