import { requestClient } from '#/api/request';

import type {
  InspectionJob,
  InspectionJobCreateRequest,
  InspectionJobListParams,
  PaginatedData,
} from './types';

export async function listInspectionJobsApi(params?: InspectionJobListParams) {
  return requestClient.get<PaginatedData<InspectionJob>>(
    '/inspection/jobs',
    { params },
  );
}

export async function createInspectionJobApi(data: InspectionJobCreateRequest) {
  return requestClient.post<InspectionJob>('/inspection/jobs', data);
}

export async function getInspectionJobApi(id: number) {
  return requestClient.get<InspectionJob>(`/inspection/jobs/${id}`);
}

export async function deleteInspectionJobApi(id: number) {
  return requestClient.delete(`/inspection/jobs/${id}`);
}
