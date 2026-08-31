import { http, HttpResponse } from 'msw';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { z } from 'zod';
import { ApiError, apiRequest, downloadApiFile, uploadApiMultipart } from '@/shared/api';
import { mockServer } from '@/shared/test/mocks/server';

describe('apiRequest', () => {
  it('never sends browser credentials to a cross-origin API by default', async () => {
    let credentials: RequestCredentials | undefined;
    mockServer.use(
      http.get('http://127.0.0.1:8080/api/v1/credential-policy', ({ request }) => {
        credentials = request.credentials;
        return HttpResponse.json({ data: { ok: true } });
      }),
    );

    await apiRequest({ path: '/credential-policy', schema: z.object({ ok: z.boolean() }) });

    expect(credentials).toBe('same-origin');
  });

  it('unwraps and validates a successful response', async () => {
    mockServer.use(
      http.get('http://127.0.0.1:8080/api/v1/example', () =>
        HttpResponse.json({ data: { id: 'example-1' } }),
      ),
    );

    await expect(
      apiRequest({ path: '/example', schema: z.object({ id: z.string() }) }),
    ).resolves.toEqual({ id: 'example-1' });
  });

  it('normalizes a structured API failure', async () => {
    mockServer.use(
      http.get('http://127.0.0.1:8080/api/v1/example', () =>
        HttpResponse.json(
          {
            error: {
              code: 'example_missing',
              message: 'Example not found.',
              requestId: 'req-1',
              details: [{ path: 'name', code: 'required', message: 'Name is required.' }],
            },
          },
          { status: 404 },
        ),
      ),
    );

    const error = await apiRequest({ path: '/example', schema: z.unknown() }).catch(
      (cause: unknown) => cause,
    );

    expect(error).toBeInstanceOf(ApiError);
    expect(error).toMatchObject({
      kind: 'http',
      status: 404,
      code: 'example_missing',
      requestId: 'req-1',
      retryable: false,
      fieldIssues: [{ path: 'name', code: 'required', message: 'Name is required.' }],
    });
  });

  it('rejects a success payload that violates the runtime schema', async () => {
    mockServer.use(
      http.get('http://127.0.0.1:8080/api/v1/example', () =>
        HttpResponse.json({ data: { id: 42 } }),
      ),
    );

    await expect(
      apiRequest({ path: '/example', schema: z.object({ id: z.string() }) }),
    ).rejects.toMatchObject({ kind: 'decode', retryable: false });
  });
});

describe('downloadApiFile', () => {
  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it('does not fetch or buffer when direct-to-disk streaming is unavailable', async () => {
    const fetchMock = vi.fn();
    vi.stubGlobal('fetch', fetchMock);

    await expect(
      downloadApiFile({ filename: 'export.zip', path: '/exports/one' }),
    ).rejects.toMatchObject({
      message: 'Direct-to-disk downloads are unavailable in this browser.',
      retryable: false,
    });

    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('validates response headers before streaming the body to a selected file', async () => {
    const events: string[] = [];
    const writable = new WritableStream<Uint8Array>({
      write(chunk) {
        events.push(`write:${Array.from(chunk).join(',')}`);
      },
    });
    const createWritable = vi.fn().mockResolvedValue(writable);
    const showSaveFilePicker = vi.fn().mockResolvedValue({ createWritable });
    vi.stubGlobal('showSaveFilePicker', showSaveFilePicker);
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(new Uint8Array([80, 75, 3, 4]), {
          headers: { 'Content-Digest': 'sha-256=:digest:', 'Content-Type': 'application/zip' },
        }),
      ),
    );

    const result = await downloadApiFile({
      filename: 'export.zip',
      path: '/exports/one',
      validateHeaders(headers) {
        events.push(`validate:${headers.get('content-digest')}`);
      },
    });

    expect(showSaveFilePicker).toHaveBeenCalledWith(
      expect.objectContaining({ suggestedName: 'export.zip' }),
    );
    expect(events).toEqual(['validate:sha-256=:digest:', 'write:80,75,3,4']);
    expect(result).toMatchObject({ delivery: 'streamed' });
  });
});

class MockXmlHttpRequest extends EventTarget {
  static latest: MockXmlHttpRequest | undefined;

  readonly upload = new EventTarget();
  readonly requestHeaders = new Map<string, string>();
  readonly responseHeaders = new Map<string, string>();
  body: Document | XMLHttpRequestBodyInit | null = null;
  method = '';
  responseText = '';
  status = 0;
  url = '';
  withCredentials = false;

  constructor() {
    super();
    MockXmlHttpRequest.latest = this;
  }

  open(method: string, url: string) {
    this.method = method;
    this.url = url;
  }

  setRequestHeader(name: string, value: string) {
    this.requestHeaders.set(name.toLowerCase(), value);
  }

  getResponseHeader(name: string) {
    return this.responseHeaders.get(name.toLowerCase()) ?? null;
  }

  send(body?: Document | XMLHttpRequestBodyInit | null) {
    this.body = body ?? null;
  }

  abort() {
    this.dispatchEvent(new Event('abort'));
  }
}

describe('uploadApiMultipart', () => {
  afterEach(() => {
    MockXmlHttpRequest.latest = undefined;
    vi.unstubAllGlobals();
  });

  it('posts bounded multipart data to the same-origin API and reports progress', async () => {
    vi.stubGlobal('XMLHttpRequest', MockXmlHttpRequest);
    const onProgress = vi.fn();
    const file = new File(['paper'], 'paper.pdf', { type: 'application/pdf' });
    const upload = uploadApiMultipart({
      path: '/orders/order-1/files',
      headers: { 'Idempotency-Key': 'upload-1' },
      fields: { role: 'primary_paper', expectedSha256: 'a'.repeat(64) },
      file,
      onProgress,
      schema: z.object({ id: z.string() }),
    });
    const request = MockXmlHttpRequest.latest;
    if (!request) throw new Error('Expected an upload request.');

    expect(request.method).toBe('POST');
    expect(request.url).toBe('http://127.0.0.1:8080/api/v1/orders/order-1/files');
    expect(request.withCredentials).toBe(true);
    expect(request.requestHeaders.get('idempotency-key')).toBe('upload-1');
    expect(request.requestHeaders.has('content-type')).toBe(false);
    expect(request.body).toBeInstanceOf(FormData);
    const body = request.body as FormData;
    expect(body.get('role')).toBe('primary_paper');
    expect(body.get('file')).toBe(file);

    request.upload.dispatchEvent(
      new ProgressEvent('progress', { lengthComputable: true, loaded: 25, total: 100 }),
    );
    expect(onProgress).toHaveBeenCalledWith(25);

    request.status = 201;
    request.responseText = JSON.stringify({ data: { id: 'file-1' } });
    request.responseHeaders.set('etag', '"2"');
    request.dispatchEvent(new Event('load'));
    await expect(upload).resolves.toEqual({ data: { id: 'file-1' }, etag: '"2"' });
  });

  it('aborts the API request when the caller cancels', async () => {
    vi.stubGlobal('XMLHttpRequest', MockXmlHttpRequest);
    const controller = new AbortController();
    const upload = uploadApiMultipart({
      path: '/orders/order-1/files',
      fields: {},
      file: new File(['paper'], 'paper.pdf', { type: 'application/pdf' }),
      signal: controller.signal,
      schema: z.unknown(),
    });

    controller.abort();
    await expect(upload).rejects.toMatchObject({ name: 'AbortError' });
  });
});
