<script lang="ts" setup>
import { ref } from 'vue';
import { Modal } from 'ant-design-vue';

import RedisDetail from './components/redis-detail.vue';
import RedisSearch from './components/redis-search.vue';
import RedisTable from './components/redis-table.vue';
import { useRedisList } from './composables/use-redis-list';

const {
  instances,
  total,
  page,
  pageSize,
  loading,
  selectedInstance,
  detailLoading,
  applyFilters,
  resetFilters,
  changePage,
  deleteInstance,
  getInstanceDetail,
  clearSelectedInstance,
} = useRedisList();

const detailOpen = ref(false);

const handleViewDetail = (instance: { id: number }) => {
  detailOpen.value = true;
  void getInstanceDetail(instance.id);
};

const handleCloseDetail = () => {
  detailOpen.value = false;
  clearSelectedInstance();
};

const handleDelete = (instanceId: number) => {
  Modal.confirm({
    title: '删除实例',
    content: '确定删除该实例吗？',
    okText: '删除',
    cancelText: '取消',
    onOk: async () => {
      await deleteInstance(instanceId);
    },
  });
};
</script>

<template>
  <div class="flex flex-col gap-4 p-5">
    <div class="flex items-center justify-between">
      <h1 class="text-xl font-semibold">Redis 实例</h1>
    </div>

    <a-card>
      <RedisSearch @search="applyFilters" @reset="resetFilters" />
    </a-card>

    <a-card>
      <RedisTable
        :instances="instances"
        :loading="loading"
        :total="total"
        :page="page"
        :page-size="pageSize"
        @page-change="changePage"
        @view-detail="handleViewDetail"
        @delete="handleDelete"
      />
    </a-card>

    <RedisDetail
      :instance="selectedInstance"
      :open="detailOpen"
      :loading="detailLoading"
      @close="handleCloseDetail"
    />
  </div>
</template>
