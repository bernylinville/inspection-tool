import { requestClient } from '#/api/request';

import type {
  Alert,
  AlertListParams,
  Incident,
  IncidentListParams,
  PaginatedData,
} from './types';

export async function listAlertsApi(params?: AlertListParams) {
  return requestClient.get<PaginatedData<Alert>>('/alerts', { params });
}

export async function getAlertApi(id: string) {
  return requestClient.get<Alert>(`/alerts/${id}`);
}

export async function listIncidentsApi(params?: IncidentListParams) {
  return requestClient.get<PaginatedData<Incident>>('/incidents', { params });
}

export async function getIncidentApi(id: string) {
  return requestClient.get<Incident>(`/incidents/${id}`);
}
