import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useRef } from 'react';
import {
  createOrder,
  getOrder,
  removeOrderFile,
  sha256,
  submitOrder,
  uploadOrderFile,
} from './order.service';
import type { FileRole, Order, OrderIntakeInput } from '../model/order.schema';

export const orderKeys = {
  all: ['claim-orders'] as const,
  detail: (orderId: string) => [...orderKeys.all, orderId] as const,
};

export function useOrder(orderId: string | null) {
  return useQuery({
    queryKey: orderKeys.detail(orderId ?? ''),
    queryFn: ({ signal }) => getOrder(orderId ?? '', signal),
    enabled: orderId !== null,
    placeholderData: (previousOrder) => previousOrder,
    staleTime: 0,
  });
}

export function useCreateOrder() {
  const queryClient = useQueryClient();
  const operation = useRef<{ fingerprint: string; key: string } | null>(null);
  return useMutation({
    mutationFn: ({ input, csrfToken }: { input: OrderIntakeInput; csrfToken: string }) => {
      const fingerprint = JSON.stringify(input);
      if (operation.current?.fingerprint !== fingerprint) {
        operation.current = { fingerprint, key: crypto.randomUUID() };
      }
      return createOrder(input, csrfToken, operation.current.key);
    },
    onSuccess: (order, { input }) => {
      if (operation.current?.fingerprint === JSON.stringify(input)) operation.current = null;
      queryClient.setQueryData(orderKeys.detail(order.id), order);
    },
  });
}

interface UploadInput {
  csrfToken: string;
  file: File;
  idempotencyKey: string;
  onHashing: () => void;
  onProgress: (percentage: number) => void;
  orderId: string;
  role: FileRole;
  signal: AbortSignal;
}

export function useUploadOrderFile() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: UploadInput) => {
      const order = queryClient.getQueryData<Order>(orderKeys.detail(input.orderId));
      if (!order) throw new Error('The draft order is no longer available.');
      input.onHashing();
      const digest = await sha256(input.file);
      if (input.signal.aborted) throw new DOMException('The upload was cancelled.', 'AbortError');
      const result = await uploadOrderFile({
        csrfToken: input.csrfToken,
        digest,
        file: input.file,
        idempotencyKey: input.idempotencyKey,
        onProgress: input.onProgress,
        orderId: input.orderId,
        role: input.role,
        signal: input.signal,
        version: order.version,
      });
      const current = queryClient.getQueryData<Order>(orderKeys.detail(input.orderId));
      if (current) {
        queryClient.setQueryData(orderKeys.detail(input.orderId), {
          ...current,
          version: result.version,
          status: 'uploading',
          files: current.files.some(({ id }) => id === result.file.id)
            ? current.files.map((file) => (file.id === result.file.id ? result.file : file))
            : [...current.files, result.file],
        });
      }
      return result.file;
    },
  });
}

export function useSubmitOrder() {
  const queryClient = useQueryClient();
  const operation = useRef<{ fingerprint: string; key: string } | null>(null);
  return useMutation({
    mutationFn: ({
      orderId,
      csrfToken,
      termsAccepted,
      uploadsAuthorized,
      analysisUseAuthorized,
    }: {
      orderId: string;
      csrfToken: string;
      termsAccepted: boolean;
      uploadsAuthorized: boolean;
      analysisUseAuthorized: boolean;
    }) => {
      const order = queryClient.getQueryData<Order>(orderKeys.detail(orderId));
      if (!order) return Promise.reject(new Error('The draft order is no longer available.'));
      const authorization = {
        termsAccepted,
        uploadsAuthorized,
        analysisUseAuthorized,
      };
      const fingerprint = JSON.stringify({ orderId, authorization });
      if (operation.current?.fingerprint !== fingerprint) {
        operation.current = { fingerprint, key: crypto.randomUUID() };
      }
      return submitOrder(orderId, order.version, csrfToken, operation.current.key, authorization);
    },
    onSuccess: (order, { orderId, termsAccepted, uploadsAuthorized, analysisUseAuthorized }) => {
      const fingerprint = JSON.stringify({
        orderId,
        authorization: { termsAccepted, uploadsAuthorized, analysisUseAuthorized },
      });
      if (operation.current?.fingerprint === fingerprint) operation.current = null;
      queryClient.setQueryData(orderKeys.detail(order.id), order);
    },
  });
}

export function useRemoveOrderFile() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      csrfToken,
      fileId,
      orderId,
    }: {
      csrfToken: string;
      fileId: string;
      orderId: string;
    }) => {
      const order = queryClient.getQueryData<Order>(orderKeys.detail(orderId));
      if (!order) throw new Error('The draft order is no longer available.');
      const version = await removeOrderFile(orderId, fileId, order.version, csrfToken);
      queryClient.setQueryData(orderKeys.detail(orderId), {
        ...order,
        version,
        files: order.files.filter(({ id }) => id !== fileId),
      });
    },
  });
}
