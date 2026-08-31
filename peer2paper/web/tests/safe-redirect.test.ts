import {describe, expect, it} from "vitest";
import {authCallbackUrl, safeLocalPath} from "@/lib/safe-redirect";
import {siteUrl} from "@/lib/site";

describe("authentication redirects", () => {
  it("keeps valid destinations on a normalized local path", () => {
    expect(safeLocalPath("/pt/dashboard?view=active#latest")).toBe("/pt/dashboard?view=active#latest");
    expect(safeLocalPath("/en/section/../dashboard")).toBe("/en/dashboard");
  });

  it.each([
    "https://evil.example/",
    "//evil.example/",
    String.raw`/\evil.example/`,
    "/%5cevil.example/",
    "/%2f%2fevil.example/",
    "/en/dashboard\nSet-Cookie: unsafe=true",
    "dashboard"
  ])("rejects unsafe destination %s", (destination) => {
    expect(safeLocalPath(destination, "/fr/dashboard")).toBe("/fr/dashboard");
  });

  it("uses the configured canonical origin for OAuth and email callbacks", () => {
    const callback = new URL(authCallbackUrl("/nl/dashboard"));
    expect(callback.origin).toBe(new URL(siteUrl).origin);
    expect(callback.pathname).toBe("/auth/callback");
    expect(callback.searchParams.get("next")).toBe("/nl/dashboard");
  });
});
