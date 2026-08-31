import {getRequestConfig} from "next-intl/server";
import {hasLocale} from "next-intl";
import {routing} from "./routing";

export default getRequestConfig(async ({requestLocale}) => {
  const requested = await requestLocale;
  const locale = hasLocale(routing.locales, requested) ? requested : routing.defaultLocale;
  const base = (await import("../messages/en.json")).default;
  const translated = locale === "en" ? base : (await import(`../messages/${locale}.json`)).default;
  return {locale, messages: deepMerge(base, translated)};
});

function deepMerge(base: Record<string, unknown>, override: Record<string, unknown>): Record<string, unknown> {
  const merged = {...base};
  for (const [key, value] of Object.entries(override)) {
    const current = merged[key];
    merged[key] = isObject(current) && isObject(value) ? deepMerge(current, value) : value;
  }
  return merged;
}

function isObject(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}
