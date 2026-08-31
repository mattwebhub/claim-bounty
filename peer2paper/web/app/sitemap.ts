import type {MetadataRoute} from "next";
import {locales} from "@/i18n/routing";
import {siteUrl} from "@/lib/site";
const pages = ["", "/results", "/methodology", "/docs", "/about", "/security", "/privacy", "/terms"];
export default function sitemap(): MetadataRoute.Sitemap { return locales.flatMap((locale) => pages.map((path) => ({url: `${siteUrl}/${locale}${path}`, lastModified: new Date(), changeFrequency: path ? "monthly" as const : "weekly" as const, priority: path ? 0.7 : 1}))); }
