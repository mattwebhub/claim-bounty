import { zodResolver } from '@hookform/resolvers/zod';
import { useMemo, useState, type MouseEvent } from 'react';
import { useForm } from 'react-hook-form';
import { Link } from 'react-router-dom';
import { isApiError, supportsStreamingApiDownloads } from '@/shared/api';
import { fileRoleLabels } from '@/features/claim-intake';
import { Alert, AlertDescription, AlertTitle, Button, PagePending } from '@/shared/ui';
import { adminFileDownloadUrl, exportDownloadUrl } from '../api/admin-order.service';
import {
  useAdminAccessRevocation,
  useAdminOrder,
  useCreateExport,
  useDownloadExport,
  useExportStatus,
} from '../api/admin-order.queries';
import {
  exportReadinessSchema,
  type AdminOrder,
  type ExportReadinessInput,
  type ExportReadinessValues,
} from '../model/admin-order.schema';
import { AdminIntakeEditor } from './admin-intake-editor';

interface AdminOrderDetailProps {
  csrfToken: string;
  onAccessDenied: () => void;
  orderId: string;
}

function readable(value: string) {
  return value.replaceAll('_', ' ');
}

function eventHistory(order: AdminOrder) {
  const events = order.events.map((event) => ({
    date: event.createdAt,
    label: readable(event.type),
  }));
  if (events.length === 0) events.push({ date: order.createdAt, label: 'Intake created' });
  return events.sort((left, right) => Date.parse(right.date) - Date.parse(left.date));
}

function contentDigest(sha256: string) {
  const bytes = sha256.match(/.{2}/g)?.map((pair) => Number.parseInt(pair, 16)) ?? [];
  return `sha-256=:${btoa(String.fromCharCode(...bytes))}:`;
}

function ExportIntegrity({ sha256 }: { sha256: string | null | undefined }) {
  if (!sha256) return null;
  return (
    <dl className="export-integrity">
      <div>
        <dt>Offline verifier SHA-256</dt>
        <dd>
          <code>{sha256}</code>
        </dd>
      </div>
      <div>
        <dt>Expected response Content-Digest</dt>
        <dd>
          <code>{contentDigest(sha256)}</code>
        </dd>
      </div>
    </dl>
  );
}

export function AdminOrderDetail({ csrfToken, onAccessDenied, orderId }: AdminOrderDetailProps) {
  const orderQuery = useAdminOrder(orderId);
  const accessDenied = useAdminAccessRevocation(orderQuery.error, onAccessDenied);
  const [createdExportId, setCreatedExportId] = useState<string | null>(null);
  const exportQuery = useExportStatus(orderId, createdExportId);
  const exportMutation = useCreateExport(orderId, csrfToken);
  const downloadMutation = useDownloadExport();
  const [downloadReceipt, setDownloadReceipt] = useState<{
    contentDigest: string;
    delivery: 'streamed';
    exportId: string;
    sha256: string;
  } | null>(null);
  const form = useForm<ExportReadinessValues, unknown, ExportReadinessInput>({
    resolver: zodResolver(exportReadinessSchema),
    defaultValues: {
      metadataReviewed: false,
      filesReviewed: false,
      retentionPolicyVersion: 'claimbounty-p0.1',
      preserveRunOutputs: true,
    },
  });
  const order = orderQuery.data;
  const history = useMemo(() => (order ? eventHistory(order) : []), [order]);

  if (orderQuery.isPending) return <PagePending />;
  if (accessDenied) {
    return (
      <Alert variant="destructive">
        <div>
          <AlertTitle>Admin access was removed</AlertTitle>
          <AlertDescription>Order data was cleared from this browser view.</AlertDescription>
        </div>
      </Alert>
    );
  }
  if (orderQuery.isError || !order) {
    return (
      <Alert variant="destructive">
        <div>
          <AlertTitle>Order could not be loaded</AlertTitle>
          <AlertDescription>
            {isApiError(orderQuery.error)
              ? `${orderQuery.error.message}${orderQuery.error.requestId ? ` Request ID: ${orderQuery.error.requestId}.` : ''}`
              : 'The order is unavailable.'}
          </AlertDescription>
          <Button variant="secondary" onClick={() => void orderQuery.refetch()}>
            Retry
          </Button>
        </div>
      </Alert>
    );
  }

  const allFilesClean =
    order.files.length > 0 && order.files.every((file) => file.status === 'clean');
  const hasPrimary = order.files.some((file) => file.role === 'primary_paper');
  const canExport =
    allFilesClean &&
    hasPrimary &&
    order.status === 'ready_for_export' &&
    Boolean(order.frozenIntake) &&
    order.readinessIssues.length === 0;
  const activeExport = exportQuery.data ?? order.exports.find(({ id }) => id === createdExportId);
  const create = form.handleSubmit(async (input) => {
    try {
      const result = await exportMutation.mutateAsync(input);
      setCreatedExportId(result.id);
    } catch {
      // The mutation state renders the safe API error and retry path.
    }
  });
  async function download(event: MouseEvent<HTMLAnchorElement>, exportRecord: typeof activeExport) {
    if (!supportsStreamingApiDownloads()) return;
    event.preventDefault();
    if (!exportRecord || downloadMutation.isPending) return;
    try {
      const result = await downloadMutation.mutateAsync(exportRecord);
      setDownloadReceipt({
        contentDigest: result.contentDigest,
        delivery: result.delivery,
        exportId: exportRecord.id,
        sha256: result.sha256,
      });
    } catch {
      // The mutation state renders the safe error and preserves any earlier verified digest.
    }
  }

  return (
    <section aria-labelledby="order-title">
      <Link className="back-link" to="/admin">
        ← All orders
      </Link>
      <div className="detail-heading">
        <div>
          <p className="eyebrow">{order.publicReference}</p>
          <h1 id="order-title">{order.title}</h1>
          <p>{order.submitterEmail ?? 'Submitter contact removed under retention policy'}</p>
        </div>
        <span className={`status-pill status-${order.status}`}>{readable(order.status)}</span>
      </div>

      <div className="admin-detail-grid">
        <div className="detail-main">
          <article className="detail-panel">
            <h2>Customer intake</h2>
            <dl className="metadata-list">
              <div>
                <dt>Purpose</dt>
                <dd>{order.purpose}</dd>
              </div>
              <div>
                <dt>Target claim</dt>
                <dd>{order.targetClaim.text}</dd>
              </div>
              <div>
                <dt>Claim location</dt>
                <dd>{order.targetClaim.sourceLocation}</dd>
              </div>
            </dl>
            <h3>Permissions and privacy</h3>
            <ul className="plain-list">
              <li>
                Execute supplied code:{' '}
                {order.permissions.executeSuppliedCode ? 'Allowed' : 'Not allowed'}
              </li>
              <li>
                External search: {order.permissions.externalSearch ? 'Allowed' : 'Not allowed'}
              </li>
              <li>
                Participant-level data:{' '}
                {order.privacy.containsParticipantLevelData ? 'Declared' : 'Not declared'}
              </li>
              <li>
                Direct identifiers:{' '}
                {order.privacy.containsDirectIdentifiers ? 'Declared' : 'Not declared'}
              </li>
            </ul>
          </article>

          <AdminIntakeEditor order={order} csrfToken={csrfToken} />

          <article className="detail-panel">
            <h2>Files and scan states</h2>
            <ul className="admin-file-list">
              {order.files.map((file) => (
                <li key={file.id}>
                  <span className="file-state">{file.status === 'clean' ? 'Clean' : 'Review'}</span>
                  <div>
                    <strong>{file.originalDisplayName}</strong>
                    <span>
                      {fileRoleLabels[file.role]} · {readable(file.status)}
                    </span>
                    {file.rejectionCode ? <span>Reason: {file.rejectionCode}</span> : null}
                  </div>
                  {file.status === 'clean' ? (
                    <Button asChild variant="secondary" size="sm">
                      <a href={adminFileDownloadUrl(order.id, file.id)} download>
                        Download
                      </a>
                    </Button>
                  ) : (
                    <span className="unavailable-label">Unavailable</span>
                  )}
                </li>
              ))}
            </ul>
          </article>

          <article className="detail-panel">
            <h2>Prepare local audit handoff</h2>
            {!supportsStreamingApiDownloads() ? (
              <p className="field-help">
                This browser will handle the export as a native download without buffering it in
                this page. Verify the saved archive with the displayed SHA-256 before opening it.
              </p>
            ) : null}
            {!canExport ? (
              <Alert variant="warning">
                <div>
                  <AlertTitle>Export is not ready</AlertTitle>
                  <AlertDescription>
                    Resolve every readiness issue, freeze the local handoff intake, and wait for all
                    included files to pass inspection.
                  </AlertDescription>
                </div>
              </Alert>
            ) : null}
            <form className="form-stack" onSubmit={create} noValidate>
              <label className="check-row">
                <input type="checkbox" {...form.register('metadataReviewed')} />I reviewed the
                claim, location, permissions, and privacy declaration.
              </label>
              {form.formState.errors.metadataReviewed ? (
                <span className="field-error">
                  {form.formState.errors.metadataReviewed.message}
                </span>
              ) : null}
              <label className="check-row">
                <input type="checkbox" {...form.register('filesReviewed')} />I confirmed every
                included file has passed inspection.
              </label>
              {form.formState.errors.filesReviewed ? (
                <span className="field-error">{form.formState.errors.filesReviewed.message}</span>
              ) : null}
              <label htmlFor="retention-version">Retention policy version</label>
              <input id="retention-version" {...form.register('retentionPolicyVersion')} />
              <label className="check-row">
                <input type="checkbox" {...form.register('preserveRunOutputs')} />
                Preserve later local run outputs with the handoff
              </label>
              <Button type="submit" disabled={!canExport || exportMutation.isPending}>
                {exportMutation.isPending ? 'Queueing export…' : 'Create export'}
              </Button>
            </form>
            {exportMutation.isError ? (
              <Alert variant="destructive">
                <div>
                  <AlertTitle>Export could not be created</AlertTitle>
                  <AlertDescription>
                    {isApiError(exportMutation.error)
                      ? `${exportMutation.error.message}${exportMutation.error.requestId ? ` Request ID: ${exportMutation.error.requestId}.` : ''}`
                      : 'Try again.'}
                  </AlertDescription>
                </div>
              </Alert>
            ) : null}
            {activeExport ? (
              <div className="active-export" role="status" aria-live="polite">
                <strong>Export {readable(activeExport.status)}</strong>
                {activeExport.status === 'ready' ? (
                  <>
                    <ExportIntegrity sha256={activeExport.sha256} />
                    <Button asChild aria-disabled={downloadMutation.isPending}>
                      <a
                        href={exportDownloadUrl(order.id, activeExport)}
                        download
                        onClick={(event) => void download(event, activeExport)}
                      >
                        {downloadMutation.isPending ? 'Downloading export…' : 'Download export'}
                      </a>
                    </Button>
                  </>
                ) : (
                  <span>The page will check again automatically.</span>
                )}
              </div>
            ) : null}
            {downloadMutation.isError ? (
              <Alert variant="destructive">
                <div>
                  <AlertTitle>Export could not be downloaded safely</AlertTitle>
                  <AlertDescription>
                    {isApiError(downloadMutation.error)
                      ? downloadMutation.error.message
                      : 'Try the download again.'}
                  </AlertDescription>
                </div>
              </Alert>
            ) : null}
            {downloadReceipt ? (
              <dl className="export-integrity" role="status" aria-live="polite">
                <div>
                  <dt>Download delivery</dt>
                  <dd>Streamed directly to the selected file.</dd>
                </div>
                <div>
                  <dt>Matched response Content-Digest header</dt>
                  <dd>
                    <code>{downloadReceipt.contentDigest}</code>
                  </dd>
                </div>
                <div>
                  <dt>Offline verifier SHA-256 for export {downloadReceipt.exportId}</dt>
                  <dd>
                    <code>{downloadReceipt.sha256}</code>
                  </dd>
                </div>
              </dl>
            ) : null}
          </article>
        </div>

        <aside className="detail-side" aria-label="Order history">
          <section className="detail-panel">
            <h2>Event history</h2>
            <ol className="timeline">
              {history.map((event, index) => (
                <li key={`${event.date}-${index}`}>
                  <strong>{event.label}</strong>
                  <time dateTime={event.date}>{new Date(event.date).toLocaleString()}</time>
                </li>
              ))}
            </ol>
          </section>
          <section className="detail-panel">
            <h2>Export history</h2>
            {order.exports.length === 0 ? (
              <p>No exports created.</p>
            ) : (
              <ul className="plain-list">
                {order.exports.map((item) => (
                  <li key={item.id}>
                    <strong>{readable(item.status)}</strong> ·{' '}
                    {new Date(item.createdAt).toLocaleString()}
                    {item.status === 'ready' ? (
                      <>
                        <ExportIntegrity sha256={item.sha256} />
                        <a
                          href={exportDownloadUrl(order.id, item)}
                          download
                          onClick={(event) => void download(event, item)}
                        >
                          Download
                        </a>
                      </>
                    ) : null}
                  </li>
                ))}
              </ul>
            )}
          </section>
        </aside>
      </div>
    </section>
  );
}
