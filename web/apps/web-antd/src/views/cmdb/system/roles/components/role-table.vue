<template>
  <div class="flex flex-col gap-4">
    <a-table
      row-key="id"
      :columns="columns"
      :data-source="roles"
      :loading="loading"
      :pagination="false"
      class="bg-white"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'permissions'">
          <a-tag v-for="permission in record.permissions" :key="permission.id">
            {{ permission.name }}
          </a-tag>
          <span v-if="!record.permissions || record.permissions.length === 0">-</span>
        </template>
        <template v-else-if="column.key === 'created_at'">
          <span>{{ formatDate(record.created_at) }}</span>
        </template>
        <template v-else-if="column.key === 'actions'">
          <div class="flex items-center gap-2">
            <a-button type="link" size="small" @click="emit('edit', record)">编辑</a-button>
            <a-button type="link" size="small" @click="emit('assign-permissions', record)">分配权限</a-button>
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
import type { Role } from '#/api/cmdb/types';

const props = defineProps<{
  roles: Role[];
  loading: boolean;
  total: number;
  page: number;
  pageSize: number;
}>();

const emit = defineEmits<{
  (event: 'page-change', page: number, pageSize: number): void;
  (event: 'edit', role: Role): void;
  (event: 'assign-permissions', role: Role): void;
  (event: 'delete', roleId: number): void;
}>();

const columns: TableColumnType<Role>[] = [
  { title: '角色名称', dataIndex: 'name', key: 'name' },
  { title: '描述', dataIndex: 'description', key: 'description' },
  { title: '权限', dataIndex: 'permissions', key: 'permissions' },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at' },
  { title: '操作', key: 'actions', fixed: 'right', width: 200 },
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
