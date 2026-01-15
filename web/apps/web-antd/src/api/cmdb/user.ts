import { requestClient } from '#/api/request';

import type {
  AssignRolesRequest,
  ChangePasswordRequest,
  PaginatedData,
  PaginationParams,
  User,
  UserCreateRequest,
  UserUpdateRequest,
} from './types';

export async function listUsersApi(params?: PaginationParams) {
  return requestClient.get<PaginatedData<User>>('/users', { params });
}

export async function createUserApi(data: UserCreateRequest) {
  return requestClient.post<User>('/users', data);
}

export async function getUserApi(id: number) {
  return requestClient.get<User>(`/users/${id}`);
}

export async function updateUserApi(id: number, data: UserUpdateRequest) {
  return requestClient.put<User>(`/users/${id}`, data);
}

export async function deleteUserApi(id: number) {
  return requestClient.delete(`/users/${id}`);
}

export async function assignUserRolesApi(id: number, data: AssignRolesRequest) {
  return requestClient.put(`/users/${id}/roles`, data);
}

export async function changePasswordApi(id: number, data: ChangePasswordRequest) {
  return requestClient.put(`/users/${id}/password`, data);
}
