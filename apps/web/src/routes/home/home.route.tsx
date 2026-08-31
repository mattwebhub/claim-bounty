import { z } from 'zod';
import { useCallback, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { ClaimIntake } from '@/features/claim-intake';
import { IdentityGate, useSession } from '@/features/identity';
import { PagePending } from '@/shared/ui';

const acceptedFiles = [
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
] as const;

const processSteps = [
  {
    number: '01',
    title: 'Confirm the central claim',
    description:
      'We identify the paper’s main testable claim and trace it to the exact data, method, figure, or table that supports it.',
  },
  {
    number: '02',
    title: 'Reproduce and challenge it',
    description:
      'We rerun the supplied analysis and test defensible alternatives across data, models, covariates, and inference.',
  },
  {
    number: '03',
    title: 'Verify and report',
    description:
      'Independent checks separate reproducible evidence from unresolved inconsistencies and document every finding.',
  },
  {
    number: '04',
    title: 'Receive the claim verdict',
    description: 'A private decision package you can act on before submission.',
    outputs: [
      'Claim verdict',
      'Robustness summary',
      'Consistency findings',
      'Prioritized corrections',
      'Reproduction evidence',
    ],
  },
] as const;

interface LandingDropzoneProps {
  files: File[];
  onFilesChange: (files: File[]) => void;
}

function LandingDropzone({ files, onFilesChange }: LandingDropzoneProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [error, setError] = useState('');
  const [isDragActive, setIsDragActive] = useState(false);

  function addFiles(selected: FileList | File[]) {
    setError('');
    const incoming = Array.from(selected).slice(0, Math.max(0, 20 - files.length));
    const next = [...files, ...incoming];
    if (incoming.some((file) => file.size === 0 || file.size > 262_144_000)) {
      setError('Each file must be larger than 0 bytes and no more than 250 MB.');
      return;
    }
    if (next.reduce((total, file) => total + file.size, 0) > 1_073_741_824) {
      setError('The complete upload must be no more than 1 GB.');
      return;
    }
    onFilesChange(next);
  }

  return (
    <div className="landing-drop-wrap">
      <input
        ref={inputRef}
        className="sr-only"
        id="landing-evidence-files"
        aria-label="Manuscript and research files"
        type="file"
        multiple
        accept={acceptedFiles.join(',')}
        onChange={(event) => {
          if (event.target.files) addFiles(event.target.files);
        }}
      />
      <button
        className="landing-dropzone focus-ring"
        data-drag-active={isDragActive}
        data-has-files={files.length > 0}
        type="button"
        onClick={() => inputRef.current?.click()}
        onDragEnter={(event) => {
          event.preventDefault();
          setIsDragActive(true);
        }}
        onDragOver={(event) => {
          event.preventDefault();
          setIsDragActive(true);
        }}
        onDragLeave={(event) => {
          const nextTarget = event.relatedTarget;
          if (!(nextTarget instanceof Node) || !event.currentTarget.contains(nextTarget)) {
            setIsDragActive(false);
          }
        }}
        onDrop={(event) => {
          event.preventDefault();
          setIsDragActive(false);
          addFiles(event.dataTransfer.files);
        }}
      >
        <strong className="dropzone-title">
          <span className="dropzone-title-idle">
            {files.length > 0 ? 'Add more research materials' : 'Drop your manuscript here'}
          </span>
          <span className="dropzone-title-hover">
            <img
              className="dropzone-action-logo"
              src="/claimbounty-fox-loupe-icon.png"
              alt="Peer2Paper"
            />
            <span>Audit my paper</span>
          </span>
        </strong>
        <span className="dropzone-accepted">
          Accepted: PDF manuscripts, Word and text documents, CSV, Excel, statistical data, code,
          notebooks, and ZIP archives. Up to 20 files, 250 MB each.
        </span>
      </button>

      {error ? (
        <p className="landing-file-error" role="alert">
          {error}
        </p>
      ) : null}

      {files.length > 0 ? (
        <ul className="landing-file-list" aria-label="Selected manuscript and research files">
          {files.map((file, index) => (
            <li key={`${file.name}-${file.size}-${file.lastModified}-${index}`}>
              <span className="file-kind" aria-hidden="true">
                File
              </span>
              <span>{file.name}</span>
              <button
                type="button"
                className="landing-file-remove focus-ring"
                aria-label={`Remove ${file.name}`}
                onClick={() => {
                  onFilesChange(files.filter((_, fileIndex) => fileIndex !== index));
                }}
              >
                <span aria-hidden="true">×</span>
              </button>
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}

export function Component() {
  const session = useSession();
  const [initialFiles, setInitialFiles] = useState<File[]>([]);
  const [search, setSearch] = useSearchParams();
  const submitterSession = session.data?.audience === 'submitter' ? session.data : null;
  const draftOrderId = z.uuid().safeParse(search.get('draft'));

  const setDraftOrderId = useCallback(
    (orderId: string | null) => {
      setSearch(
        (current) => {
          const next = new URLSearchParams(current);
          if (orderId) next.set('draft', orderId);
          else next.delete('draft');
          return next;
        },
        { replace: true },
      );
    },
    [setSearch],
  );

  return (
    <main id="main-content" className="claim-page" tabIndex={-1}>
      <section className="claim-hero" aria-labelledby="hero-title">
        <img
          className="brain-archival-projection"
          src="/claimbounty-archival-projection-v3.png"
          alt=""
          aria-hidden="true"
        />
        <div className="claim-hero-inner">
          <h1 id="hero-title">Find the weakness in your paper before reviewers do</h1>
          <p className="hero-description">
            Upload your manuscript and research materials. Peer2Paper independently tests whether
            your central claim can be reproduced, how robust it is, and what should be fixed before
            submission.
          </p>

          {!submitterSession ? (
            <div className="hero-intake">
              <LandingDropzone files={initialFiles} onFilesChange={setInitialFiles} />
              {initialFiles.length > 0 && !session.isPending ? (
                <div className="claim-auth-reveal">
                  <IdentityGate audience="submitter" />
                </div>
              ) : null}
            </div>
          ) : null}
        </div>
      </section>

      {session.isPending ? (
        <div className="claim-workflow claim-workflow-pending">
          <PagePending />
        </div>
      ) : submitterSession ? (
        <div className="claim-workflow">
          <ClaimIntake
            csrfToken={submitterSession.csrfToken}
            draftOrderId={draftOrderId.success ? draftOrderId.data : null}
            initialFiles={initialFiles}
            onDraftOrderIdChange={setDraftOrderId}
          />
        </div>
      ) : null}

      <section className="process-section" aria-labelledby="process-title">
        <div className="process-inner">
          <header className="process-heading">
            <h2 id="process-title">See the full audit before reviewers see the paper.</h2>
            <p>
              From materials to verdict
              <br />
              Peer2Paper audit sequence
            </p>
          </header>

          <div className="process-row materials-row">
            <p className="process-row-label">You provide</p>
            <h3>Paper, data, code, and supplementary information</h3>
            <p>
              Start with the complete record behind the result so every claim can be traced to its
              source.
            </p>
          </div>

          <ol className="process-list">
            {processSteps.map((step) => (
              <li key={step.number}>
                <p className="process-row-label">Step {step.number}</p>
                <h3>{step.title}</h3>
                <div className="process-step-copy">
                  <p>{step.description}</p>
                  {'outputs' in step ? (
                    <ul className="output-list" aria-label="Peer2Paper audit outputs">
                      {step.outputs.map((output) => (
                        <li key={output}>{output}</li>
                      ))}
                    </ul>
                  ) : null}
                </div>
              </li>
            ))}
          </ol>
        </div>
      </section>

      <footer className="claim-footer">
        <p>Authored by Matheus Paranhos</p>
      </footer>
    </main>
  );
}
