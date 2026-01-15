import type { Alert, Incident } from '#/api/cmdb/types';
import { ref } from 'vue';
import { defineStore } from 'pinia';
import { listAlertsApi, listIncidentsApi } from '#/api/cmdb';

export const useCmdbAlertStore = defineStore('cmdbAlert', () => {
  const alerts = ref<Alert[]>([]);
  const alertsTotal = ref(0);
  const alertsLoading = ref(false);

  const incidents = ref<Incident[]>([]);
  const incidentsTotal = ref(0);
  const incidentsLoading = ref(false);

  async function fetchAlerts(params?: { page?: number; page_size?: number; start_time?: number; end_time?: number }) {
    alertsLoading.value = true;
    try {
      const data = await listAlertsApi(params);
      alerts.value = data.items;
      alertsTotal.value = data.total;
    } finally {
      alertsLoading.value = false;
    }
  }

  async function fetchIncidents(params?: { page?: number; page_size?: number; start_time?: number; end_time?: number }) {
    incidentsLoading.value = true;
    try {
      const data = await listIncidentsApi(params);
      incidents.value = data.items;
      incidentsTotal.value = data.total;
    } finally {
      incidentsLoading.value = false;
    }
  }

  function $reset() {
    alerts.value = [];
    alertsTotal.value = 0;
    incidents.value = [];
    incidentsTotal.value = 0;
  }

  return {
    alerts, alertsTotal, alertsLoading, fetchAlerts,
    incidents, incidentsTotal, incidentsLoading, fetchIncidents,
    $reset,
  };
});
