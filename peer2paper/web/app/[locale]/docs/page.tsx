import type {Metadata} from "next";
import {Archive, ArrowRight, Braces, CheckCircle2, FileText, PackageCheck, RotateCcw} from "lucide-react";
import {getTranslations} from "next-intl/server";
import {Link} from "@/i18n/navigation";

export const metadata: Metadata = {title: "Documentation"};

export default async function DocsPage() {
  const t = await getTranslations("docsPage");
  const inputs = t.raw("inputs") as string[];
  const stages = t.raw("stages") as {title: string; body: string}[];
  return (
    <div className="page-shell docs-layout">
      <aside className="docs-nav"><strong>{t("contents")}</strong><a href="#start">{t("start")}</a><a href="#workflow">{t("workflow")}</a><a href="#outputs">{t("outputs")}</a><a href="#statuses">{t("statuses")}</a><a href="#privacy">{t("privacy")}</a></aside>
      <article className="docs-content">
        <div className="page-hero" id="start"><span className="eyebrow">{t("eyebrow")}</span><h1>{t("title")}</h1><p>{t("intro")}</p></div>
        <section><h2>{t("beforeTitle")}</h2><p>{t("beforeBody")}</p><ul className="check-list">{inputs.map((item) => <li key={item}><CheckCircle2 /> {item}</li>)}</ul></section>
        <section id="workflow"><h2>{t("workflowTitle")}</h2><div className="doc-stages">{stages.map((stage, index) => <div key={stage.title}><span>0{index + 1}</span><div><h3>{stage.title}</h3><p>{stage.body}</p></div></div>)}</div></section>
        <section id="outputs"><h2>{t("outputsTitle")}</h2><p>{t("outputsBody")}</p><div className="doc-output-grid"><div><FileText /><h3>{t("briefTitle")}</h3><p>{t("briefBody")}</p></div><div><Braces /><h3>{t("jsonTitle")}</h3><p>{t("jsonBody")}</p></div><div><Archive /><h3>{t("evidenceTitle")}</h3><p>{t("evidenceBody")}</p></div><div><RotateCcw /><h3>{t("replayTitle")}</h3><p>{t("replayBody")}</p></div></div></section>
        <section id="statuses"><h2>{t("statusesTitle")}</h2><p>{t("statusesBody")}</p><div className="status-table"><div><strong>{t("statusReproduced")}</strong><span>{t("statusReproducedBody")}</span></div><div><strong>{t("statusPartial")}</strong><span>{t("statusPartialBody")}</span></div><div><strong>{t("statusNot")}</strong><span>{t("statusNotBody")}</span></div></div></section>
        <section id="privacy"><h2>{t("privacyTitle")}</h2><p>{t("privacyBody")}</p><Link href="/security" className="text-link">{t("privacyCta")} <ArrowRight size={15} /></Link></section>
        <div className="docs-cta"><PackageCheck /><div><h2>{t("readyTitle")}</h2><p>{t("readyBody")}</p></div><Link href="/signup" className="button">{t("readyCta")}</Link></div>
      </article>
    </div>
  );
}
