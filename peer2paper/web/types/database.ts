export type AuditStatus = "submitted" | "in_review" | "running" | "completed" | "needs_input";

export type AuditRequest = {
  id: string;
  user_id: string;
  title: string;
  claim: string;
  paper_url: string | null;
  materials_url: string | null;
  notes: string | null;
  status: AuditStatus;
  created_at: string;
  updated_at: string;
};
