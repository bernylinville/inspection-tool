import { requestClient } from '#/api/request';

import type { HealthCheckResponse } from './types';

export async function healthCheckApi() {
  return requestClient.get<HealthCheckResponse>('/health');
}
