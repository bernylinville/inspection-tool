import type { MySQLInstance, RedisInstance, NginxInstance, TomcatInstance, ElasticsearchCluster } from '#/api/cmdb/types';
import { ref } from 'vue';
import { defineStore } from 'pinia';
import {
  listMySQLInstancesApi,
  listRedisInstancesApi,
  listNginxInstancesApi,
  listTomcatInstancesApi,
  listESClustersApi,
} from '#/api/cmdb';

export const useCmdbMiddlewareStore = defineStore('cmdbMiddleware', () => {
  const mysqlInstances = ref<MySQLInstance[]>([]);
  const mysqlTotal = ref(0);
  const mysqlLoading = ref(false);

  const redisInstances = ref<RedisInstance[]>([]);
  const redisTotal = ref(0);
  const redisLoading = ref(false);

  const nginxInstances = ref<NginxInstance[]>([]);
  const nginxTotal = ref(0);
  const nginxLoading = ref(false);

  const tomcatInstances = ref<TomcatInstance[]>([]);
  const tomcatTotal = ref(0);
  const tomcatLoading = ref(false);

  const esClusters = ref<ElasticsearchCluster[]>([]);
  const esTotal = ref(0);
  const esLoading = ref(false);

  async function fetchMySQLInstances(params?: { page?: number; page_size?: number }) {
    mysqlLoading.value = true;
    try {
      const data = await listMySQLInstancesApi(params);
      mysqlInstances.value = data.items;
      mysqlTotal.value = data.total;
    } finally {
      mysqlLoading.value = false;
    }
  }

  async function fetchRedisInstances(params?: { page?: number; page_size?: number }) {
    redisLoading.value = true;
    try {
      const data = await listRedisInstancesApi(params);
      redisInstances.value = data.items;
      redisTotal.value = data.total;
    } finally {
      redisLoading.value = false;
    }
  }

  async function fetchNginxInstances(params?: { page?: number; page_size?: number }) {
    nginxLoading.value = true;
    try {
      const data = await listNginxInstancesApi(params);
      nginxInstances.value = data.items;
      nginxTotal.value = data.total;
    } finally {
      nginxLoading.value = false;
    }
  }

  async function fetchTomcatInstances(params?: { page?: number; page_size?: number }) {
    tomcatLoading.value = true;
    try {
      const data = await listTomcatInstancesApi(params);
      tomcatInstances.value = data.items;
      tomcatTotal.value = data.total;
    } finally {
      tomcatLoading.value = false;
    }
  }

  async function fetchESClusters(params?: { page?: number; page_size?: number }) {
    esLoading.value = true;
    try {
      const data = await listESClustersApi(params);
      esClusters.value = data.items;
      esTotal.value = data.total;
    } finally {
      esLoading.value = false;
    }
  }

  function $reset() {
    mysqlInstances.value = [];
    redisInstances.value = [];
    nginxInstances.value = [];
    tomcatInstances.value = [];
    esClusters.value = [];
  }

  return {
    mysqlInstances, mysqlTotal, mysqlLoading, fetchMySQLInstances,
    redisInstances, redisTotal, redisLoading, fetchRedisInstances,
    nginxInstances, nginxTotal, nginxLoading, fetchNginxInstances,
    tomcatInstances, tomcatTotal, tomcatLoading, fetchTomcatInstances,
    esClusters, esTotal, esLoading, fetchESClusters,
    $reset,
  };
});
