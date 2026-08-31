import {getTranslations} from "next-intl/server";
import {ArrowRight, CircleDot, Clock3, FileSearch, Settings2} from "lucide-react";
import {Link} from "@/i18n/navigation";
import {createClient} from "@/lib/supabase/server";
import {isSupabaseConfigured} from "@/lib/supabase/config";
import {signOut} from "@/app/actions/auth";
import type {AuditRequest} from "@/types/database";
import type {Metadata} from "next";

export const metadata: Metadata = {robots: {index: false, follow: false}};

export default async function Dashboard({params, searchParams}: {params: Promise<{locale: string}>; searchParams: Promise<{notice?: string}>}) {
  const [{locale}, query, t] = await Promise.all([params, searchParams, getTranslations("dashboard")]);
  if (!isSupabaseConfigured()) return <div className="page-shell"><div className="config-notice"><Settings2/><h1>{t("setupTitle")}</h1><p>{t("setupBody")}</p><code>cp .env.example .env.local</code></div></div>;
  const supabase = await createClient();
  const {data: {user}} = await supabase.auth.getUser();
  const {data} = user ? await supabase.from("audit_requests").select("*").order("created_at", {ascending: false}) : {data: []};
  const audits = (data ?? []) as AuditRequest[];
  return <div className="dashboard-shell"><aside className="dashboard-sidebar"><strong>Peer2Paper</strong><nav><Link href="/dashboard"><FileSearch/> {t("audits")}</Link><Link href="/submit"><CircleDot/> {t("newRequest")}</Link></nav><form action={signOut}><input type="hidden" name="locale" value={locale}/><button>{t("signout")}</button></form></aside><div className="dashboard-main"><header><div><span className="kicker">{t("workspace")}</span><h1>{t("title")}</h1><p>{user?.email}</p></div><Link href="/submit" className="button">{t("newRequest")} <ArrowRight size={16}/></Link></header>{query.notice && <p className="form-message form-message--success">{t(`notices.${query.notice === "request_created" ? "requestCreated" : "passwordUpdated"}`)}</p>}{audits.length === 0 ? <div className="empty-state"><FileSearch/><h2>{t("emptyTitle")}</h2><p>{t("emptyBody")}</p><Link href="/submit" className="button button--secondary">{t("emptyCta")}</Link></div> : <div className="audit-list">{audits.map((audit) => <article key={audit.id}><div><span className={`status status--${audit.status}`}>{t(`statuses.${audit.status}`)}</span><h2>{audit.title}</h2><p>{audit.claim}</p></div><span className="audit-date"><Clock3/>{new Intl.DateTimeFormat(locale, {dateStyle: "medium"}).format(new Date(audit.created_at))}</span></article>)}</div>}</div></div>;
}
