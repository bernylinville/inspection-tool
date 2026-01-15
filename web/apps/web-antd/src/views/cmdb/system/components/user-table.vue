<template>
  <div class="flex flex-col gap-4">
    <a-table
      row-key="id"
      :columns="columns"
      :data-source="users"
      :loading="loading"
      :pagination="false"
      class="bg-white"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'status'">
          <a-tag :color="record.status === 'active' ? 'green' : 'default'">
            {{ record.status === 'active' ? '启用' : '禁用' }}
          </a-tag>
        </template>
        <template v-else-if="column.key === 'roles'">
          <a-tag v-for="role in record.roles" :key="role.id">
            {{ role.name }}
          </a-tag>
          <span v-if="!record.roles || record.roles.length === 0">-</span>
        </template>
        <template v-else-if="column.key === 'created_at'">
          <span>{{ formatDate(record.created_at) }}</span>
        </template>
        <template v-else-if="column.key === 'actions'">
          <div class="flex items-center gap-2">
            <a-button type="link" size="small" @click="emit('view-detail', record.id)">查看</a-button>
            <a-button type="link" size="small" @click="emit('edit', record)">编辑</a-button>
            <a-button type="link" size="small" @click="emit('assign-roles', record)">分配角色</a-button>
            <a-button type="link" size="small" danger @click="emit('delete', record.id)">删除</a-button>
          </div>
        </template>
      </template>
    </a-table>

    <div class="flex items-center justify-end">
      <a-pagination
        :current="page"
        :page-size="pageSize"
        :total="total"
        :show-size-changer="true"
        :show-quick-jumper="true"
        :show-total="showTotal"
        @change="handlePageChange"
        @show-size-change="handlePageChange"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { TableColumnType } from 'ant-design-vue';
import type { User } from '#/api/cmdb/types';

const props = defineProps<{
  users: User[];
  loading: boolean;
  total: number;
  page: number;
  pageSize: number;
}>();

const emit = defineEmits<{
  (event: 'page-change', page: number, pageSize: number): void;
  (event: 'view-detail', userId: number): void;
  (event: 'edit', user: User): void;
  (event: 'assign-roles', user: User): void;
  (event: 'delete', userId: number): void;
}>();

const columns: TableColumnType<User>[] = [
  { title: '用户名', dataIndex: 'username', key: 'username' },
  { title: '显示名称', dataIndex: 'display_name', key: 'display_name' },
  { title: '邮箱', dataIndex: 'email', key: 'email' },
  { title: '状态', dataIndex: 'status', key: 'status' },
  { title: '角色', dataIndex: 'roles', key: 'roles' },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at' },
  { title: '操作', key: 'actions', fixed: 'right', width: 220 },
];

const formatDate = (value: string) => {
  if (!value) {
    return '-';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
};

const showTotal = (totalCount: number) => `共 ${totalCount} 条`;

const handlePageChange = (nextPage: number, nextPageSize: number) => {
  emit('page-change', nextPage, nextPageSize);
};
</script>
