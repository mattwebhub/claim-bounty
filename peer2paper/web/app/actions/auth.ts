"use server";

import {redirect} from "next/navigation";
import {locales, type Locale} from "@/i18n/routing";
import {createClient} from "@/lib/supabase/server";
import {isSupabaseConfigured} from "@/lib/supabase/config";
import {authCallbackUrl, safeLocalPath} from "@/lib/safe-redirect";

function localeOf(value: FormDataEntryValue | null): Locale {
  return locales.includes(value as Locale) ? (value as Locale) : "en";
}

function fail(locale: Locale, page: string, code: string): never {
  redirect(`/${locale}/${page}?error=${encodeURIComponent(code)}`);
}

export async function signIn(formData: FormData) {
  const locale = localeOf(formData.get("locale"));
  if (!isSupabaseConfigured()) fail(locale, "signin", "not_configured");
  const email = String(formData.get("email") ?? "").trim();
  const password = String(formData.get("password") ?? "");
  if (!email || !password) fail(locale, "signin", "missing_fields");
  const supabase = await createClient();
  const {error} = await supabase.auth.signInWithPassword({email, password});
  if (error) fail(locale, "signin", "invalid_credentials");
  redirect(safeLocalPath(formData.get("next"), `/${locale}/dashboard`));
}

export async function signUp(formData: FormData) {
  const locale = localeOf(formData.get("locale"));
  if (!isSupabaseConfigured()) fail(locale, "signup", "not_configured");
  const email = String(formData.get("email") ?? "").trim();
  const password = String(formData.get("password") ?? "");
  const name = String(formData.get("name") ?? "").trim();
  if (!email || password.length < 8) fail(locale, "signup", "invalid_signup");
  const supabase = await createClient();
  const {error} = await supabase.auth.signUp({
    email,
    password,
    options: {
      data: {full_name: name},
      emailRedirectTo: authCallbackUrl(`/${locale}/dashboard`)
    }
  });
  if (error) fail(locale, "signup", error.code === "user_already_exists" ? "already_exists" : "signup_failed");
  redirect(`/${locale}/signin?notice=check_email`);
}

export async function signInWithGoogle(formData: FormData) {
  const locale = localeOf(formData.get("locale"));
  if (!isSupabaseConfigured()) fail(locale, "signin", "not_configured");
  const supabase = await createClient();
  const {data, error} = await supabase.auth.signInWithOAuth({
    provider: "google",
    options: {redirectTo: authCallbackUrl(`/${locale}/dashboard`), queryParams: {access_type: "offline", prompt: "consent"}}
  });
  if (error || !data.url) fail(locale, "signin", "oauth_failed");
  redirect(data.url);
}

export async function requestPasswordReset(formData: FormData) {
  const locale = localeOf(formData.get("locale"));
  if (!isSupabaseConfigured()) fail(locale, "forgot-password", "not_configured");
  const email = String(formData.get("email") ?? "").trim();
  const supabase = await createClient();
  await supabase.auth.resetPasswordForEmail(email, {redirectTo: authCallbackUrl(`/${locale}/update-password`)});
  redirect(`/${locale}/forgot-password?notice=check_email`);
}

export async function updatePassword(formData: FormData) {
  const locale = localeOf(formData.get("locale"));
  const password = String(formData.get("password") ?? "");
  if (password.length < 8) fail(locale, "update-password", "invalid_signup");
  const supabase = await createClient();
  const {error} = await supabase.auth.updateUser({password});
  if (error) fail(locale, "update-password", "update_failed");
  redirect(`/${locale}/dashboard?notice=password_updated`);
}

export async function signOut(formData: FormData) {
  const locale = localeOf(formData.get("locale"));
  if (isSupabaseConfigured()) {
    const supabase = await createClient();
    await supabase.auth.signOut();
  }
  redirect(`/${locale}/`);
}
