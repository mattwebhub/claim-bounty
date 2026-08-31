import {spawn} from "node:child_process";

const port = 3107;
const origin = `http://127.0.0.1:${port}`;
const server = spawn(process.execPath, ["node_modules/next/dist/bin/next", "start", "-p", String(port)], {
  stdio: ["ignore", "pipe", "pipe"],
  env: {...process.env, NEXT_PUBLIC_SITE_URL: origin}
});

let logs = "";
server.stdout.on("data", (chunk) => { logs += chunk; });
server.stderr.on("data", (chunk) => { logs += chunk; });

try {
  await waitUntilReady();
  await expectRedirect("/", "/en");
  await expectPage("/en", "Know whether a claim");
  await expectPage("/pt", "Saiba se uma alegação");
  await expectPage("/ru", "Узнайте, выдерживает ли утверждение");
  await expectPage("/en/results", "See what a Peer2Paper result looks like");
  await expectPage("/en/docs", "From request to audit package");
  await expectPage("/en/signin", "Continue with Google", true);
  await expectPage("/en/dashboard", "Connect Supabase to enable the workspace", true);
  await expectPage("/robots.txt", "Sitemap:");
  await expectPage("/sitemap.xml", "/pt/results");
  console.log("runtime smoke check passed: locale, content, auth and metadata routes");
} finally {
  server.kill("SIGTERM");
  await Promise.race([
    new Promise((resolve) => server.once("exit", resolve)),
    new Promise((resolve) => setTimeout(resolve, 2_000))
  ]);
}

async function waitUntilReady() {
  for (let attempt = 0; attempt < 50; attempt += 1) {
    try {
      const response = await fetch(`${origin}/en`);
      if (response.ok) return;
    } catch {}
    await new Promise((resolve) => setTimeout(resolve, 200));
  }
  throw new Error(`Next.js did not become ready\n${logs}`);
}

async function expectRedirect(route, destination) {
  const response = await fetch(`${origin}${route}`, {redirect: "manual"});
  if (![307, 308].includes(response.status) || response.headers.get("location") !== destination) {
    throw new Error(`${route}: expected redirect to ${destination}, got ${response.status} ${response.headers.get("location")}`);
  }
  expectSecurityHeaders(route, response);
}

async function expectPage(route, text, noIndex = false) {
  const response = await fetch(`${origin}${route}`);
  const body = await response.text();
  if (!response.ok) throw new Error(`${route}: expected 2xx, got ${response.status}`);
  if (!body.includes(text)) throw new Error(`${route}: expected response to contain ${JSON.stringify(text)}`);
  if (noIndex && !body.includes('name="robots" content="noindex, nofollow"')) {
    throw new Error(`${route}: expected noindex metadata`);
  }
  expectSecurityHeaders(route, response);
}

function expectSecurityHeaders(route, response) {
  const expected = {
    "content-security-policy": "default-src 'self'",
    "x-content-type-options": "nosniff",
    "x-frame-options": "DENY",
    "referrer-policy": "strict-origin-when-cross-origin"
  };
  for (const [header, value] of Object.entries(expected)) {
    const actual = response.headers.get(header);
    const matches = header === "content-security-policy" ? actual?.startsWith(value) : actual === value;
    if (!matches) throw new Error(`${route}: missing ${header}: ${value}`);
  }
}
