# Security policy

Security fixes are applied to the latest version on the default branch.

Do not open a public issue for suspected vulnerabilities. Use the repository's **Security** tab and select **Report a vulnerability** to open a private security advisory with the minimum reproducible information. Private vulnerability reporting must be enabled before this repository becomes public. Never attach credentials, private papers, datasets, participant records, answer keys, or audit packages to the initial report. The maintainers will arrange an appropriate private transfer if additional evidence is necessary.

Include the affected component, observed and expected behavior, practical impact, reproduction steps, and any mitigation already attempted. Please allow the maintainers a reasonable opportunity to investigate and coordinate a fix before public disclosure.

The monorepo does not require production secrets to build or test. Supabase service-role keys, Google client secrets, grader tokens, and private answer keys must remain outside Git and outside browser-visible configuration. A deployment operator is responsible for Row Level Security, authentication rate limits, SMTP, OAuth configuration, dependency patching, logging, backups, monitoring, and incident response.

This policy covers Peer2Paper-authored code in this repository. Report vulnerabilities in a dependency to its upstream maintainer as well as to Peer2Paper when the repository's use of that dependency creates a practical risk.
