<script lang="ts" setup>
import { ref } from 'vue';
import { Modal } from 'ant-design-vue';

import TomcatDetail from './components/tomcat-detail.vue';
import TomcatSearch from './components/tomcat-search.vue';
import TomcatTable from './components/tomcat-table.vue';
import { useTomcatList } from './composables/use-tomcat-list';

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
} = useTomcatList();

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
      <h1 class="text-xl font-semibold">Tomcat 实例</h1>
    </div>

    <a-card>
      <TomcatSearch @search="applyFilters" @reset="resetFilters" />
    </a-card>

    <a-card>
      <TomcatTable
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

    <TomcatDetail
      :instance="selectedInstance"
      :open="detailOpen"
      :loading="detailLoading"
      @close="handleCloseDetail"
    />
  </div>
</template>
