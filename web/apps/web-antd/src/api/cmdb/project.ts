import { requestClient } from '#/api/request';

import type {
  Project,
  PaginatedData,
  PaginationParams,
} from './types';

export async function getProjectListApi(params?: PaginationParams) {
  return requestClient.get<PaginatedData<Project>>('/projects', { params });
}
