"use client";

import {useLocale} from "next-intl";
import {usePathname, useRouter} from "@/i18n/navigation";
import {localeNames} from "@/lib/site";
import type {Locale} from "@/i18n/routing";

export function LocaleSwitcher({dark = false}: {dark?: boolean}) {
  const locale = useLocale() as Locale;
  const pathname = usePathname();
  const router = useRouter();

  return (
    <label className={`locale-picker ${dark ? "locale-picker--dark" : ""}`}>
      <span className="sr-only">Language</span>
      <select
        value={locale}
        aria-label="Language"
        onChange={(event) => router.replace(pathname, {locale: event.target.value as Locale})}
      >
        {Object.entries(localeNames).map(([value, label]) => (
          <option key={value} value={value}>{label}</option>
        ))}
      </select>
    </label>
  );
}
