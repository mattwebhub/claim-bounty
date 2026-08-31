import {ArrowUpRight, Check, FileJson, FileText, RotateCcw, ShieldCheck} from "lucide-react";
import {getTranslations} from "next-intl/server";
import {Link} from "@/i18n/navigation";
import {demoResult} from "@/lib/demo-result";

export async function ResultPreview({compact = false}: {compact?: boolean}) {
  const t = await getTranslations("result");
  return (
    <article className={`result-card ${compact ? "result-card--compact" : ""}`}>
      <div className="result-card__top">
        <span className="eyebrow eyebrow--green"><Check size={13} /> {t("completed")}</span>
        <span className="mono">{demoResult.id}</span>
      </div>
      <div className="result-verdict">
        <div className="verdict-orbit" aria-hidden="true"><ShieldCheck /></div>
        <div>
          <span>{t("verdict")}</span>
          <h3>{t("verdictValue")}</h3>
          <p>{t("confidence")}: <strong>{t("high")}</strong></p>
        </div>
      </div>
      <p className="result-claim">“{t("claim") }”</p>
      <div className="comparison" aria-label={t("comparisonLabel")}>
        <div><span>{t("metric")}</span><strong>{t("estimate")}</strong><strong>{t("interval")}</strong><strong>p</strong></div>
        <div><span>{t("reported")}</span><strong>{demoResult.reported.estimate}</strong><strong>{demoResult.reported.interval}</strong><strong>{demoResult.reported.pValue}</strong></div>
        <div className="comparison__accent"><span>{t("audited")}</span><strong>{demoResult.audited.estimate}</strong><strong>{demoResult.audited.interval}</strong><strong>{demoResult.audited.pValue}</strong></div>
      </div>
      {!compact && (
        <div className="artifact-row">
          <span><FileText /> {t("brief")}</span>
          <span><FileJson /> JSON</span>
          <span><RotateCcw /> {t("replay")}</span>
        </div>
      )}
      <Link href="/results" className="result-link">{t("explore")} <ArrowUpRight size={16} /></Link>
    </article>
  );
}
