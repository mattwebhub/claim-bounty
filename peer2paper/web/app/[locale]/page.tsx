import { getTranslations, setRequestLocale } from "next-intl/server";
import Image from "next/image";
import { LandingDropzone } from "@/components/landing-dropzone";

type Item = { title: string; body: string };

export default async function HomePage({
  params,
}: {
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations("landing");
  const steps = t.raw("steps") as Item[];
  const outputs = t.raw("outputs") as Item[];

  return (
    <>
      <section className="landing-hero" aria-labelledby="landing-title">
        <Image
          className="landing-brain"
          src="/peer2paper-brain.png"
          alt=""
          aria-hidden="true"
          width={1448}
          height={1086}
          priority
        />
        <div className="landing-hero-inner">
          <h1 id="landing-title">
            {t("titleStart")} {t("titleEmphasis")} {t("titleEnd")}
          </h1>
          <p className="landing-description">{t("lede")}</p>
          <LandingDropzone />
        </div>
      </section>

      <section
        id="how"
        className="landing-process"
        aria-labelledby="process-title"
      >
        <div className="landing-process-inner">
          <header className="landing-process-heading">
            <h2 id="process-title">{t("howTitle")}</h2>
            <p>{t("howBody")}</p>
          </header>
          <div className="landing-process-row landing-materials-row">
            <p className="landing-process-label">{t("materials.label")}</p>
            <h3>{t("materials.title")}</h3>
            <p>{t("materials.body")}</p>
          </div>
          <ol className="landing-process-list">
            {steps.map((step, index) => (
              <li key={step.title}>
                <p className="landing-process-label">
                  {String(index + 1).padStart(2, "0")}
                </p>
                <h3>{step.title}</h3>
                <div className="landing-process-copy">
                  <p>{step.body}</p>
                  {index === steps.length - 1 ? (
                    <ul
                      className="landing-output-list"
                      aria-label={t("outputsTitle")}
                    >
                      {outputs.map((output) => (
                        <li key={output.title}>{output.title}</li>
                      ))}
                    </ul>
                  ) : null}
                </div>
              </li>
            ))}
          </ol>
        </div>
      </section>
    </>
  );
}
