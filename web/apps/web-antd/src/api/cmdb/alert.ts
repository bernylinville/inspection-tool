import { requestClient } from '#/api/request';

import type {
  Alert,
  AlertListParams,
  Incident,
  IncidentListParams,
  PaginatedData,
} from './types';

export async function listAlertsApi(params?: AlertListParams): Promise<PaginatedData<Alert>> {
  try {
    return await requestClient.get<PaginatedData<Alert>>('/alerts', { params });
  } catch {
    return { items: [], total: 0, page: params?.page ?? 1, page_size: params?.page_size ?? 20, total_pages: 0 };
  }
}

export async function getAlertApi(id: string) {
  return requestClient.get<Alert>(`/alerts/${id}`);
}

export async function listIncidentsApi(params?: IncidentListParams): Promise<PaginatedData<Incident>> {
  try {
    return await requestClient.get<PaginatedData<Incident>>('/incidents', { params });
  } catch {
    return { items: [], total: 0, page: params?.page ?? 1, page_size: params?.page_size ?? 20, total_pages: 0 };
  }
}

export async function getIncidentApi(id: string) {
  return requestClient.get<Incident>(`/incidents/${id}`);
}

export interface AlertStatisticsData {
  labels: string[];
  critical: number[];
  warning: number[];
}

export async function getAlertStatisticsApi() {
  return requestClient.get<{ code: number; data: AlertStatisticsData; service_unavailable?: boolean }>(
    '/alerts/statistics',
  );
}
