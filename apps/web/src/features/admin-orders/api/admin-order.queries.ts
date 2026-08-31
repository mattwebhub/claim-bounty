import { queryOptions, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useEffect, useRef } from 'react';
import { isApiError } from '@/shared/api';
import {
  createExport,
  downloadExport,
  getAdminOrder,
  getExport,
  listAdminOrders,
  updateAdminIntake,
  type AdminOrderFilters,
} from './admin-order.service';
import type {
  AdminIntake,
  AdminOrder,
  ExportReadinessInput,
  ExportRecord,
} from '../model/admin-order.schema';

export const adminOrderKeys = {
  all: ['admin-orders'] as const,
  list: (filters: AdminOrderFilters) => [...adminOrderKeys.all, 'list', filters] as const,
  details: () => [...adminOrderKeys.all, 'detail'] as const,
  detail: (orderId: string) => [...adminOrderKeys.details(), orderId] as const,
  export: (orderId: string, exportId: string) =>
    [...adminOrderKeys.detail(orderId), 'export', exportId] as const,
};

export function useAdminOrders(filters: AdminOrderFilters) {
  return useQuery({
    queryKey: adminOrderKeys.list(filters),
    queryFn: ({ signal }) => listAdminOrders(filters, signal),
  });
}

export function adminOrderOptions(orderId: string) {
  return queryOptions({
    queryKey: adminOrderKeys.detail(orderId),
    queryFn: ({ signal }) => getAdminOrder(orderId, signal),
    enabled: orderId.length > 0,
  });
}

export function useAdminOrder(orderId: string) {
  return useQuery(adminOrderOptions(orderId));
}

export function useExportStatus(orderId: string, exportId: string | null) {
  return useQuery({
    queryKey: adminOrderKeys.export(orderId, exportId ?? ''),
    queryFn: ({ signal }) => getExport(orderId, exportId ?? '', signal),
    enabled: exportId !== null,
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      return status === 'queued' || status === 'building' ? 2_000 : false;
    },
  });
}

export function useCreateExport(orderId: string, csrfToken: string) {
  const queryClient = useQueryClient();
  const operation = useRef<{ fingerprint: string; key: string } | null>(null);
  return useMutation({
    mutationFn: (input: ExportReadinessInput) => {
      const order = queryClient.getQueryData<AdminOrder>(adminOrderKeys.detail(orderId));
      if (!order) return Promise.reject(new Error('The order is no longer available.'));
      const fingerprint = JSON.stringify({ orderId, input });
      if (operation.current?.fingerprint !== fingerprint) {
        operation.current = { fingerprint, key: crypto.randomUUID() };
      }
      return createExport(orderId, order.version, csrfToken, input, operation.current.key);
    },
    onSuccess: async (exportRecord, input) => {
      if (operation.current?.fingerprint === JSON.stringify({ orderId, input })) {
        operation.current = null;
      }
      queryClient.setQueryData(adminOrderKeys.export(orderId, exportRecord.id), exportRecord);
      await queryClient.invalidateQueries({ queryKey: adminOrderKeys.detail(orderId) });
    },
  });
}

export function useDownloadExport() {
  return useMutation({ mutationFn: (exportRecord: ExportRecord) => downloadExport(exportRecord) });
}

export function useUpdateAdminIntake(orderId: string, csrfToken: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: AdminIntake) => {
      const order = queryClient.getQueryData<AdminOrder>(adminOrderKeys.detail(orderId));
      if (!order) return Promise.reject(new Error('The order is no longer available.'));
      return updateAdminIntake(orderId, order.version, csrfToken, input);
    },
    onSuccess: (order) => queryClient.setQueryData(adminOrderKeys.detail(orderId), order),
  });
}

export function useAdminAccessRevocation(error: unknown, onAccessDenied: () => void) {
  const queryClient = useQueryClient();
  const denied = isApiError(error) && (error.status === 401 || error.status === 403);

  useEffect(() => {
    if (!denied) return;
    queryClient.removeQueries({ queryKey: adminOrderKeys.all });
    onAccessDenied();
  }, [denied, onAccessDenied, queryClient]);

  return denied;
}
