const configuredSiteUrl = process.env.NEXT_PUBLIC_SITE_URL ?? "https://peer2paper.com";

export const siteUrl = new URL(configuredSiteUrl).origin;
export const legalEmail = process.env.NEXT_PUBLIC_LEGAL_EMAIL ?? "privacy@peer2paper.com";

export const localeNames = {
  en: "English",
  pt: "Português",
  es: "Español",
  fr: "Français",
  de: "Deutsch",
  it: "Italiano",
  nl: "Nederlands",
  ru: "Русский"
} as const;
