import { zodResolver } from '@hookform/resolvers/zod';
import { useEffect, useRef, useState } from 'react';
import { useForm } from 'react-hook-form';
import { isApiError } from '@/shared/api';
import { Alert, AlertDescription, AlertTitle, Button, Input, PagePending } from '@/shared/ui';
import {
  useCreateOrder,
  useOrder,
  useRemoveOrderFile,
  useSubmitOrder,
  useUploadOrderFile,
} from '../api/order.queries';
import {
  fileRoleLabels,
  fileRoles,
  orderIntakeSchema,
  type FileRole,
  type Order,
  type OrderIntakeInput,
  type OrderIntakeValues,
  type SelectedFile,
} from '../model/order.schema';

interface ClaimIntakeProps {
  csrfToken: string;
  draftOrderId?: string | null;
  initialFiles?: File[];
  onDraftOrderIdChange?: (orderId: string | null) => void;
}

const acceptedEvidenceTypes = [
  '.pdf',
  '.txt',
  '.md',
  '.docx',
  '.csv',
  '.tsv',
  '.xlsx',
  '.json',
  '.parquet',
  '.dta',
  '.sav',
  '.rds',
  '.rdata',
  '.r',
  '.py',
  '.ipynb',
  '.do',
  '.sql',
  '.sh',
  '.zip',
  '.yaml',
  '.yml',
  '.toml',
].join(',');

const recoverableStatuses = new Set<Order['status']>([
  'draft',
  'awaiting_email_verification',
  'uploading',
]);

function ignoreDraftOrderIdChange() {
  return undefined;
}

function describeError(error: unknown) {
  if (error instanceof DOMException && error.name === 'AbortError') return 'Upload cancelled.';
  if (!isApiError(error)) return 'Something went wrong. Try again.';
  return `${error.message}${error.requestId ? ` Request ID: ${error.requestId}.` : ''}`;
}

function isPdf(file: File) {
  return file.type === 'application/pdf' || file.name.toLowerCase().endsWith('.pdf');
}

function selectInitialFiles(initialFiles: File[]): SelectedFile[] {
  let hasPrimary = false;
  return initialFiles.map((file) => {
    const role: FileRole = !hasPrimary && isPdf(file) ? 'primary_paper' : 'other_evidence';
    if (role === 'primary_paper') hasPrimary = true;
    return {
      id: crypto.randomUUID(),
      idempotencyKey: crypto.randomUUID(),
      file,
      role,
      progress: 0,
      status: 'selected',
    };
  });
}

function formatBytes(bytes: number) {
  if (bytes < 1_000_000) return `${Math.ceil(bytes / 1000)} KB`;
  return `${(bytes / 1_000_000).toFixed(1)} MB`;
}

export function ClaimIntake({
  csrfToken,
  draftOrderId = null,
  initialFiles = [],
  onDraftOrderIdChange = ignoreDraftOrderIdChange,
}: ClaimIntakeProps) {
  const [files, setFiles] = useState<SelectedFile[]>(() => selectInitialFiles(initialFiles));
  const [createdDraft, setCreatedDraft] = useState<Order | null>(null);
  const [submittedReceipt, setSubmittedReceipt] = useState<Order | null>(null);
  const [workflowError, setWorkflowError] = useState('');
  const [isDropActive, setIsDropActive] = useState(false);
  const [termsAccepted, setTermsAccepted] = useState(false);
  const [uploadsAuthorized, setUploadsAuthorized] = useState(false);
  const [analysisUseAuthorized, setAnalysisUseAuthorized] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const receiptRef = useRef<HTMLHeadingElement>(null);
  const [controllers] = useState(() => new Map<string, AbortController>());
  const orderQuery = useOrder(draftOrderId);
  const createMutation = useCreateOrder();
  const uploadMutation = useUploadOrderFile();
  const removeMutation = useRemoveOrderFile();
  const submitMutation = useSubmitOrder();
  const restoredOrder = orderQuery.data;
  const restoredDraft =
    restoredOrder && recoverableStatuses.has(restoredOrder.status) ? restoredOrder : null;
  const draft = restoredDraft ?? createdDraft;
  const restoredReceipt =
    restoredOrder && !recoverableStatuses.has(restoredOrder.status) ? restoredOrder : null;
  const receipt = submittedReceipt ?? restoredReceipt;
  const {
    formState: { errors },
    handleSubmit,
    register,
    reset,
  } = useForm<OrderIntakeValues, unknown, OrderIntakeInput>({
    resolver: zodResolver(orderIntakeSchema),
    defaultValues: {
      title: '',
      purpose: '',
      targetClaim: { text: '', sourceLocation: '' },
      permissions: { executeSuppliedCode: false, externalSearch: false },
      privacy: { containsParticipantLevelData: false, containsDirectIdentifiers: false },
    },
  });

  useEffect(() => {
    if (receipt) receiptRef.current?.focus();
  }, [receipt]);

  useEffect(() => {
    if (draftOrderId && restoredReceipt) {
      onDraftOrderIdChange(null);
    }
  }, [draftOrderId, onDraftOrderIdChange, restoredReceipt]);

  useEffect(() => {
    if (!restoredDraft) return;
    reset({
      title: restoredDraft.title,
      purpose: restoredDraft.purpose,
      targetClaim: {
        text: restoredDraft.targetClaim.text,
        sourceLocation: restoredDraft.targetClaim.sourceLocation ?? '',
      },
      permissions: restoredDraft.permissions,
      privacy: restoredDraft.privacy,
    });
  }, [reset, restoredDraft]);

  function addFiles(selected: FileList | File[]) {
    setWorkflowError('');
    setFiles((current) => {
      const room = Math.max(0, 20 - current.length);
      const incoming = Array.from(selected).slice(0, room);
      const exceedsIndividualLimit = incoming.some(
        (file) => file.size > 262_144_000 || file.size === 0,
      );
      const exceedsOrderLimit =
        [...current.map(({ file }) => file), ...incoming].reduce(
          (total, file) => total + file.size,
          0,
        ) > 1_073_741_824;
      if (exceedsIndividualLimit || exceedsOrderLimit) {
        setWorkflowError(
          'Each file must be 250 MB or less and the full intake must be 1 GB or less.',
        );
        return current;
      }
      let hasPrimary =
        current.some(({ role }) => role === 'primary_paper') ||
        Boolean(draft?.files.some(({ role }) => role === 'primary_paper'));
      return [
        ...current,
        ...incoming.map((file): SelectedFile => {
          const role: FileRole = !hasPrimary && isPdf(file) ? 'primary_paper' : 'other_evidence';
          if (role === 'primary_paper') hasPrimary = true;
          return {
            id: crypto.randomUUID(),
            idempotencyKey: crypto.randomUUID(),
            file,
            role,
            progress: 0,
            status: 'selected',
          };
        }),
      ];
    });
  }

  function updateFile(id: string, change: Partial<SelectedFile>) {
    setFiles((current) => current.map((item) => (item.id === id ? { ...item, ...change } : item)));
  }

  function setRole(id: string, role: FileRole) {
    setFiles((current) =>
      current.map((item) => {
        if (item.id === id) return { ...item, role };
        if (role === 'primary_paper' && item.role === 'primary_paper') {
          return { ...item, role: 'other_evidence' };
        }
        return item;
      }),
    );
  }

  async function removeFile(selected: SelectedFile) {
    setWorkflowError('');
    if (draft && selected.serverFileId) {
      try {
        await removeMutation.mutateAsync({
          csrfToken,
          fileId: selected.serverFileId,
          orderId: draft.id,
        });
      } catch (error) {
        setWorkflowError(describeError(error));
        return;
      }
    }
    setFiles((current) => current.filter((item) => item.id !== selected.id));
  }

  async function removeRestoredFile(fileId: string) {
    if (!draft) return;
    setWorkflowError('');
    try {
      await removeMutation.mutateAsync({ csrfToken, fileId, orderId: draft.id });
    } catch (error) {
      setWorkflowError(describeError(error));
    }
  }

  async function uploadOne(orderId: string, selected: SelectedFile) {
    const controller = new AbortController();
    controllers.set(selected.id, controller);
    updateFile(selected.id, { status: 'hashing', progress: 0 });
    try {
      const uploaded = await uploadMutation.mutateAsync({
        csrfToken,
        orderId,
        file: selected.file,
        idempotencyKey: selected.idempotencyKey,
        role: selected.role,
        signal: controller.signal,
        onHashing: () => {
          updateFile(selected.id, { status: 'hashing' });
        },
        onProgress: (progress) => {
          updateFile(selected.id, { status: 'uploading', progress });
        },
      });
      updateFile(selected.id, { status: 'complete', progress: 100, serverFileId: uploaded.id });
      return true;
    } catch (error) {
      updateFile(selected.id, {
        status:
          error instanceof DOMException && error.name === 'AbortError' ? 'cancelled' : 'error',
      });
      setWorkflowError(describeError(error));
      return false;
    } finally {
      controllers.delete(selected.id);
    }
  }

  const createAndUpload = handleSubmit(async (input) => {
    setWorkflowError('');
    const primaryFiles = files.filter(({ role }) => role === 'primary_paper');
    const primaryFile = primaryFiles[0];
    const restoredPrimaryCount =
      draft?.files.filter(
        ({ role, status }) =>
          role === 'primary_paper' && status !== 'rejected' && status !== 'expired',
      ).length ?? 0;
    const primaryCount = restoredPrimaryCount + primaryFiles.length;
    if (
      primaryCount !== 1 ||
      (primaryFiles.length === 1 && (!primaryFile || !isPdf(primaryFile.file)))
    ) {
      setWorkflowError('Add exactly one PDF and label it Primary paper.');
      document.querySelector<HTMLInputElement>('#evidence-files')?.focus();
      return;
    }
    if (primaryFile && primaryFile.file.size > 52_428_800) {
      setWorkflowError('The primary PDF must be 50 MB or less.');
      document.querySelector<HTMLInputElement>('#evidence-files')?.focus();
      return;
    }
    try {
      const order = draft ?? (await createMutation.mutateAsync({ input, csrfToken }));
      if (!draft) onDraftOrderIdChange(order.id);
      setCreatedDraft(order);
      for (const selected of files) {
        if (selected.status !== 'complete') await uploadOne(order.id, selected);
      }
    } catch (error) {
      setWorkflowError(describeError(error));
    }
  });

  async function retryFile(selected: SelectedFile) {
    if (draft) await uploadOne(draft.id, selected);
  }

  async function finalizeOrder() {
    if (!draft || !termsAccepted || !uploadsAuthorized || !analysisUseAuthorized) return;
    setWorkflowError('');
    try {
      const submitted = await submitMutation.mutateAsync({
        orderId: draft.id,
        csrfToken,
        termsAccepted,
        uploadsAuthorized,
        analysisUseAuthorized,
      });
      setSubmittedReceipt(submitted);
      onDraftOrderIdChange(null);
    } catch (error) {
      setWorkflowError(describeError(error));
    }
  }

  if (draftOrderId && orderQuery.isPending) return <PagePending />;

  if (draftOrderId && orderQuery.isError) {
    return (
      <Alert variant="destructive">
        <div>
          <AlertTitle>Draft could not be restored</AlertTitle>
          <AlertDescription>
            {describeError(orderQuery.error)} The existing draft was not replaced.
          </AlertDescription>
          <Button
            variant="secondary"
            onClick={() => {
              onDraftOrderIdChange(null);
            }}
          >
            Start a new intake
          </Button>
        </div>
      </Alert>
    );
  }

  if (receipt) {
    const accepted = !['rejected', 'cancelled', 'expired'].includes(receipt.status);
    return (
      <section className="receipt-card" aria-labelledby="receipt-title">
        <p className="step-label">Submitted</p>
        <h2 id="receipt-title" ref={receiptRef} tabIndex={-1}>
          {accepted ? 'Your evidence is in review' : 'This intake is closed'}
        </h2>
        <p>
          Save this private reference: <strong>{receipt.publicReference}</strong>
        </p>
        <dl className="receipt-details">
          <div>
            <dt>Status</dt>
            <dd>{receipt.status}</dd>
          </div>
          <div>
            <dt>Files received</dt>
            <dd>{receipt.files.length}</dd>
          </div>
        </dl>
        <p className="privacy-note">
          {accepted
            ? 'Files now enter private malware inspection. An administrator will prepare a local handoff; no audit is run from this website.'
            : 'This order cannot accept more uploads. Contact support with the private reference if you need help.'}
        </p>
      </section>
    );
  }

  const restoredFiles = (draft?.files ?? []).filter(
    (file) => !files.some(({ serverFileId }) => serverFileId === file.id),
  );
  const allRestoredFilesUsable = restoredFiles.every(
    ({ status }) => status !== 'rejected' && status !== 'expired',
  );
  const transfersDone =
    allRestoredFilesUsable &&
    files.every(({ status }) => status === 'complete') &&
    (restoredFiles.some(({ role }) => role === 'primary_paper') ||
      files.some(({ role, status }) => role === 'primary_paper' && status === 'complete'));

  return (
    <section className="intake-card" aria-labelledby="intake-title">
      <p className="step-label">Step 3 of 4</p>
      <h2 id="intake-title">Tell us what to test</h2>
      <p className="section-copy">
        Your files are ready. Give the audit one precise claim and show us where it appears in the
        paper.
      </p>
      <form className="form-stack intake-form" onSubmit={createAndUpload} noValidate>
        <fieldset disabled={Boolean(draft)}>
          <legend>Study and claim</legend>
          <label htmlFor="order-title">Study title</label>
          <Input id="order-title" invalid={Boolean(errors.title)} {...register('title')} />
          {errors.title ? <span className="field-error">{errors.title.message}</span> : null}
          <label htmlFor="order-purpose">What should the review establish?</label>
          <textarea id="order-purpose" rows={3} {...register('purpose')} />
          {errors.purpose ? <span className="field-error">{errors.purpose.message}</span> : null}
          <label htmlFor="claim-text">Exact target claim</label>
          <textarea id="claim-text" rows={5} {...register('targetClaim.text')} />
          {errors.targetClaim?.text ? (
            <span className="field-error">{errors.targetClaim.text.message}</span>
          ) : null}
          <label htmlFor="claim-location">Claim location</label>
          <Input
            id="claim-location"
            placeholder="Page, section, figure, or table"
            invalid={Boolean(errors.targetClaim?.sourceLocation)}
            {...register('targetClaim.sourceLocation')}
          />
          {errors.targetClaim?.sourceLocation ? (
            <span className="field-error">{errors.targetClaim.sourceLocation.message}</span>
          ) : null}
        </fieldset>

        <fieldset disabled={Boolean(draft)}>
          <legend>Permissions</legend>
          <p className="field-help">Choose only actions you authorize for the later local audit.</p>
          <label className="check-row">
            <input type="checkbox" {...register('permissions.executeSuppliedCode')} />
            Execute code I supply in an isolated environment
          </label>
          <label className="check-row">
            <input type="checkbox" {...register('permissions.externalSearch')} />
            Search external scholarly sources
          </label>
        </fieldset>

        <fieldset disabled={Boolean(draft)}>
          <legend>Privacy declaration</legend>
          <label className="check-row">
            <input type="checkbox" {...register('privacy.containsParticipantLevelData')} />
            Files contain participant-level data
          </label>
          <label className="check-row">
            <input type="checkbox" {...register('privacy.containsDirectIdentifiers')} />
            Files contain direct personal identifiers
          </label>
          <p className="privacy-note">
            Do not upload secrets, credentials, or data you lack permission to share. Files are
            quarantined until inspection passes.
          </p>
        </fieldset>

        <fieldset className="files-fieldset">
          <legend>Evidence files</legend>
          <input
            ref={inputRef}
            className="sr-only"
            id="evidence-files"
            aria-label="Evidence files"
            type="file"
            multiple
            accept={acceptedEvidenceTypes}
            onChange={(event) => {
              if (event.target.files) addFiles(event.target.files);
            }}
          />
          <button
            className="drop-surface focus-ring"
            data-drag-active={isDropActive}
            type="button"
            onClick={() => inputRef.current?.click()}
            onDragEnter={(event) => {
              event.preventDefault();
              setIsDropActive(true);
            }}
            onDragOver={(event) => {
              event.preventDefault();
              setIsDropActive(true);
            }}
            onDragLeave={(event) => {
              const nextTarget = event.relatedTarget;
              if (!(nextTarget instanceof Node) || !event.currentTarget.contains(nextTarget)) {
                setIsDropActive(false);
              }
            }}
            onDrop={(event) => {
              event.preventDefault();
              setIsDropActive(false);
              addFiles(event.dataTransfer.files);
            }}
          >
            <strong className="dropzone-title">
              <span className="dropzone-title-idle">Choose or drop more evidence here</span>
              <span className="dropzone-title-hover">
                <img
                  className="dropzone-action-logo"
                  src="/claimbounty-fox-loupe-icon.png"
                  alt="Peer2Paper"
                />
                <span>Audit my paper</span>
              </span>
            </strong>
            <span>
              Accepted: PDF, Word, text, tabular and statistical data, code, notebooks, and ZIP. One
              primary PDF is required.
            </span>
          </button>
        </fieldset>

        {restoredFiles.length > 0 ? (
          <div className="restored-files" role="status">
            <h3>Files restored from this draft</h3>
            <ul className="file-list" aria-label="Restored evidence files">
              {restoredFiles.map((file) => (
                <li key={file.id}>
                  <span className="file-kind" aria-hidden="true">
                    File
                  </span>
                  <div className="file-summary">
                    <strong>{file.originalDisplayName}</strong>
                    <span>
                      {formatBytes(file.sizeBytes)} · {file.status.replaceAll('_', ' ')}
                    </span>
                  </div>
                  <span>{fileRoleLabels[file.role]}</span>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    aria-label={`Remove ${file.originalDisplayName}`}
                    disabled={removeMutation.isPending}
                    onClick={() => {
                      void removeRestoredFile(file.id);
                    }}
                  >
                    <span aria-hidden="true">×</span>
                  </Button>
                </li>
              ))}
            </ul>
            <p className="field-help">
              Browser-selected files that did not finish uploading cannot be restored. Choose those
              files again; they will be added to this draft.
            </p>
          </div>
        ) : null}

        {files.length > 0 ? (
          <ul className="file-list" aria-label="Selected evidence files">
            {files.map((selected) => (
              <li key={selected.id}>
                <span className="file-kind" aria-hidden="true">
                  File
                </span>
                <div className="file-summary">
                  <strong>{selected.file.name}</strong>
                  <span>
                    {formatBytes(selected.file.size)} · {selected.status}
                  </span>
                  {selected.status === 'uploading' ? (
                    <progress value={selected.progress} max="100">
                      {selected.progress}%
                    </progress>
                  ) : null}
                </div>
                <label className="role-field">
                  <span>Role</span>
                  <select
                    value={selected.role}
                    disabled={selected.status === 'hashing' || selected.status === 'uploading'}
                    onChange={(event) => {
                      setRole(selected.id, event.target.value as FileRole);
                    }}
                  >
                    {fileRoles.map((role) => (
                      <option
                        key={role}
                        value={role}
                        disabled={role === 'primary_paper' && !isPdf(selected.file)}
                      >
                        {fileRoleLabels[role]}
                      </option>
                    ))}
                  </select>
                </label>
                {selected.status !== 'hashing' && selected.status !== 'uploading' ? (
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    aria-label={`Remove ${selected.file.name}`}
                    disabled={removeMutation.isPending}
                    onClick={() => {
                      void removeFile(selected);
                    }}
                  >
                    <span aria-hidden="true">×</span>
                  </Button>
                ) : null}
                {selected.status === 'hashing' || selected.status === 'uploading' ? (
                  <Button
                    type="button"
                    variant="secondary"
                    size="sm"
                    onClick={() => controllers.get(selected.id)?.abort()}
                  >
                    Cancel
                  </Button>
                ) : null}
                {selected.status === 'error' || selected.status === 'cancelled' ? (
                  <Button
                    type="button"
                    variant="secondary"
                    size="sm"
                    onClick={() => void retryFile(selected)}
                  >
                    Retry
                  </Button>
                ) : null}
              </li>
            ))}
          </ul>
        ) : null}

        {!draft ? (
          <Button type="submit" size="lg" disabled={createMutation.isPending}>
            {createMutation.isPending ? 'Creating secure intake…' : 'Create intake and upload'}
          </Button>
        ) : (
          <div className="submission-panel">
            <p className="step-label">Step 4 of 4</p>
            {files.some(({ status }) => status === 'selected') ? (
              <Button type="submit" size="lg" disabled={uploadMutation.isPending}>
                {uploadMutation.isPending ? 'Uploading…' : 'Upload selected files'}
              </Button>
            ) : null}
            <label className="check-row">
              <input
                type="checkbox"
                checked={termsAccepted}
                onChange={(event) => {
                  setTermsAccepted(event.target.checked);
                }}
              />
              I accept the Peer2Paper intake terms for this order.
            </label>
            <label className="check-row">
              <input
                type="checkbox"
                checked={uploadsAuthorized}
                onChange={(event) => {
                  setUploadsAuthorized(event.target.checked);
                }}
              />
              I authorize Peer2Paper to retain and privately inspect these uploaded files for this
              order.
            </label>
            <label className="check-row">
              <input
                type="checkbox"
                checked={analysisUseAuthorized}
                onChange={(event) => {
                  setAnalysisUseAuthorized(event.target.checked);
                }}
              />
              I authorize these files and private derived analysis files to be included in the
              internal local scientific audit handoff.
            </label>
            <p className="privacy-note">
              External redistribution is not authorized. This P0 intake cannot grant that
              permission, and an administrator cannot expand it.
            </p>
            <Button
              type="button"
              size="lg"
              disabled={
                !transfersDone ||
                !termsAccepted ||
                !uploadsAuthorized ||
                !analysisUseAuthorized ||
                submitMutation.isPending
              }
              onClick={() => void finalizeOrder()}
            >
              {submitMutation.isPending ? 'Submitting…' : 'Submit for review'}
            </Button>
            {!transfersDone ? (
              <p className="field-help">Finish or retry every upload first.</p>
            ) : null}
          </div>
        )}
      </form>
      {workflowError ? (
        <Alert variant="destructive">
          <div>
            <AlertTitle>Intake needs attention</AlertTitle>
            <AlertDescription>{workflowError}</AlertDescription>
          </div>
        </Alert>
      ) : null}
      <p className="sr-only" role="status" aria-live="polite">
        {files.filter(({ status }) => status === 'complete').length} of {files.length} files
        uploaded.
      </p>
    </section>
  );
}
