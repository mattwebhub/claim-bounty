import type {Metadata} from "next";
import {getTranslations} from "next-intl/server";

export const metadata: Metadata = {title: "Methodology"};

export default async function MethodologyPage() {
  const t = await getTranslations("methodologyPage");
  const phases = t.raw("phases") as {title: string; body: string; evidence: string}[];
  return <div className="page-shell prose-page"><div className="page-hero"><span className="eyebrow">{t("eyebrow")}</span><h1>{t("title")}</h1><p>{t("intro")}</p></div><div className="methodology-rail">{phases.map((phase, index) => <section key={phase.title}><span>{index + 1}</span><div><h2>{phase.title}</h2><p>{phase.body}</p><small>{phase.evidence}</small></div></section>)}</div><section className="editorial-note"><h2>{t("limitsTitle")}</h2><p>{t("limitsBody")}</p></section></div>;
}
