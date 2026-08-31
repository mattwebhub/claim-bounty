import {ArrowRight, Braces, CheckCircle2, FileCheck2, Fingerprint, FlaskConical, Layers3, LockKeyhole, SearchCheck, ShieldCheck} from "lucide-react";
import {getTranslations, setRequestLocale} from "next-intl/server";
import {Link} from "@/i18n/navigation";
import {ResultPreview} from "@/components/result-preview";

type Item = {title: string; body: string};

export default async function HomePage({params}: {params: Promise<{locale: string}>}) {
  const {locale} = await params;
  setRequestLocale(locale);
  const t = await getTranslations("landing");
  const steps = t.raw("steps") as Item[];
  const outputs = t.raw("outputs") as Item[];
  const principles = t.raw("principles") as Item[];
  const icons = [SearchCheck, FlaskConical, ShieldCheck];
  const outputIcons = [FileCheck2, Layers3, Braces];

  return (
    <>
      <section className="hero section-shell">
        <div className="hero-copy">
          <span className="eyebrow"><Fingerprint size={14} /> {t("eyebrow")}</span>
          <h1>{t("titleStart")} <em>{t("titleEmphasis")}</em> {t("titleEnd")}</h1>
          <p className="hero-lede">{t("lede")}</p>
          <div className="hero-actions">
            <Link href="/signup" className="button">{t("primaryCta")} <ArrowRight size={17} /></Link>
            <Link href="/results" className="button button--secondary">{t("secondaryCta")}</Link>
          </div>
          <div className="hero-assurances">
            <span><CheckCircle2 /> {t("assurance1")}</span>
            <span><CheckCircle2 /> {t("assurance2")}</span>
            <span><CheckCircle2 /> {t("assurance3")}</span>
          </div>
        </div>
        <div className="hero-result"><ResultPreview compact /></div>
      </section>

      <section className="signal-band" aria-label={t("signalLabel")}>
        <div><strong>{t("signal1Value")}</strong><span>{t("signal1Label")}</span></div>
        <div><strong>{t("signal2Value")}</strong><span>{t("signal2Label")}</span></div>
        <div><strong>{t("signal3Value")}</strong><span>{t("signal3Label")}</span></div>
        <div><strong>{t("signal4Value")}</strong><span>{t("signal4Label")}</span></div>
      </section>

      <section id="how" className="content-section section-shell">
        <div className="section-heading">
          <span className="kicker">{t("howKicker")}</span>
          <h2>{t("howTitle")}</h2>
          <p>{t("howBody")}</p>
        </div>
        <div className="steps-grid">
          {steps.map((step, index) => {
            const Icon = icons[index];
            return <article key={step.title} className="step-card"><div className="step-number">0{index + 1}</div><Icon /><h3>{step.title}</h3><p>{step.body}</p></article>;
          })}
        </div>
      </section>

      <section className="results-section">
        <div className="section-shell split-section">
          <div className="split-copy">
            <span className="kicker kicker--light">{t("resultKicker")}</span>
            <h2>{t("resultTitle")}</h2>
            <p>{t("resultBody")}</p>
            <ul>
              <li><CheckCircle2 /> {t("resultPoint1")}</li>
              <li><CheckCircle2 /> {t("resultPoint2")}</li>
              <li><CheckCircle2 /> {t("resultPoint3")}</li>
            </ul>
            <Link href="/results" className="light-link">{t("resultCta")} <ArrowRight size={16} /></Link>
          </div>
          <ResultPreview />
        </div>
      </section>

      <section className="content-section section-shell">
        <div className="section-heading section-heading--center">
          <span className="kicker">{t("outputsKicker")}</span>
          <h2>{t("outputsTitle")}</h2>
          <p>{t("outputsBody")}</p>
        </div>
        <div className="outputs-grid">
          {outputs.map((output, index) => {
            const Icon = outputIcons[index];
            return <article key={output.title}><Icon /><h3>{output.title}</h3><p>{output.body}</p></article>;
          })}
        </div>
      </section>

      <section className="trust-section">
        <div className="section-shell trust-grid">
          <div className="section-heading">
            <span className="kicker">{t("trustKicker")}</span>
            <h2>{t("trustTitle")}</h2>
            <p>{t("trustBody")}</p>
            <Link href="/methodology" className="text-link">{t("trustCta")} <ArrowRight size={15} /></Link>
          </div>
          <div className="principles-list">
            {principles.map((item, index) => (
              <article key={item.title}><span>0{index + 1}</span><div><h3>{item.title}</h3><p>{item.body}</p></div></article>
            ))}
          </div>
        </div>
      </section>

      <section className="cta-section section-shell">
        <div><span className="kicker kicker--light">{t("ctaKicker")}</span><h2>{t("ctaTitle")}</h2><p>{t("ctaBody")}</p></div>
        <Link href="/signup" className="button button--cream">{t("ctaButton")} <ArrowRight size={17} /></Link>
        <div className="cta-watermark" aria-hidden="true"><LockKeyhole /><span>P2P</span></div>
      </section>
    </>
  );
}
