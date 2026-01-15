import { computed, onMounted, ref } from 'vue';
import { message } from 'ant-design-vue';

import {
  deleteRedisInstanceApi,
  getRedisInstanceApi,
  listRedisInstancesApi,
} from '#/api/cmdb';
import type { RedisInstance } from '#/api/cmdb/types';

export interface RedisFilters {
  status?: 'online' | 'offline';
}

export function useRedisList() {
  const instances = ref<RedisInstance[]>([]);
  const total = ref(0);
  const page = ref(1);
  const pageSize = ref(20);
  const loading = ref(false);
  const error = ref<Error | null>(null);
  const filters = ref<RedisFilters>({});

  const selectedInstance = ref<RedisInstance | null>(null);
  const detailLoading = ref(false);

  const totalPages = computed(() => {
    if (pageSize.value <= 0) {
      return 0;
    }
    return Math.ceil(total.value / pageSize.value);
  });

  const hasInstances = computed(() => instances.value.length > 0);

  const fetchInstances = async () => {
    loading.value = true;
    error.value = null;

    try {
      const result = (await listRedisInstancesApi({
        page: page.value,
        page_size: pageSize.value,
        ...filters.value,
      })) as { items: RedisInstance[]; total: number };

      instances.value = Array.isArray(result.items) ? result.items : [];
      total.value = Number.isFinite(result.total) ? result.total : 0;
    } catch (err) {
      error.value = err as Error;
      message.error('获取 Redis 列表失败');
    } finally {
      loading.value = false;
    }
  };

  const refresh = async () => {
    await fetchInstances();
  };

  const changePage = async (nextPage: number, nextPageSize?: number) => {
    page.value = nextPage;
    if (typeof nextPageSize === 'number') {
      pageSize.value = nextPageSize;
    }
    await fetchInstances();
  };

  const applyFilters = async (nextFilters: RedisFilters) => {
    filters.value = { ...nextFilters };
    page.value = 1;
    await fetchInstances();
  };

  const resetFilters = async () => {
    filters.value = {};
    page.value = 1;
    await fetchInstances();
  };

  const deleteInstance = async (instanceId: number) => {
    try {
      await deleteRedisInstanceApi(instanceId);
      message.success('删除成功');
      await fetchInstances();
    } catch (err) {
      message.error('删除 Redis 实例失败');
    }
  };

  const getInstanceDetail = async (instanceId: number) => {
    detailLoading.value = true;
    try {
      const result = (await getRedisInstanceApi(instanceId)) as RedisInstance;
      selectedInstance.value = result ?? null;
    } catch (err) {
      message.error('获取 Redis 详情失败');
    } finally {
      detailLoading.value = false;
    }
  };

  const clearSelectedInstance = () => {
    selectedInstance.value = null;
  };

  onMounted(() => {
    void fetchInstances();
  });

  return {
    instances,
    total,
    page,
    pageSize,
    loading,
    error,
    filters,
    selectedInstance,
    detailLoading,
    totalPages,
    hasInstances,
    fetchInstances,
    refresh,
    changePage,
    applyFilters,
    resetFilters,
    deleteInstance,
    getInstanceDetail,
    clearSelectedInstance,
  };
}
