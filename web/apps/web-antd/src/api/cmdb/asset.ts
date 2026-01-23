import { requestClient } from '#/api/request';

import type {
  Application,
  ApplicationCreateRequest,
  ApplicationUpdateRequest,
  Host,
  HostCreateRequest,
  HostListParams,
  HostUpdateRequest,
  PaginatedData,
  PaginationParams,
  Project,
  ProjectCreateRequest,
  ProjectUpdateRequest,
  SyncHostsResult,
  SyncProjectsResult,
} from './types';

export async function listProjectsApi(params?: PaginationParams) {
  return requestClient.get<PaginatedData<Project>>('/projects', { params });
}

export async function createProjectApi(data: ProjectCreateRequest) {
  return requestClient.post<Project>('/projects', data);
}

export async function getProjectApi(id: number) {
  return requestClient.get<Project>(`/projects/${id}`);
}

export async function updateProjectApi(id: number, data: ProjectUpdateRequest) {
  return requestClient.put<Project>(`/projects/${id}`, data);
}

export async function deleteProjectApi(id: number) {
  return requestClient.delete(`/projects/${id}`);
}

export async function syncProjectsApi() {
  return requestClient.post<SyncProjectsResult>('/projects/sync');
}

export async function listApplicationsApi(params?: PaginationParams) {
  return requestClient.get<PaginatedData<Application>>('/applications', { params });
}

export async function createApplicationApi(data: ApplicationCreateRequest) {
  return requestClient.post<Application>('/applications', data);
}

export async function getApplicationApi(id: number) {
  return requestClient.get<Application>(`/applications/${id}`);
}

export async function updateApplicationApi(id: number, data: ApplicationUpdateRequest) {
  return requestClient.put<Application>(`/applications/${id}`, data);
}

export async function deleteApplicationApi(id: number) {
  return requestClient.delete(`/applications/${id}`);
}

export async function listHostsApi(params?: HostListParams) {
  return requestClient.get<PaginatedData<Host>>('/hosts', { params });
}

export async function createHostApi(data: HostCreateRequest) {
  return requestClient.post<Host>('/hosts', data);
}

export async function getHostApi(id: number) {
  return requestClient.get<Host>(`/hosts/${id}`);
}

export async function updateHostApi(id: number, data: HostUpdateRequest) {
  return requestClient.put<Host>(`/hosts/${id}`, data);
}

export async function deleteHostApi(id: number) {
  return requestClient.delete(`/hosts/${id}`);
}

export async function syncHostsApi() {
  return requestClient.post<SyncHostsResult>('/hosts/sync');
}
