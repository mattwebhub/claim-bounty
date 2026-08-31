export const demoResult = {
  id: "P2P-CS-0001",
  status: "complete",
  verdict: "Source verified; material sensitivity",
  claim: "Among unvaccinated Republican or Republican-leaning participants, a Republican elite endorsement increases vaccination intentions relative to a Democratic elite endorsement.",
  confidence: "High",
  reported: {estimate: "0.028", interval: "[0.006, 0.041]", pValue: "0.002"},
  audited: {estimate: "0.026631", interval: "[0.008497, 0.044765]", pValue: "0.004053"},
  findings: [
    "The primary source report was verified for its coefficient, interval, test statistic and p value, with explicit limits.",
    "Statistical support changes across verified adjusted and unadjusted analysis paths, while both estimates remain positive.",
    "Source documents conflict about the population included in the analysis."
  ],
  outputs: ["Decision brief", "Technical report", "Machine-readable JSON", "Evidence manifest", "Replay instructions"],
  sources: {
    paper: "https://pmc.ncbi.nlm.nih.gov/articles/PMC8364165/",
    materials: "https://osf.io/rb3cn/"
  }
} as const;
