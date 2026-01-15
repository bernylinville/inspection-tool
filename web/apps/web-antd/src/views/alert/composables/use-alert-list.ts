import { computed, onMounted, ref } from 'vue';
import { listAlertsApi } from '#/api/cmdb/alert';
import type { Alert, AlertStatus } from '#/api/cmdb/types';

export function useAlertList() {
  // List state
  const alerts = ref<Alert[]>([]);
  const total = ref(0);
  const page = ref(1);
  const pageSize = ref(20);
  const loading = ref(false);
  const error = ref<string | null>(null);

  // Filter state
  const filters = ref<{
    status?: AlertStatus;
    startTime?: number;
    endTime?: number;
  }>({});

  // Detail state
  const selectedAlert = ref<Alert | null>(null);
  const detailVisible = ref(false);

  // Computed
  const totalPages = computed(() => Math.ceil(total.value / pageSize.value));

  // Fetch alerts
  async function fetchAlerts() {
    loading.value = true;
    error.value = null;
    try {
      const params = {
        page: page.value,
        page_size: pageSize.value,
        ...(filters.value.status && { status: filters.value.status }),
        ...(filters.value.startTime && { start_time: filters.value.startTime }),
        ...(filters.value.endTime && { end_time: filters.value.endTime }),
      };
      const res = await listAlertsApi(params);
      alerts.value = res.items || [];
      total.value = res.total || 0;
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch alerts';
      alerts.value = [];
      total.value = 0;
    } finally {
      loading.value = false;
    }
  }

  function changePage(newPage: number, newPageSize: number) {
    page.value = newPage;
    pageSize.value = newPageSize;
    fetchAlerts();
  }

  function applyFilters() {
    page.value = 1;
    fetchAlerts();
  }

  function resetFilters() {
    filters.value = {};
    page.value = 1;
    fetchAlerts();
  }

  function viewDetail(alert: Alert) {
    selectedAlert.value = alert;
    detailVisible.value = true;
  }

  function closeDetail() {
    detailVisible.value = false;
    selectedAlert.value = null;
  }

  onMounted(() => {
    fetchAlerts();
  });

  return {
    alerts,
    total,
    page,
    pageSize,
    loading,
    error,
    filters,
    totalPages,
    selectedAlert,
    detailVisible,
    fetchAlerts,
    changePage,
    applyFilters,
    resetFilters,
    viewDetail,
    closeDetail,
  };
}
