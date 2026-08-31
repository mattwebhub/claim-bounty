import {getTranslations} from "next-intl/server";
import {Menu} from "lucide-react";
import {Link} from "@/i18n/navigation";
import {Brand} from "./brand";
import {LocaleSwitcher} from "./locale-switcher";

export async function Header() {
  const t = await getTranslations("nav");
  return (
    <header className="site-header">
      <div className="header-inner">
        <Brand />
        <nav className="desktop-nav" aria-label={t("primaryLabel")}>
          <Link href="/#how">{t("how")}</Link>
          <Link href="/results">{t("results")}</Link>
          <Link href="/methodology">{t("methodology")}</Link>
          <Link href="/docs">{t("docs")}</Link>
        </nav>
        <div className="header-actions">
          <LocaleSwitcher />
          <Link href="/signin" className="text-link header-signin">{t("signin")}</Link>
          <Link href="/signup" className="button button--small">{t("start")}</Link>
          <details className="mobile-menu">
            <summary aria-label={t("menu")}><Menu size={20} /></summary>
            <nav aria-label={t("mobileLabel")}>
              <Link href="/#how">{t("how")}</Link>
              <Link href="/results">{t("results")}</Link>
              <Link href="/methodology">{t("methodology")}</Link>
              <Link href="/docs">{t("docs")}</Link>
              <Link href="/signin">{t("signin")}</Link>
              <Link href="/signup">{t("start")}</Link>
            </nav>
          </details>
        </div>
      </div>
    </header>
  );
}
