import {readFileSync} from "node:fs";
import {fileURLToPath} from "node:url";
import path from "node:path";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const locales = ["en", "pt", "es", "fr", "de", "it", "nl", "ru"];

const catalogs = Object.fromEntries(
  locales.map((locale) => [locale, JSON.parse(readFileSync(path.join(root, "messages", `${locale}.json`), "utf8"))])
);

const englishPaths = catalogPaths(catalogs.en);
const expectedPaths = [...englishPaths].sort();
for (const locale of locales) {
  const actualPaths = catalogPaths(catalogs[locale]);
  const missing = expectedPaths.filter((keyPath) => !actualPaths.has(keyPath));
  const extra = [...actualPaths].sort().filter((keyPath) => !englishPaths.has(keyPath));

  if (missing.length || extra.length) {
    const details = [
      missing.length ? `missing: ${missing.join(", ")}` : "",
      extra.length ? `extra: ${extra.join(", ")}` : ""
    ].filter(Boolean);
    throw new Error(`${locale}: catalog does not match en (${details.join("; ")})`);
  }
}

console.log(`i18n check passed: ${locales.length} complete locale catalogs and ${expectedPaths.length} values per locale`);

function catalogPaths(value, keyPath = "", paths = new Set()) {
  if (Array.isArray(value)) {
    if (value.length === 0) throw new Error(`${keyPath || "catalog"}: arrays must not be empty`);
    value.forEach((item, index) => catalogPaths(item, `${keyPath}[${index}]`, paths));
    return paths;
  }

  if (value && typeof value === "object") {
    const entries = Object.entries(value);
    if (entries.length === 0) throw new Error(`${keyPath || "catalog"}: objects must not be empty`);
    for (const [key, child] of entries) {
      catalogPaths(child, keyPath ? `${keyPath}.${key}` : key, paths);
    }
    return paths;
  }

  if (typeof value !== "string" || value.trim() === "") {
    throw new Error(`${keyPath || "catalog"}: translations must be non-empty strings`);
  }
  paths.add(keyPath);
  return paths;
}
