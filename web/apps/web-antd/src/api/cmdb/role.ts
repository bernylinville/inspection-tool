import { requestClient } from '#/api/request';

import type {
  AssignPermissionsRequest,
  PaginatedData,
  PaginationParams,
  Role,
  RoleCreateRequest,
  RoleUpdateRequest,
} from './types';

export async function listRolesApi(params?: PaginationParams) {
  return requestClient.get<PaginatedData<Role>>('/roles', { params });
}

export async function createRoleApi(data: RoleCreateRequest) {
  return requestClient.post<Role>('/roles', data);
}

export async function getRoleApi(id: number) {
  return requestClient.get<Role>(`/roles/${id}`);
}

export async function updateRoleApi(id: number, data: RoleUpdateRequest) {
  return requestClient.put<Role>(`/roles/${id}`, data);
}

export async function deleteRoleApi(id: number) {
  return requestClient.delete(`/roles/${id}`);
}

export async function assignRolePermissionsApi(
  id: number,
  data: AssignPermissionsRequest,
) {
  return requestClient.put(`/roles/${id}/permissions`, data);
}
