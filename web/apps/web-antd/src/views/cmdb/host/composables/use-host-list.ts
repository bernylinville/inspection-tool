import { computed, onMounted, ref } from 'vue';
import { message } from 'ant-design-vue';

import {
  deleteHostApi,
  getHostApi,
  listHostsApi,
  syncHostsApi,
} from '#/api/cmdb';
import type { Host, SyncHostsResult } from '#/api/cmdb/types';

export interface HostFilters {
  status?: 'online' | 'offline';
  business_group?: string;
}

export function useHostList() {
  const hosts = ref<Host[]>([]);
  const total = ref(0);
  const page = ref(1);
  const pageSize = ref(20);
  const loading = ref(false);
  const error = ref<Error | null>(null);
  const filters = ref<HostFilters>({});

  const syncLoading = ref(false);
  const lastSyncResult = ref<SyncHostsResult | null>(null);

  const selectedHost = ref<Host | null>(null);
  const detailLoading = ref(false);

  const totalPages = computed(() => {
    if (pageSize.value <= 0) {
      return 0;
    }
    return Math.ceil(total.value / pageSize.value);
  });

  const hasHosts = computed(() => hosts.value.length > 0);

  const fetchHosts = async () => {
    loading.value = true;
    error.value = null;

    try {
      const result = (await listHostsApi({
        page: page.value,
        page_size: pageSize.value,
        ...filters.value,
      })) as { items: Host[]; total: number };

      hosts.value = Array.isArray(result.items) ? result.items : [];
      total.value = Number.isFinite(result.total) ? result.total : 0;
    } catch (err) {
      error.value = err as Error;
      message.error('获取主机列表失败');
    } finally {
      loading.value = false;
    }
  };

  const refresh = async () => {
    await fetchHosts();
  };

  const changePage = async (nextPage: number, nextPageSize?: number) => {
    page.value = nextPage;
    if (typeof nextPageSize === 'number') {
      pageSize.value = nextPageSize;
    }
    await fetchHosts();
  };

  const applyFilters = async (nextFilters: HostFilters) => {
    filters.value = { ...nextFilters };
    page.value = 1;
    await fetchHosts();
  };

  const resetFilters = async () => {
    filters.value = {};
    page.value = 1;
    await fetchHosts();
  };

  const syncHosts = async () => {
    syncLoading.value = true;
    try {
      const result = (await syncHostsApi()) as SyncHostsResult;
      lastSyncResult.value = result;
      message.success(
        `同步完成（新增 ${result.new_hosts}，更新 ${result.updated_hosts}，失败 ${result.failed_hosts}）`,
      );
      await fetchHosts();
    } catch (err) {
      lastSyncResult.value = null;
      message.error('同步主机失败');
    } finally {
      syncLoading.value = false;
    }
  };

  const deleteHost = async (hostId: number) => {
    try {
      await deleteHostApi(hostId);
      message.success('删除成功');
      await fetchHosts();
    } catch (err) {
      message.error('删除主机失败');
    }
  };

  const getHostDetail = async (hostId: number) => {
    detailLoading.value = true;
    try {
      const result = (await getHostApi(hostId)) as Host;
      selectedHost.value = result ?? null;
    } catch (err) {
      message.error('获取主机详情失败');
    } finally {
      detailLoading.value = false;
    }
  };

  const clearSelectedHost = () => {
    selectedHost.value = null;
  };

  onMounted(() => {
    void fetchHosts();
  });

  return {
    hosts,
    total,
    page,
    pageSize,
    loading,
    error,
    filters,
    syncLoading,
    lastSyncResult,
    selectedHost,
    detailLoading,
    totalPages,
    hasHosts,
    fetchHosts,
    refresh,
    changePage,
    applyFilters,
    resetFilters,
    syncHosts,
    deleteHost,
    getHostDetail,
    clearSelectedHost,
  };
}
