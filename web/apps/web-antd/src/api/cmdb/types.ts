/**
 * CMDB API TypeScript 类型定义
 * 基于 OpenAPI 3.1 规范自动生成
 * @see cmdb-memory/api/openapi.yaml
 */

// ============ 通用类型 ============

/** API 响应基础结构 */
export interface ApiResponse<T = unknown> {
  code: number;
  message: string;
  data?: T;
}

/** 分页参数 */
export interface PaginationParams {
  page?: number;
  page_size?: number;
}

/** 分页响应 */
export interface PaginatedData<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages?: number;
}

/** 错误响应 */
export interface ErrorResponse {
  code: number;
  message: string;
  details?: string;
}

// ============ 认证相关 ============

/** 登录请求 */
export interface LoginRequest {
  username: string;
  password: string;
}

/** 登录响应数据 */
export interface LoginResponseData {
  access_token: string;
  refresh_token: string;
  expires_at: number;
  token_type: string;
  user?: User;
}

/** Token 刷新请求 */
export interface TokenRefreshRequest {
  refresh_token: string;
}

/** Token 刷新响应数据 */
export interface TokenRefreshResponseData {
  access_token: string;
  refresh_token: string;
  expires_at: number;
}

// ============ 用户相关 ============

/** 用户状态 */
export type UserStatus = 'active' | 'inactive';

/** 用户 */
export interface User {
  id: number;
  username: string;
  email?: string;
  display_name?: string;
  status: UserStatus;
  roles?: Role[];
  last_login_at?: string;
  created_at: string;
  updated_at?: string;
}

/** 创建用户请求 */
export interface UserCreateRequest {
  username: string;
  password: string;
  email?: string;
  display_name?: string;
  role_ids?: number[];
}

/** 更新用户请求 */
export interface UserUpdateRequest {
  email?: string;
  display_name?: string;
  status?: UserStatus;
}

/** 分配角色请求 */
export interface AssignRolesRequest {
  role_ids: number[];
}

/** 修改密码请求 */
export interface ChangePasswordRequest {
  old_password: string;
  new_password: string;
}

// ============ 角色相关 ============

/** 角色 */
export interface Role {
  id: number;
  name: string;
  description?: string;
  permissions?: Permission[];
  created_at: string;
}

/** 创建角色请求 */
export interface RoleCreateRequest {
  name: string;
  description?: string;
  permission_ids?: number[];
}

/** 更新角色请求 */
export interface RoleUpdateRequest {
  name?: string;
  description?: string;
}

/** 分配权限请求 */
export interface AssignPermissionsRequest {
  permission_ids: number[];
}

/** 权限 */
export interface Permission {
  id: number;
  name: string;
  resource: string;
  action: string;
  description?: string;
}

// ============ 项目相关 ============

/** 项目状态 */
export type ProjectStatus = 'active' | 'inactive';

/** 项目 */
export interface Project {
  id: number;
  name: string;
  code: string;
  description?: string;
  owner?: string;
  status: ProjectStatus;
  created_at: string;
  updated_at?: string;
}

/** 创建项目请求 */
export interface ProjectCreateRequest {
  name: string;
  code: string;
  description?: string;
  owner?: string;
}

/** 更新项目请求 */
export interface ProjectUpdateRequest {
  name?: string;
  description?: string;
  owner?: string;
  status?: ProjectStatus;
}

// ============ 应用相关 ============

/** 应用状态 */
export type ApplicationStatus = 'active' | 'inactive';

/** 应用 */
export interface Application {
  id: number;
  project_id: number;
  name: string;
  code: string;
  description?: string;
  owner?: string;
  status: ApplicationStatus;
  created_at: string;
  updated_at?: string;
}

/** 创建应用请求 */
export interface ApplicationCreateRequest {
  project_id: number;
  name: string;
  code: string;
  description?: string;
  owner?: string;
}

/** 更新应用请求 */
export interface ApplicationUpdateRequest {
  name?: string;
  description?: string;
  owner?: string;
  status?: ApplicationStatus;
}

// ============ 主机资产 ============

export type HostStatus = 'active' | 'inactive';

export interface Host {
  id: number;
  ident: string;
  hostname: string;
  ip: string;
  os?: string;
  os_version?: string;
  kernel_version?: string;
  cpu_cores?: number;
  cpu_model?: string;
  memory_total?: number;
  status: HostStatus;
  business_group?: string;
  env?: string;
  application_id?: number;
  tags?: Record<string, string>;
  last_sync_at?: string;
  created_at: string;
}

export interface HostCreateRequest {
  ident: string;
  hostname: string;
  ip: string;
  os?: string;
  business_group?: string;
  env?: string;
}

export interface HostUpdateRequest {
  hostname?: string;
  business_group?: string;
  env?: string;
  status?: HostStatus;
}

export interface HostListParams extends PaginationParams {
  status?: HostStatus;
  business_group?: string;
}

export interface SyncHostsResult {
  total_hosts: number;
  new_hosts: number;
  updated_hosts: number;
  failed_hosts: number;
  duration: string;
}

// ============ 中间件实例 ============

export type InstanceStatus = 'online' | 'offline';
export type MySQLClusterMode = 'mgr' | 'dual-master' | 'master-slave';
export type RedisRole = 'master' | 'slave';
export type ESClusterStatus = 'green' | 'yellow' | 'red';

export interface MySQLInstance {
  id: number;
  address: string;
  ip: string;
  port: number;
  version?: string;
  cluster_mode?: MySQLClusterMode;
  server_id?: string;
  host_id?: number;
  application_id?: number;
  status: InstanceStatus;
  last_sync_at?: string;
  created_at: string;
}

export interface RedisInstance {
  id: number;
  address: string;
  ip: string;
  port: number;
  version?: string;
  cluster_mode?: string;
  role?: RedisRole;
  host_id?: number;
  application_id?: number;
  status: InstanceStatus;
  last_sync_at?: string;
  created_at: string;
}

export interface NginxInstance {
  id: number;
  address: string;
  ip: string;
  port: number;
  version?: string;
  host_id?: number;
  application_id?: number;
  status: InstanceStatus;
  last_sync_at?: string;
  created_at: string;
}

export interface TomcatInstance {
  id: number;
  address: string;
  ip: string;
  port: number;
  version?: string;
  jvm_version?: string;
  host_id?: number;
  application_id?: number;
  status: InstanceStatus;
  last_sync_at?: string;
  created_at: string;
}

export interface ElasticsearchCluster {
  id: number;
  cluster_name: string;
  version?: string;
  node_count?: number;
  status: ESClusterStatus;
  application_id?: number;
  last_sync_at?: string;
  created_at: string;
}

// ============ 监控相关 ============

export interface MonitorQueryParams {
  query: string;
}

export interface MonitorRangeQueryParams {
  query: string;
  start: string;
  end: string;
  step?: string;
  format?: 'raw' | 'echarts';
}

export interface EChartsDataPoint {
  timestamp: number;
  value: number;
}

export interface EChartsSeries {
  metric: Record<string, string>;
  values: EChartsDataPoint[];
}

export interface EChartsResponse {
  series: EChartsSeries[];
}

// ============ 告警相关 ============

export type AlertSeverity = 'critical' | 'warning' | 'info';
export type AlertStatus = 'firing' | 'resolved';

export interface Alert {
  id: string;
  title: string;
  description?: string;
  severity: AlertSeverity;
  status: AlertStatus;
  source?: string;
  created_at: string;
}

export interface AlertListParams extends PaginationParams {
  status?: AlertStatus;
  start_time?: number;
  end_time?: number;
  orderby?: string;
}

export interface Incident {
  id: string;
  title: string;
  status: string;
  severity: string;
  created_at: string;
}

export interface IncidentListParams extends PaginationParams {
  status?: string;
  start_time?: number;
  end_time?: number;
}

// ============ 巡检相关 ============

export type InspectionType =
  | 'host'
  | 'mysql'
  | 'redis'
  | 'nginx'
  | 'tomcat'
  | 'elasticsearch'
  | 'all';

export type InspectionTriggerType = 'manual' | 'api' | 'cron';
export type InspectionStatus = 'pending' | 'running' | 'completed' | 'failed';

export interface InspectionJob {
  id: number;
  type: InspectionType;
  trigger_type: InspectionTriggerType;
  status: InspectionStatus;
  start_time?: string;
  end_time?: string;
  duration_seconds?: number;
  report_excel_path?: string;
  report_html_path?: string;
  summary?: Record<string, unknown>;
  error_message?: string;
  created_by?: string;
  created_at: string;
}

export interface InspectionJobCreateRequest {
  type: InspectionType;
}

export interface InspectionJobListParams extends PaginationParams {
  status?: InspectionStatus;
  type?: InspectionType;
}

// ============ 健康检查 ============

export interface HealthCheckResponse {
  status: string;
  timestamp: string;
  components?: Record<
    string,
    {
      status: string;
      message?: string;
      latency?: string;
    }
  >;
}
