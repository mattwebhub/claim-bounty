import {defineRouting} from "next-intl/routing";

export const locales = ["en", "pt", "es", "fr", "de", "it", "nl", "ru"] as const;
export type Locale = (typeof locales)[number];

export const routing = defineRouting({
  locales,
  defaultLocale: "en",
  localePrefix: "always",
  localeDetection: true
});
