import type {Metadata} from "next";
import {NextIntlClientProvider, hasLocale} from "next-intl";
import {getMessages, getTranslations, setRequestLocale} from "next-intl/server";
import {notFound} from "next/navigation";
import {Inter, Instrument_Serif} from "next/font/google";
import {routing} from "@/i18n/routing";
import {siteUrl} from "@/lib/site";
import {Header} from "@/components/header";
import {Footer} from "@/components/footer";
import "../globals.css";

const inter = Inter({subsets: ["latin", "cyrillic"], variable: "--font-sans"});
const serif = Instrument_Serif({weight: "400", subsets: ["latin"], variable: "--font-serif"});

export function generateStaticParams() {
  return routing.locales.map((locale) => ({locale}));
}

export async function generateMetadata({params}: {params: Promise<{locale: string}>}): Promise<Metadata> {
  const {locale} = await params;
  if (!hasLocale(routing.locales, locale)) return {};
  const t = await getTranslations({locale, namespace: "meta"});
  const languages = Object.fromEntries(routing.locales.map((item) => [item, `${siteUrl}/${item}`]));
  return {
    metadataBase: new URL(siteUrl),
    title: {default: t("title"), template: `%s · Peer2Paper`},
    description: t("description"),
    alternates: {canonical: `${siteUrl}/${locale}`, languages: {...languages, "x-default": `${siteUrl}/en`}},
    openGraph: {title: t("title"), description: t("description"), type: "website", siteName: "Peer2Paper", locale},
    twitter: {card: "summary_large_image", title: t("title"), description: t("description")},
    robots: {index: true, follow: true}
  };
}

export default async function LocaleLayout({children, params}: Readonly<{children: React.ReactNode; params: Promise<{locale: string}>}>) {
  const {locale} = await params;
  if (!hasLocale(routing.locales, locale)) notFound();
  setRequestLocale(locale);
  const messages = await getMessages();
  return (
    <html lang={locale} className={`${inter.variable} ${serif.variable}`}>
      <body>
        <NextIntlClientProvider messages={messages}>
          <a className="skip-link" href="#main">Skip to content</a>
          <Header />
          <main id="main">{children}</main>
          <Footer />
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
