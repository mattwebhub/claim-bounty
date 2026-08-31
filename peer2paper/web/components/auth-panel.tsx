import {getTranslations} from "next-intl/server";
import {Link} from "@/i18n/navigation";
import {signIn, signInWithGoogle, signUp} from "@/app/actions/auth";
import {SubmitButton} from "./submit-button";

type Props = {mode: "signin" | "signup"; locale: string; error?: string; notice?: string; next?: string};

export async function AuthPanel({mode, locale, error, notice, next}: Props) {
  const t = await getTranslations("auth");
  const action = mode === "signin" ? signIn : signUp;
  const errorKeys = ["not_configured", "missing_fields", "invalid_credentials", "invalid_signup", "already_exists", "signup_failed", "oauth_failed", "callback_failed"] as const;
  const safeError = errorKeys.includes(error as typeof errorKeys[number]) ? error as typeof errorKeys[number] : "generic";
  return (
    <div className="auth-shell">
      <section className="auth-aside">
        <span className="eyebrow eyebrow--dark">{t("asideEyebrow")}</span>
        <h1>{mode === "signin" ? t("signinTitle") : t("signupTitle")}</h1>
        <p>{mode === "signin" ? t("signinIntro") : t("signupIntro")}</p>
        <blockquote>“{t("quote")}”<cite>{t("quoteBy")}</cite></blockquote>
      </section>
      <section className="auth-card">
        <div>
          <h2>{mode === "signin" ? t("welcome") : t("create")}</h2>
          <p>{mode === "signin" ? t("noAccount") : t("hasAccount")} <Link href={mode === "signin" ? "/signup" : "/signin"}>{mode === "signin" ? t("signupLink") : t("signinLink")}</Link></p>
        </div>
        {error && <p className="form-message form-message--error" role="alert">{t(`errors.${safeError}`)}</p>}
        {notice === "check_email" && <p className="form-message form-message--success">{t("checkEmail")}</p>}
        <form action={signInWithGoogle}>
          <input type="hidden" name="locale" value={locale} />
          <button className="google-button" type="submit"><GoogleIcon /> {t("google")}</button>
        </form>
        <div className="form-divider"><span>{t("or")}</span></div>
        <form action={action} className="auth-form">
          <input type="hidden" name="locale" value={locale} />
          <input type="hidden" name="next" value={next ?? ""} />
          {mode === "signup" && <label>{t("name")}<input name="name" autoComplete="name" placeholder={t("namePlaceholder")} /></label>}
          <label>{t("email")}<input type="email" name="email" autoComplete="email" required placeholder="you@example.com" /></label>
          <label>{t("password")}<input type="password" name="password" minLength={8} maxLength={72} autoComplete={mode === "signin" ? "current-password" : "new-password"} required placeholder={t("passwordPlaceholder")} /></label>
          {mode === "signin" && <Link className="forgot-link" href="/forgot-password">{t("forgot")}</Link>}
          <SubmitButton idle={mode === "signin" ? t("signinButton") : t("signupButton")} pending={t("working")} />
        </form>
        <p className="form-legal">{t("legalStart")} <Link href="/terms">{t("terms")}</Link> {t("and")} <Link href="/privacy">{t("privacy")}</Link>.</p>
      </section>
    </div>
  );
}

function GoogleIcon() {
  return <svg viewBox="0 0 24 24" aria-hidden="true"><path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.32v2.77h3.57c2.08-1.92 3.27-4.74 3.27-8.1Z"/><path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.24 1.06-3.71 1.06-2.86 0-5.29-1.94-6.16-4.54H2.18v2.84A11 11 0 0 0 12 23Z"/><path fill="#FBBC05" d="M5.84 14.09A6.6 6.6 0 0 1 5.5 12c0-.73.13-1.43.34-2.09V7.07H2.18A11 11 0 0 0 1 12c0 1.77.42 3.44 1.18 4.93l3.66-2.84Z"/><path fill="#EA4335" d="M12 5.37c1.62 0 3.06.56 4.2 1.64l3.15-3.15A10.55 10.55 0 0 0 12 1a11 11 0 0 0-9.82 6.07l3.66 2.84c.87-2.6 3.3-4.54 6.16-4.54Z"/></svg>;
}
