<script lang="ts" setup>
import { ref } from 'vue';
import { Modal } from 'ant-design-vue';

import ESDetail from './components/es-detail.vue';
import ESSearch from './components/es-search.vue';
import ESTable from './components/es-table.vue';
import { useESList } from './composables/use-es-list';

const {
  clusters,
  total,
  page,
  pageSize,
  loading,
  selectedCluster,
  detailLoading,
  applyFilters,
  resetFilters,
  changePage,
  deleteCluster,
  getClusterDetail,
  clearSelectedCluster,
} = useESList();

const detailOpen = ref(false);

const handleViewDetail = (cluster: { id: number }) => {
  detailOpen.value = true;
  void getClusterDetail(cluster.id);
};

const handleCloseDetail = () => {
  detailOpen.value = false;
  clearSelectedCluster();
};

const handleDelete = (clusterId: number) => {
  Modal.confirm({
    title: '删除集群',
    content: '确定删除该集群吗？',
    okText: '删除',
    cancelText: '取消',
    onOk: async () => {
      await deleteCluster(clusterId);
    },
  });
};
</script>

<template>
  <div class="flex flex-col gap-4 p-5">
    <div class="flex items-center justify-between">
      <h1 class="text-xl font-semibold">Elasticsearch 集群</h1>
    </div>

    <a-card>
      <ESSearch @search="applyFilters" @reset="resetFilters" />
    </a-card>

    <a-card>
      <ESTable
        :clusters="clusters"
        :loading="loading"
        :total="total"
        :page="page"
        :page-size="pageSize"
        @page-change="changePage"
        @view-detail="handleViewDetail"
        @delete="handleDelete"
      />
    </a-card>

    <ESDetail
      :cluster="selectedCluster"
      :open="detailOpen"
      :loading="detailLoading"
      @close="handleCloseDetail"
    />
  </div>
</template>
