<script setup lang="ts">
import { Card, Alert as AntAlert } from 'ant-design-vue';
import { useAlertList } from './composables/use-alert-list';
import AlertTable from './components/alert-table.vue';
import AlertSearch from './components/alert-search.vue';
import AlertDetail from './components/alert-detail.vue';

const {
  alerts,
  total,
  page,
  pageSize,
  loading,
  error,
  filters,
  selectedAlert,
  detailVisible,
  changePage,
  applyFilters,
  resetFilters,
  viewDetail,
  closeDetail,
} = useAlertList();
</script>

<template>
  <div class="p-4">
    <div class="mb-4">
      <h2 class="text-xl font-semibold">告警列表</h2>
    </div>

    <AntAlert
      v-if="error"
      type="error"
      :message="error"
      closable
      class="mb-4"
    />

    <Card class="mb-4">
      <AlertSearch
        :filters="filters"
        @update:filters="(f) => (filters = f)"
        @search="applyFilters"
        @reset="resetFilters"
      />
    </Card>

    <Card>
      <AlertTable
        :alerts="alerts"
        :loading="loading"
        :total="total"
        :page="page"
        :page-size="pageSize"
        @page-change="changePage"
        @view-detail="viewDetail"
      />
    </Card>

    <AlertDetail
      :visible="detailVisible"
      :alert="selectedAlert"
      @update:visible="(v) => !v && closeDetail()"
    />
  </div>
</template>
