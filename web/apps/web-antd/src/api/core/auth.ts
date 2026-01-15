import { baseRequestClient, requestClient } from '#/api/request';

export namespace AuthApi {
  /** 登录接口参数 */
  export interface LoginParams {
    password?: string;
    username?: string;
  }

  /** 登录接口返回值 */
  export interface LoginResult {
    accessToken: string;
  }

  export interface RefreshTokenResult {
    data: string;
    status: number;
  }
}

/**
 * 登录
 */
/** CMDB 后端登录响应格式 (snake_case) */
interface CmdbLoginResponse {
  access_token: string;
  refresh_token: string;
  expires_at: number;
  token_type: string;
}

/**
 * 登录
 * 转换 CMDB 后端响应格式 (snake_case) 为前端期望格式 (camelCase)
 */
export async function loginApi(data: AuthApi.LoginParams) {
  const response = await requestClient.post<CmdbLoginResponse>(
    '/auth/login',
    data,
  );

  // Transform snake_case to camelCase
  return {
    accessToken: response.access_token,
    refreshToken: response.refresh_token,
    expiresAt: response.expires_at,
  };
}

/**
 * 刷新accessToken
 */
export async function refreshTokenApi(refreshToken?: string) {
  if (refreshToken) {
    return baseRequestClient.post<AuthApi.RefreshTokenResult>('/auth/refresh', {
      refresh_token: refreshToken,
    });
  }
  return baseRequestClient.post<AuthApi.RefreshTokenResult>('/auth/refresh', {
    withCredentials: true,
  });
}

/**
 * 退出登录
 */
export async function logoutApi() {
  return baseRequestClient.post('/auth/logout', {
    withCredentials: true,
  });
}

/**
 * 获取用户权限码
 */
export async function getAccessCodesApi() {
  return requestClient.get<string[]>('/auth/codes');
}
