<script lang="ts" setup>
import { ref } from 'vue';
import { Modal } from 'ant-design-vue';

import HostDetail from './components/host-detail.vue';
import HostSearch from './components/host-search.vue';
import HostTable from './components/host-table.vue';
import SyncButton from './components/sync-button.vue';
import { useHostList } from './composables/use-host-list';

const {
  hosts,
  total,
  page,
  pageSize,
  loading,
  syncLoading,
  selectedHost,
  detailLoading,
  applyFilters,
  resetFilters,
  changePage,
  syncHosts,
  deleteHost,
  getHostDetail,
  clearSelectedHost,
} = useHostList();

const detailOpen = ref(false);

const handleViewDetail = (host: { id: number }) => {
  detailOpen.value = true;
  void getHostDetail(host.id);
};

const handleCloseDetail = () => {
  detailOpen.value = false;
  clearSelectedHost();
};

const handleDelete = (hostId: number) => {
  Modal.confirm({
    title: '删除主机',
    content: '确定删除该主机吗？',
    okText: '删除',
    cancelText: '取消',
    onOk: async () => {
      await deleteHost(hostId);
    },
  });
};
</script>

<template>
  <div class="flex flex-col gap-4 p-5">
    <div class="flex items-center justify-between">
      <h1 class="text-xl font-semibold">主机管理</h1>
      <SyncButton :loading="syncLoading" @sync="syncHosts" />
    </div>

    <a-card>
      <HostSearch @search="applyFilters" @reset="resetFilters" />
    </a-card>

    <a-card>
      <HostTable
        :hosts="hosts"
        :loading="loading"
        :total="total"
        :page="page"
        :page-size="pageSize"
        @page-change="changePage"
        @view-detail="handleViewDetail"
        @delete="handleDelete"
      />
    </a-card>

    <HostDetail
      :host="selectedHost"
      :open="detailOpen"
      :loading="detailLoading"
      @close="handleCloseDetail"
    />
  </div>
</template>
