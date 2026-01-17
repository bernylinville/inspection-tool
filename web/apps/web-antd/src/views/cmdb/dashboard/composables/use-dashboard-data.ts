import { onMounted, ref } from 'vue';

import {
  listAlertsApi,
  listApplicationsApi,
  listESClustersApi,
  listHostsApi,
  listInspectionJobsApi,
  listMySQLInstancesApi,
  listNginxInstancesApi,
  listProjectsApi,
  listRedisInstancesApi,
  listTomcatInstancesApi,
} from '#/api/cmdb';
import type { Alert } from '#/api/cmdb/types';

export interface DashboardStats {
  projectCount: number;
  applicationCount: number;
  hostCount: number;
  middlewareCount: number;
  alertCount: number;
  inspectionCount: number;
}

export interface MiddlewareDistribution {
  mysql: number;
  redis: number;
  nginx: number;
  tomcat: number;
  elasticsearch: number;
}

export function useDashboardData() {
  const stats = ref<DashboardStats>({
    projectCount: 0,
    applicationCount: 0,
    hostCount: 0,
    middlewareCount: 0,
    alertCount: 0,
    inspectionCount: 0,
  });
  const middlewareDistribution = ref<MiddlewareDistribution>({
    mysql: 0,
    redis: 0,
    nginx: 0,
    tomcat: 0,
    elasticsearch: 0,
  });
  const recentAlerts = ref<Alert[]>([]);
  const loading = ref(false);
  const error = ref<Error | null>(null);
  const alertServiceUnavailable = ref(false);

  const fetchStats = async () => {
    loading.value = true;
    error.value = null;
    alertServiceUnavailable.value = false;

    try {
      const [
        projectsRes,
        applicationsRes,
        hostsRes,
        mysqlRes,
        redisRes,
        nginxRes,
        tomcatRes,
        esRes,
        inspectionsRes,
      ] = await Promise.all([
        listProjectsApi({ page: 1, page_size: 1 }),
        listApplicationsApi({ page: 1, page_size: 1 }),
        listHostsApi({ page: 1, page_size: 1 }),
        listMySQLInstancesApi({ page: 1, page_size: 1 }),
        listRedisInstancesApi({ page: 1, page_size: 1 }),
        listNginxInstancesApi({ page: 1, page_size: 1 }),
        listTomcatInstancesApi({ page: 1, page_size: 1 }),
        listESClustersApi({ page: 1, page_size: 1 }),
        listInspectionJobsApi({ page: 1, page_size: 1 }),
      ]);

      const mysqlCount = mysqlRes.total ?? 0;
      const redisCount = redisRes.total ?? 0;
      const nginxCount = nginxRes.total ?? 0;
      const tomcatCount = tomcatRes.total ?? 0;
      const elasticsearchCount = esRes.total ?? 0;

      stats.value = {
        projectCount: projectsRes.total ?? 0,
        applicationCount: applicationsRes.total ?? 0,
        hostCount: hostsRes.total ?? 0,
        middlewareCount:
          mysqlCount + redisCount + nginxCount + tomcatCount + elasticsearchCount,
        alertCount: 0,
        inspectionCount: inspectionsRes.total ?? 0,
      };

      middlewareDistribution.value = {
        mysql: mysqlCount,
        redis: redisCount,
        nginx: nginxCount,
        tomcat: tomcatCount,
        elasticsearch: elasticsearchCount,
      };

      try {
        const alertsRes = await listAlertsApi({ page: 1, page_size: 1 });
        stats.value.alertCount = alertsRes.total ?? 0;
        const recentAlertsRes = await listAlertsApi({ page: 1, page_size: 5 });
        recentAlerts.value = recentAlertsRes.items ?? [];
      } catch {
        alertServiceUnavailable.value = true;
        stats.value.alertCount = 0;
        recentAlerts.value = [];
      }
    } catch (err) {
      error.value =
        err instanceof Error
          ? err
          : new Error('Failed to load dashboard data');
    } finally {
      loading.value = false;
    }
  };

  const refresh = () => fetchStats();

  onMounted(() => {
    void fetchStats();
  });

  return {
    stats,
    middlewareDistribution,
    recentAlerts,
    loading,
    error,
    alertServiceUnavailable,
    refresh,
  };
}
