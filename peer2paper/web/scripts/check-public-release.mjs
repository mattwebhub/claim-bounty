import {readFileSync, readdirSync, statSync} from "node:fs";
import {fileURLToPath} from "node:url";
import path from "node:path";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const ignoredDirectories = new Set([".git", ".next", "node_modules", "coverage"]);
const forbiddenNames = [
  /hidden[-_ ]?answer[-_ ]?key/i,
  /source[-_ ]?archives?/i,
  /assessment[-_ ]?bundles?/i,
  /participant[-_ ]?data/i,
  /\.env\.(?!example$)/i,
  /(^|\/)\.env$/i,
  /\.(pem|p12|pfx|key|rds|rdata|sav|dta)$/i
];
const secretPatterns = [
  /-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----/,
  /\bAKIA[0-9A-Z]{16}\b/,
  /\bgh[ps]_[A-Za-z0-9]{30,}\b/,
  /\bsk_(?:live|test)_[A-Za-z0-9]{20,}\b/,
  /service_role\s*[=:]\s*["'][A-Za-z0-9._-]{20,}/i,
  new RegExp("external_release_" + "authorized[\\\"']?\\s*:\\s*false", "i"),
  new RegExp("release_" + "scope[\\\"']?\\s*:\\s*[\\\"']internal[\\\"']", "i")
];
const legacyBrand = new RegExp("claim" + "[- ]?bounty", "i");
const requiredFiles = [
  "LICENSE",
  "README.md",
  "SECURITY.md",
  "CONTRIBUTING.md",
  "CODE_OF_CONDUCT.md",
  ".env.example",
  "package-lock.json",
  "supabase/migrations/202608310001_create_audit_requests.sql"
];

const files = walk(root);
const failures = [];

for (const required of requiredFiles) {
  if (!files.includes(required)) failures.push(`missing required release file: ${required}`);
}

for (const relative of files) {
  if (forbiddenNames.some((pattern) => pattern.test(relative))) {
    failures.push(`forbidden release filename: ${relative}`);
    continue;
  }
  const absolute = path.join(root, relative);
  if (statSync(absolute).size > 5_000_000) failures.push(`unexpected file over 5 MB: ${relative}`);
  if (isBinary(relative) || relative === "scripts/check-public-release.mjs") continue;
  const contents = readFileSync(absolute, "utf8");
  if (legacyBrand.test(contents)) failures.push(`legacy product brand in ${relative}`);
  for (const pattern of secretPatterns) {
    if (pattern.test(contents)) failures.push(`restricted or secret-like content in ${relative}: ${pattern}`);
  }
}

if (failures.length) {
  console.error(failures.map((failure) => `- ${failure}`).join("\n"));
  process.exit(1);
}

console.log(`public-release check passed: ${files.length} public web repository files scanned`);

function walk(directory, prefix = "") {
  return readdirSync(directory, {withFileTypes: true}).flatMap((entry) => {
    if (entry.name === ".DS_Store" || entry.name.endsWith(".tsbuildinfo")) return [];
    if (entry.isDirectory() && ignoredDirectories.has(entry.name)) return [];
    const relative = path.posix.join(prefix, entry.name);
    return entry.isDirectory() ? walk(path.join(directory, entry.name), relative) : [relative];
  });
}

function isBinary(filename) {
  return /\.(png|jpe?g|gif|webp|ico|woff2?|ttf)$/i.test(filename);
}
