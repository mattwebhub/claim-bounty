import type {Metadata} from "next";
import {CheckCircle2, Download, FileJson, FileText, RotateCcw, TriangleAlert} from "lucide-react";
import {getTranslations} from "next-intl/server";
import {ResultPreview} from "@/components/result-preview";
import {demoResult} from "@/lib/demo-result";

export const metadata: Metadata = {title: "Sample audit result"};

export default async function ResultsPage() {
  const t = await getTranslations("resultsPage");
  const findings = t.raw("findings") as string[];
  return (
    <div className="page-shell result-page">
      <div className="page-hero page-hero--center">
        <span className="eyebrow">{t("eyebrow")}</span>
        <h1>{t("title")}</h1>
        <p>{t("intro")}</p>
      </div>
      <ResultPreview />
      <div className="result-detail-grid">
        <section className="paper-card">
          <span className="kicker">{t("findingsKicker")}</span>
          <h2>{t("findingsTitle")}</h2>
          <div className="finding-list">
            {findings.map((finding, index) => <div key={finding}><span>{index === 0 ? <CheckCircle2 /> : <TriangleAlert />}</span><p>{finding}</p></div>)}
          </div>
        </section>
        <aside className="paper-card output-list">
          <span className="kicker">{t("packageKicker")}</span>
          <h2>{t("packageTitle")}</h2>
          <div><FileText /><span><strong>{t("brief")}</strong><small>PDF · HTML</small></span></div>
          <div><FileJson /><span><strong>{t("machine")}</strong><small>JSON · schema v1</small></span></div>
          <div><RotateCcw /><span><strong>{t("replay")}</strong><small>Manifest · checksums</small></span></div>
          <p className="notice"><Download /> {t("demoNotice")}</p>
        </aside>
      </div>
      <div className="editorial-note">
        <strong>{t("disclosureTitle")}</strong><p>{t("disclosure")}</p>
        <p><a href={demoResult.sources.paper} rel="noreferrer">{t("paperSource")}</a> · <a href={demoResult.sources.materials} rel="noreferrer">{t("materialsSource")}</a></p>
      </div>
    </div>
  );
}
