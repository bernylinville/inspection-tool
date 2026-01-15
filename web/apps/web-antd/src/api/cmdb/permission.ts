import { requestClient } from '#/api/request';

import type { PaginatedData, PaginationParams, Permission } from './types';

export async function listPermissionsApi(params?: PaginationParams) {
  return requestClient.get<PaginatedData<Permission>>('/permissions', { params });
}
