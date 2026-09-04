'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  cleanupSettingsData,
  fetchSettings,
  fetchSettingsStatus,
  resetSettings,
  updateSettings,
} from '@/lib/api/settings';
import { queryKeys } from '@/lib/query-keys';
import type { UpdateSystemSettingsInput } from '@/types/api';

export function useSettings() {
  return useQuery({
    queryKey: queryKeys.settings(),
    queryFn: fetchSettings,
  });
}

export function useSettingsStatus() {
  return useQuery({
    queryKey: queryKeys.settingsStatus(),
    queryFn: fetchSettingsStatus,
    refetchInterval: 60_000,
  });
}

export function useUpdateSettings() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: UpdateSystemSettingsInput) => updateSettings(input),
    onSuccess: (settings) => {
      qc.setQueryData(queryKeys.settings(), settings);
    },
  });
}

export function useResetSettings() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: resetSettings,
    onSuccess: (settings) => {
      qc.setQueryData(queryKeys.settings(), settings);
    },
  });
}

export function useCleanupSettingsData() {
  return useMutation({
    mutationFn: cleanupSettingsData,
  });
}
