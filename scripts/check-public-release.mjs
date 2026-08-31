import {execFileSync} from "node:child_process";
import {existsSync, readFileSync, statSync} from "node:fs";
import {fileURLToPath} from "node:url";
import path from "node:path";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const legacyBrand = new RegExp("claim[\\s_-]*" + "bounty", "i");
const publicCommitEmail = /^noreply@peer2paper\.com$/i;
const privatePrefixes = [
  ".tmp/",
  "exchange/",
  ".toone/organization/.toone/",
  "peer2paper-routine-export/",
  "peer2paper/audits/",
  "peer2paper/assessment-bundles/",
  "peer2paper/state/",
  "peer2paper/validation/cases/",
  "peer2paper/validation/sources/"
];
const forbiddenNames = [
  legacyBrand,
  /hidden[-_ ]?answer[-_ ]?key/i,
  /source[-_ ]?archives?/i,
  /participant[-_ ]?data/i,
  /(^|\/)\.env$/i,
  /\.env\.(?!example$)/i,
  /\.(pdf|csv|zip|pem|p12|pfx|key|rds|rdata|sav|dta)$/i
];
const forbiddenContent = [
  legacyBrand,
  /-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----/,
  /\bAKIA[0-9A-Z]{16}\b/,
  /\bgh[ps]_[A-Za-z0-9]{30,}\b/,
  /\bsk_(?:live|test)_[A-Za-z0-9]{20,}\b/,
  /service_role\s*[=:]\s*["'][A-Za-z0-9._-]{20,}/i,
  new RegExp("\\/Users\\/" + "[^/\\s\"']+\\/"),
  new RegExp("external_release_" + "authorized[\\\"']?\\s*:\\s*false", "i"),
  new RegExp("release_" + "scope[\\\"']?\\s*:\\s*[\\\"']internal[\\\"']", "i")
];
const requiredFiles = [
  "README.md",
  "LICENSE",
  "SECURITY.md",
  "CONTRIBUTING.md",
  "CODE_OF_CONDUCT.md",
  ".github/workflows/ci.yml",
  ".github/dependabot.yml",
  "peer2paper/PUBLIC_RELEASE.md",
  "peer2paper/web/package-lock.json"
];

const files = listCandidateFiles();
const failures = [];

for (const required of requiredFiles) {
  if (!files.includes(required)) failures.push(`missing required public file: ${required}`);
}

for (const relative of files) {
  if (privatePrefixes.some((prefix) => relative === prefix.slice(0, -1) || relative.startsWith(prefix))) {
    failures.push(`private runtime path is tracked: ${relative}`);
    continue;
  }
  if (forbiddenNames.some((pattern) => pattern.test(relative))) {
    failures.push(`prohibited public filename: ${relative}`);
    continue;
  }
  const absolute = path.join(root, relative);
  if (statSync(absolute).size > 5_000_000) failures.push(`unexpected file over 5 MB: ${relative}`);
  if (isBinary(relative) || relative === "scripts/check-public-release.mjs") continue;
  const contents = readFileSync(absolute, "utf8");
  for (const pattern of forbiddenContent) {
    if (pattern.test(contents)) failures.push(`restricted or secret-like content in ${relative}: ${pattern}`);
  }
}

const history = scanReachableHistory();
failures.push(...history.failures);

if (failures.length) {
  console.error(failures.map((failure) => `- ${failure}`).join("\n"));
  process.exit(1);
}

console.log(
  `Peer2Paper public-release check passed: ${files.length} tracked and candidate files plus ${history.commits} reachable commits scanned`
);

function listCandidateFiles() {
  const output = execFileSync("git", ["ls-files", "--cached", "--others", "--exclude-standard", "-z"], {cwd: root});
  return output
    .toString("utf8")
    .split("\0")
    .filter((relative) => relative && existsSync(path.join(root, relative)))
    .sort();
}

function isBinary(filename) {
  return /\.(png|jpe?g|gif|webp|ico|woff2?|ttf)$/i.test(filename);
}

function scanReachableHistory() {
  const commits = execFileSync("git", ["rev-list", "--all"], {cwd: root, encoding: "utf8"})
    .split("\n")
    .filter(Boolean);
  const failures = [];
  const checkedBlobs = new Set();

  for (const commit of commits) {
    const commitMetadata = execFileSync(
      "git",
      ["show", "-s", "--format=%an%n%ae%n%cn%n%ce%n%s", commit],
      {cwd: root, encoding: "utf8"}
    );
    if (legacyBrand.test(commitMetadata)) {
      failures.push(`legacy brand exists in reachable commit metadata: ${commit}`);
    }
    const [authorName, authorEmail, committerName, committerEmail, subject] = commitMetadata.trimEnd().split("\n");
    const isSyntheticPullRequestMerge =
      process.env.GITHUB_ACTIONS === "true" &&
      process.env.GITHUB_EVENT_NAME === "pull_request" &&
      commit === execFileSync("git", ["rev-parse", "HEAD"], {cwd: root, encoding: "utf8"}).trim() &&
      committerName === "GitHub" &&
      committerEmail === "noreply@github.com" &&
      /^Merge [0-9a-f]{40} into [0-9a-f]{40}$/.test(subject) &&
      execFileSync("git", ["show", "-s", "--format=%P", commit], {cwd: root, encoding: "utf8"})
        .trim()
        .split(/\s+/).length === 2;
    if (
      !isSyntheticPullRequestMerge &&
      (!publicCommitEmail.test(authorEmail) || !publicCommitEmail.test(committerEmail))
    ) {
      failures.push(`non-public author or committer email exists in reachable commit metadata: ${commit}`);
    }
    const records = execFileSync("git", ["ls-tree", "-r", "-z", "--full-tree", commit], {
      cwd: root,
      maxBuffer: 20 * 1024 * 1024
    })
      .toString("utf8")
      .split("\0")
      .filter(Boolean);

    for (const record of records) {
      const separator = record.indexOf("\t");
      if (separator === -1) continue;
      const metadata = record.slice(0, separator).split(" ");
      const relative = record.slice(separator + 1);
      const blob = metadata[2];

      if (privatePrefixes.some((prefix) => relative === prefix.slice(0, -1) || relative.startsWith(prefix))) {
        failures.push(`private runtime path exists in reachable history: ${relative}`);
        continue;
      }
      if (forbiddenNames.some((pattern) => pattern.test(relative))) {
        failures.push(`prohibited filename exists in reachable history: ${relative}`);
        continue;
      }
      if (!blob || checkedBlobs.has(blob) || isBinary(relative)) continue;
      checkedBlobs.add(blob);

      const size = Number(execFileSync("git", ["cat-file", "-s", blob], {cwd: root, encoding: "utf8"}));
      if (size > 5_000_000) {
        failures.push(`unexpected blob over 5 MB in reachable history: ${relative}`);
        continue;
      }
      const contents = execFileSync("git", ["cat-file", "blob", blob], {
        cwd: root,
        encoding: "utf8",
        maxBuffer: 6 * 1024 * 1024
      });
      for (const pattern of forbiddenContent) {
        if (pattern.test(contents)) {
          failures.push(`restricted or secret-like content exists in reachable history: ${relative}: ${pattern}`);
        }
      }
    }
  }

  return {commits: commits.length, failures: [...new Set(failures)]};
}
