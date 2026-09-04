import apiClient from '@/lib/api-client';
import type { AuthUser } from '@/types/api';

export interface LoginInput {
  email: string;
  password: string;
}

export interface LoginResponse {
  user: AuthUser;
}

export async function login(input: LoginInput): Promise<LoginResponse> {
  const { data } = await apiClient.post<LoginResponse>('/v1/auth/login', input);
  return data;
}

export async function fetchCurrentUser(): Promise<AuthUser> {
  const { data } = await apiClient.get<AuthUser>('/v1/auth/me');
  return data;
}

export async function logout(): Promise<void> {
  await apiClient.post('/v1/auth/logout');
}
