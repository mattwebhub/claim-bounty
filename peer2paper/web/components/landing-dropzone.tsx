"use client";

import { useRef, useState } from "react";
import { useTranslations } from "next-intl";
import Image from "next/image";
import { Link } from "@/i18n/navigation";

const acceptedFiles = [
  ".pdf",
  ".txt",
  ".md",
  ".docx",
  ".csv",
  ".tsv",
  ".xlsx",
  ".json",
  ".parquet",
  ".dta",
  ".sav",
  ".rds",
  ".rdata",
  ".r",
  ".py",
  ".ipynb",
  ".do",
  ".sql",
  ".sh",
  ".zip",
  ".yaml",
  ".yml",
  ".toml",
] as const;

export function LandingDropzone() {
  const t = useTranslations("landing.upload");
  const inputRef = useRef<HTMLInputElement>(null);
  const [files, setFiles] = useState<File[]>([]);
  const [error, setError] = useState("");
  const [isDragActive, setIsDragActive] = useState(false);

  function addFiles(selected: FileList | File[]) {
    setError("");
    const incoming = Array.from(selected).slice(
      0,
      Math.max(0, 20 - files.length),
    );
    const next = [...files, ...incoming];
    if (incoming.some((file) => file.size === 0 || file.size > 262_144_000)) {
      setError(t("sizeError"));
      return;
    }
    if (next.reduce((total, file) => total + file.size, 0) > 1_073_741_824) {
      setError(t("totalError"));
      return;
    }
    setFiles(next);
  }

  return (
    <div className="landing-drop-wrap">
      <input
        ref={inputRef}
        className="sr-only"
        id="landing-evidence-files"
        aria-label={t("inputLabel")}
        type="file"
        multiple
        accept={acceptedFiles.join(",")}
        onChange={(event) => {
          if (event.target.files) addFiles(event.target.files);
          event.currentTarget.value = "";
        }}
      />
      <button
        className="landing-dropzone"
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
          const next = event.relatedTarget;
          if (!(next instanceof Node) || !event.currentTarget.contains(next))
            setIsDragActive(false);
        }}
        onDrop={(event) => {
          event.preventDefault();
          setIsDragActive(false);
          addFiles(event.dataTransfer.files);
        }}
      >
        <strong className="landing-drop-title">
          <span className="landing-drop-idle">
            {files.length ? t("more") : t("idle")}
          </span>
          <span className="landing-drop-hover">
            <Image
              src="/peer2paper-fox-loupe.png"
              alt=""
              width={512}
              height={512}
            />
            <span>{t("hover")}</span>
          </span>
        </strong>
        <span className="landing-drop-accepted">{t("accepted")}</span>
      </button>
      {error ? (
        <p className="landing-file-error" role="alert">
          {error}
        </p>
      ) : null}
      {files.length ? (
        <>
          <ul className="landing-file-list" aria-label={t("selectedFiles")}>
            {files.map((file, index) => (
              <li
                key={`${file.name}-${file.size}-${file.lastModified}-${index}`}
              >
                <span className="landing-file-kind" aria-hidden="true">
                  {t("file")}
                </span>
                <span>{file.name}</span>
                <button
                  type="button"
                  aria-label={t("remove", { name: file.name })}
                  onClick={() =>
                    setFiles(
                      files.filter((_, fileIndex) => fileIndex !== index),
                    )
                  }
                >
                  ×
                </button>
              </li>
            ))}
          </ul>
          <div className="landing-upload-next">
            <p>{t("localNotice")}</p>
            <Link href="/signup" className="button">
              {t("continue")}
            </Link>
          </div>
        </>
      ) : null}
    </div>
  );
}
