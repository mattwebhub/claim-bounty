#!/usr/bin/env node

import { readFileSync } from 'node:fs';

const reportPath = process.argv[2];

if (!reportPath) {
  console.error('usage: validate-ai-review.mjs <report.json>');
  process.exit(2);
}

let report;
try {
  report = JSON.parse(readFileSync(reportPath, 'utf8'));
} catch (error) {
  console.error(`ai-review: invalid JSON report: ${error.message}`);
  process.exit(2);
}

const severities = new Set(['blocker', 'high', 'medium', 'low']);
const requiredStrings = ['rule_id', 'file', 'title', 'evidence', 'remediation'];

if (
  !report ||
  !['pass', 'fail'].includes(report.verdict) ||
  typeof report.summary !== 'string' ||
  report.summary.length === 0 ||
  !Array.isArray(report.findings)
) {
  console.error('ai-review: report does not satisfy the local review contract');
  process.exit(2);
}

for (const [index, finding] of report.findings.entries()) {
  const valid =
    finding &&
    severities.has(finding.severity) &&
    Number.isInteger(finding.confidence) &&
    finding.confidence >= 80 &&
    finding.confidence <= 100 &&
    Number.isInteger(finding.line) &&
    finding.line >= 1 &&
    requiredStrings.every(
      (field) => typeof finding[field] === 'string' && finding[field].length > 0,
    );

  if (!valid) {
    console.error(`ai-review: finding ${index + 1} does not satisfy the local review contract`);
    process.exit(2);
  }
}

const blocking = report.findings.filter((finding) => finding.severity !== 'low');
const expectedVerdict = blocking.length > 0 ? 'fail' : 'pass';

console.log(`\nai-review: ${report.verdict.toUpperCase()} — ${report.summary}`);

for (const finding of report.findings) {
  console.log(
    `\n[${finding.severity.toUpperCase()}] ${finding.rule_id} ${finding.file}:${finding.line}`,
  );
  console.log(finding.title);
  console.log(`Evidence: ${finding.evidence}`);
  console.log(`Fix: ${finding.remediation}`);
  console.log(`Confidence: ${finding.confidence}`);
}

if (report.verdict !== expectedVerdict) {
  console.error(
    `\nai-review: inconsistent verdict; expected ${expectedVerdict} from reported severities`,
  );
  process.exit(2);
}

if (blocking.length > 0) {
  console.error(`\nai-review: push blocked by ${blocking.length} actionable finding(s)`);
  process.exit(1);
}

console.log('\nai-review: semantic gate passed');
