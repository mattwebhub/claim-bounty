import {getTranslations} from "next-intl/server";
import {CheckCircle2, LockKeyhole} from "lucide-react";
import {createAuditRequest} from "@/app/actions/audits";
import {SubmitButton} from "@/components/submit-button";
import type {Metadata} from "next";

export const metadata: Metadata = {robots: {index: false, follow: false}};

export default async function SubmitPage({params, searchParams}: {params: Promise<{locale: string}>; searchParams: Promise<{error?: string}>}) {
  const [{locale}, query, t] = await Promise.all([params, searchParams, getTranslations("submit")]);
  const requirements = t.raw("requirements") as string[];
  return <div className="submission-shell"><aside><span className="eyebrow eyebrow--dark">{t("eyebrow")}</span><h1>{t("title")}</h1><p>{t("intro")}</p><ul>{requirements.map((item) => <li key={item}><CheckCircle2/>{item}</li>)}</ul><div className="privacy-callout"><LockKeyhole/><p>{t("privacy")}</p></div></aside><section className="submission-card"><h2>{t("formTitle")}</h2><p>{t("formIntro")}</p>{query.error && <p className="form-message form-message--error">{t("error")}</p>}<form action={createAuditRequest} className="request-form"><input type="hidden" name="locale" value={locale}/><label>{t("projectTitle")}<input name="title" required minLength={3} placeholder={t("projectPlaceholder")}/></label><label>{t("claim")}<textarea name="claim" required minLength={20} rows={4} placeholder={t("claimPlaceholder")}/><small>{t("claimHelp")}</small></label><div className="form-row"><label>{t("paperUrl")}<input type="url" name="paper_url" placeholder="https://..."/></label><label>{t("materialsUrl")}<input type="url" name="materials_url" placeholder="https://..."/></label></div><label>{t("notes")}<textarea name="notes" rows={3} placeholder={t("notesPlaceholder")}/></label><SubmitButton idle={t("submitButton")} pending={t("submitting")}/><p className="form-legal">{t("agreement")}</p></form></section></div>;
}
