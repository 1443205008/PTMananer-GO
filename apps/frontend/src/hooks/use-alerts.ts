'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  dismissAlert,
  fetchAlerts,
  fetchUnreadCount,
  markAlertRead,
  markAllAlertsRead,
} from '@/lib/api/alerts';
import { queryKeys } from '@/lib/query-keys';

export function useAlerts(unreadOnly = false) {
  return useQuery({
    queryKey: queryKeys.alerts(unreadOnly),
    queryFn: () => fetchAlerts(unreadOnly),
    refetchInterval: 60_000,
  });
}

export function useAlertsUnreadCount() {
  return useQuery({
    queryKey: queryKeys.alertsUnreadCount(),
    queryFn: fetchUnreadCount,
    refetchInterval: 30_000,
  });
}

export function useMarkAlertRead() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: markAlertRead,
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['alerts'] });
    },
  });
}

export function useMarkAllAlertsRead() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: markAllAlertsRead,
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['alerts'] });
    },
  });
}

export function useDismissAlert() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: dismissAlert,
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['alerts'] });
    },
  });
}
