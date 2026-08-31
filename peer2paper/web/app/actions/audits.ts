"use server";

import {redirect} from "next/navigation";
import {createClient} from "@/lib/supabase/server";
import {locales, type Locale} from "@/i18n/routing";

export async function createAuditRequest(formData: FormData) {
  const candidate = String(formData.get("locale") ?? "en") as Locale;
  const locale = locales.includes(candidate) ? candidate : "en";
  const supabase = await createClient();
  const {data: {user}} = await supabase.auth.getUser();
  if (!user) redirect(`/${locale}/signin?next=/${locale}/submit`);

  const title = String(formData.get("title") ?? "").trim();
  const claim = String(formData.get("claim") ?? "").trim();
  const paperUrl = String(formData.get("paper_url") ?? "").trim();
  const materialsUrl = String(formData.get("materials_url") ?? "").trim();
  const notes = String(formData.get("notes") ?? "").trim();
  if (title.length < 3 || claim.length < 20) redirect(`/${locale}/submit?error=invalid_request`);

  const {error} = await supabase.from("audit_requests").insert({
    user_id: user.id,
    title,
    claim,
    paper_url: paperUrl || null,
    materials_url: materialsUrl || null,
    notes: notes || null
  });
  if (error) redirect(`/${locale}/submit?error=save_failed`);
  redirect(`/${locale}/dashboard?notice=request_created`);
}
