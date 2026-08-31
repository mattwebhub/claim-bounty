import {getTranslations} from "next-intl/server";

export async function InfoPage({kind}: {kind: "about" | "security" | "privacy" | "terms"}) {
  const t = await getTranslations(`info.${kind}`);
  const sections = t.raw("sections") as {title: string; body: string}[];
  return <div className="page-shell prose-page"><div className="page-hero"><span className="eyebrow">{t("eyebrow")}</span><h1>{t("title")}</h1><p>{t("intro")}</p></div><div className="prose-sections">{sections.map((section) => <section key={section.title}><h2>{section.title}</h2><p>{section.body}</p></section>)}</div></div>;
}
