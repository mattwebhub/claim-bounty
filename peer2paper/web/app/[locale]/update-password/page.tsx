import {getTranslations} from "next-intl/server";
import {updatePassword} from "@/app/actions/auth";
import {SubmitButton} from "@/components/submit-button";
import type {Metadata} from "next";

export const metadata: Metadata = {robots: {index: false, follow: false}};

export default async function Page({params}: {params: Promise<{locale: string}>}) {
  const [{locale}, t] = await Promise.all([params, getTranslations("auth")]);
  return <div className="form-page"><div className="simple-form-card"><span className="eyebrow">{t("recovery")}</span><h1>{t("updateTitle")}</h1><p>{t("updateIntro")}</p><form action={updatePassword} className="auth-form"><input type="hidden" name="locale" value={locale}/><label>{t("newPassword")}<input type="password" name="password" minLength={8} maxLength={72} required autoComplete="new-password"/></label><SubmitButton idle={t("updateButton")} pending={t("working")}/></form></div></div>;
}
