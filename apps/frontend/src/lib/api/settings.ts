import apiClient from '@/lib/api-client';
import type {
  SettingsCleanupResult,
  SettingsStatus,
  SystemSettings,
  UpdateSystemSettingsInput,
} from '@/types/api';

export async function fetchSettings(): Promise<SystemSettings> {
  const { data } = await apiClient.get<SystemSettings>('/v1/settings');
  return data;
}

export async function updateSettings(input: UpdateSystemSettingsInput): Promise<SystemSettings> {
  const { data } = await apiClient.patch<SystemSettings>('/v1/settings', input);
  return data;
}

export async function resetSettings(): Promise<SystemSettings> {
  const { data } = await apiClient.post<SystemSettings>('/v1/settings/reset');
  return data;
}

export async function fetchSettingsStatus(): Promise<SettingsStatus> {
  const { data } = await apiClient.get<SettingsStatus>('/v1/settings/status');
  return data;
}

export async function cleanupSettingsData(): Promise<SettingsCleanupResult> {
  const { data } = await apiClient.post<SettingsCleanupResult>('/v1/settings/cleanup');
  return data;
}
