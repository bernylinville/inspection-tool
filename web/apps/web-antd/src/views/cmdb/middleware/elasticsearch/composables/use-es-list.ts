import { computed, onMounted, ref } from 'vue';
import { message } from 'ant-design-vue';

import {
  deleteESClusterApi,
  getESClusterApi,
  listESClustersApi,
} from '#/api/cmdb';
import type { ElasticsearchCluster } from '#/api/cmdb/types';

export interface ESFilters {
  status?: 'green' | 'yellow' | 'red';
}

export function useESList() {
  const clusters = ref<ElasticsearchCluster[]>([]);
  const total = ref(0);
  const page = ref(1);
  const pageSize = ref(20);
  const loading = ref(false);
  const error = ref<Error | null>(null);
  const filters = ref<ESFilters>({});

  const selectedCluster = ref<ElasticsearchCluster | null>(null);
  const detailLoading = ref(false);

  const totalPages = computed(() => {
    if (pageSize.value <= 0) {
      return 0;
    }
    return Math.ceil(total.value / pageSize.value);
  });

  const hasClusters = computed(() => clusters.value.length > 0);

  const fetchClusters = async () => {
    loading.value = true;
    error.value = null;

    try {
      const result = (await listESClustersApi({
        page: page.value,
        page_size: pageSize.value,
        ...filters.value,
      })) as { items: ElasticsearchCluster[]; total: number };

      clusters.value = Array.isArray(result.items) ? result.items : [];
      total.value = Number.isFinite(result.total) ? result.total : 0;
    } catch (err) {
      error.value = err as Error;
      message.error('获取 Elasticsearch 集群列表失败');
    } finally {
      loading.value = false;
    }
  };

  const refresh = async () => {
    await fetchClusters();
  };

  const changePage = async (nextPage: number, nextPageSize?: number) => {
    page.value = nextPage;
    if (typeof nextPageSize === 'number') {
      pageSize.value = nextPageSize;
    }
    await fetchClusters();
  };

  const applyFilters = async (nextFilters: ESFilters) => {
    filters.value = { ...nextFilters };
    page.value = 1;
    await fetchClusters();
  };

  const resetFilters = async () => {
    filters.value = {};
    page.value = 1;
    await fetchClusters();
  };

  const deleteCluster = async (clusterId: number) => {
    try {
      await deleteESClusterApi(clusterId);
      message.success('删除成功');
      await fetchClusters();
    } catch (err) {
      message.error('删除 Elasticsearch 集群失败');
    }
  };

  const getClusterDetail = async (clusterId: number) => {
    detailLoading.value = true;
    try {
      const result = (await getESClusterApi(clusterId)) as ElasticsearchCluster;
      selectedCluster.value = result ?? null;
    } catch (err) {
      message.error('获取 Elasticsearch 集群详情失败');
    } finally {
      detailLoading.value = false;
    }
  };

  const clearSelectedCluster = () => {
    selectedCluster.value = null;
  };

  onMounted(() => {
    void fetchClusters();
  });

  return {
    clusters,
    total,
    page,
    pageSize,
    loading,
    error,
    filters,
    selectedCluster,
    detailLoading,
    totalPages,
    hasClusters,
    fetchClusters,
    refresh,
    changePage,
    applyFilters,
    resetFilters,
    deleteCluster,
    getClusterDetail,
    clearSelectedCluster,
  };
}
