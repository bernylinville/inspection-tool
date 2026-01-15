import { requestClient } from '#/api/request';

import type {
  LoginRequest,
  LoginResponseData,
  TokenRefreshRequest,
  TokenRefreshResponseData,
} from './types';

export async function cmdbLoginApi(data: LoginRequest) {
  return requestClient.post<LoginResponseData>('/auth/login', data);
}

export async function cmdbLogoutApi() {
  return requestClient.post('/auth/logout');
}

export async function cmdbRefreshTokenApi(data: TokenRefreshRequest) {
  return requestClient.post<TokenRefreshResponseData>('/auth/refresh', data);
}
