import {NextResponse} from "next/server";
import {createClient} from "@/lib/supabase/server";
import {safeLocalPath} from "@/lib/safe-redirect";
import {siteUrl} from "@/lib/site";

export async function GET(request: Request) {
  const url = new URL(request.url);
  const code = url.searchParams.get("code");
  const next = safeLocalPath(url.searchParams.get("next"));
  if (code) {
    const supabase = await createClient();
    const {error} = await supabase.auth.exchangeCodeForSession(code);
    if (!error) return NextResponse.redirect(new URL(next, siteUrl));
  }
  return NextResponse.redirect(new URL("/en/signin?error=callback_failed", siteUrl));
}
