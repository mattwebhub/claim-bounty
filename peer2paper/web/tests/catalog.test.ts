import {describe, expect, it} from "vitest";
import {readFileSync} from "node:fs";
import en from "@/messages/en.json";
import {locales} from "@/i18n/routing";
import {demoResult} from "@/lib/demo-result";
import {siteUrl} from "@/lib/site";

describe("public content contracts", () => {
  it("offers the same eight locales as the reference landing page", () => {
    expect(locales).toEqual(["en", "pt", "es", "fr", "de", "it", "nl", "ru"]);
    expect(en.meta.title).toContain("Peer2Paper");
    expect(siteUrl).toBe("https://peer2paper.com");
  });

  it("keeps the result demo explicitly evidence-rich", () => {
    expect(demoResult.outputs).toContain("Machine-readable JSON");
    expect(demoResult.findings.length).toBeGreaterThanOrEqual(3);
    expect(en.resultsPage.intro).toContain("sanitized");
  });

  it("keeps the web preview aligned with the reviewed public case study", () => {
    const caseStudy = JSON.parse(
      readFileSync(new URL("../../examples/elite-party-cues/result.json", import.meta.url), "utf8")
    ) as {
      verdict: string;
      source_report: {estimate: number; p_value: number};
      robustness_comparison: {adjusted_hc2: {estimate: number; p_value: number}};
    };

    expect(caseStudy.verdict).toContain("source_report_verified");
    expect(Number(demoResult.reported.estimate)).toBe(caseStudy.source_report.estimate);
    expect(Number(demoResult.reported.pValue)).toBe(caseStudy.source_report.p_value);
    expect(Number(demoResult.audited.estimate)).toBe(caseStudy.robustness_comparison.adjusted_hc2.estimate);
    expect(Number(demoResult.audited.pValue)).toBe(caseStudy.robustness_comparison.adjusted_hc2.p_value);
  });
});
