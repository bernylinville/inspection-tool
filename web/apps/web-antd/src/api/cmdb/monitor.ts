import { requestClient } from '#/api/request';

import type {
  EChartsResponse,
  MonitorQueryParams,
  MonitorRangeQueryParams,
} from './types';

export async function queryMetricsApi(params: MonitorQueryParams) {
  return requestClient.get('/monitor/query', { params });
}

export async function queryMetricsRangeApi(params: MonitorRangeQueryParams) {
  return requestClient.get<EChartsResponse>('/monitor/query_range', { params });
}
