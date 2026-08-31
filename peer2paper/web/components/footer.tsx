import {getTranslations} from "next-intl/server";
import {Link} from "@/i18n/navigation";
import {Brand} from "./brand";
import {LocaleSwitcher} from "./locale-switcher";
import {legalEmail} from "@/lib/site";

export async function Footer() {
  const t = await getTranslations("footer");
  return (
    <footer className="site-footer">
      <div className="footer-top">
        <div className="footer-intro">
          <Brand inverse />
          <p>{t("tagline")}</p>
          <LocaleSwitcher dark />
        </div>
        <div className="footer-column">
          <h2>{t("product")}</h2>
          <Link href="/#how">{t("how")}</Link>
          <Link href="/results">{t("sample")}</Link>
          <Link href="/methodology">{t("methodology")}</Link>
          <Link href="/signup">{t("request")}</Link>
        </div>
        <div className="footer-column">
          <h2>{t("resources")}</h2>
          <Link href="/docs">{t("docs")}</Link>
          <Link href="/docs#outputs">{t("outputs")}</Link>
          <Link href="/security">{t("security")}</Link>
          <a href={`mailto:${legalEmail}`}>{t("contact")}</a>
        </div>
        <div className="footer-column">
          <h2>{t("company")}</h2>
          <Link href="/about">{t("about")}</Link>
          <Link href="/privacy">{t("privacy")}</Link>
          <Link href="/terms">{t("terms")}</Link>
          <Link href="/signin">{t("signin")}</Link>
        </div>
      </div>
      <div className="footer-bottom">
        <span>© {new Date().getFullYear()} Peer2Paper. {t("rights")}</span>
        <span>{t("independence")}</span>
      </div>
    </footer>
  );
}
