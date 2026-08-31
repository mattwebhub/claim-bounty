import {getTranslations} from "next-intl/server";
import {requestPasswordReset} from "@/app/actions/auth";
import {SubmitButton} from "@/components/submit-button";
import {Link} from "@/i18n/navigation";
import type {Metadata} from "next";

export const metadata: Metadata = {robots: {index: false, follow: false}};

export default async function Page({params, searchParams}: {params: Promise<{locale: string}>; searchParams: Promise<{error?: string; notice?: string}>}) {
  const [{locale}, query, t] = await Promise.all([params, searchParams, getTranslations("auth")]);
  return <div className="form-page"><div className="simple-form-card"><span className="eyebrow">{t("recovery")}</span><h1>{t("forgotTitle")}</h1><p>{t("forgotIntro")}</p>{query.notice && <p className="form-message form-message--success">{t("checkEmail")}</p>}{query.error && <p className="form-message form-message--error">{t("errors.generic")}</p>}<form action={requestPasswordReset} className="auth-form"><input type="hidden" name="locale" value={locale}/><label>{t("email")}<input type="email" name="email" required autoComplete="email" placeholder="you@example.com"/></label><SubmitButton idle={t("resetButton")} pending={t("working")}/></form><Link href="/signin" className="text-link">{t("backSignin")}</Link></div></div>;
}
