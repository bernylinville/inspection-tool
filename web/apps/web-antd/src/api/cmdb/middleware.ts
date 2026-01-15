import { requestClient } from '#/api/request';

import type {
  ElasticsearchCluster,
  MySQLInstance,
  NginxInstance,
  PaginatedData,
  PaginationParams,
  RedisInstance,
  TomcatInstance,
} from './types';

export async function listMySQLInstancesApi(params?: PaginationParams) {
  return requestClient.get<PaginatedData<MySQLInstance>>('/mysql', { params });
}

export async function getMySQLInstanceApi(id: number) {
  return requestClient.get<MySQLInstance>(`/mysql/${id}`);
}

export async function deleteMySQLInstanceApi(id: number) {
  return requestClient.delete(`/mysql/${id}`);
}

export async function listRedisInstancesApi(params?: PaginationParams) {
  return requestClient.get<PaginatedData<RedisInstance>>('/redis', { params });
}

export async function getRedisInstanceApi(id: number) {
  return requestClient.get<RedisInstance>(`/redis/${id}`);
}

export async function deleteRedisInstanceApi(id: number) {
  return requestClient.delete(`/redis/${id}`);
}

export async function listNginxInstancesApi(params?: PaginationParams) {
  return requestClient.get<PaginatedData<NginxInstance>>('/nginx', { params });
}

export async function getNginxInstanceApi(id: number) {
  return requestClient.get<NginxInstance>(`/nginx/${id}`);
}

export async function deleteNginxInstanceApi(id: number) {
  return requestClient.delete(`/nginx/${id}`);
}

export async function listTomcatInstancesApi(params?: PaginationParams) {
  return requestClient.get<PaginatedData<TomcatInstance>>('/tomcat', { params });
}

export async function getTomcatInstanceApi(id: number) {
  return requestClient.get<TomcatInstance>(`/tomcat/${id}`);
}

export async function deleteTomcatInstanceApi(id: number) {
  return requestClient.delete(`/tomcat/${id}`);
}

export async function listESClustersApi(params?: PaginationParams) {
  return requestClient.get<PaginatedData<ElasticsearchCluster>>(
    '/elasticsearch',
    { params },
  );
}

export async function getESClusterApi(id: number) {
  return requestClient.get<ElasticsearchCluster>(`/elasticsearch/${id}`);
}

export async function deleteESClusterApi(id: number) {
  return requestClient.delete(`/elasticsearch/${id}`);
}
