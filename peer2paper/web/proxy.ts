import createIntlMiddleware from "next-intl/middleware";
import {createServerClient} from "@supabase/ssr";
import {NextResponse, type NextRequest} from "next/server";
import {routing} from "@/i18n/routing";
import {supabaseAnonKey, supabaseUrl} from "@/lib/supabase/config";

const intl = createIntlMiddleware(routing);
const protectedPath = /^\/(en|pt|es|fr|de|it|nl|ru)\/(dashboard|submit)(?:\/|$)/;

export default async function proxy(request: NextRequest) {
  const response = intl(request);
  if (!supabaseUrl || !supabaseAnonKey) return response;

  const supabase = createServerClient(supabaseUrl, supabaseAnonKey, {
    cookies: {
      getAll: () => request.cookies.getAll(),
      setAll: (items) => {
        items.forEach(({name, value}) => request.cookies.set(name, value));
        items.forEach(({name, value, options}) => response.cookies.set(name, value, options));
      }
    }
  });
  const {data: {user}} = await supabase.auth.getUser();
  if (!user && protectedPath.test(request.nextUrl.pathname)) {
    const locale = request.nextUrl.pathname.split("/")[1] || "en";
    const target = new URL(`/${locale}/signin`, request.url);
    target.searchParams.set("next", request.nextUrl.pathname);
    return NextResponse.redirect(target);
  }
  return response;
}

export const config = {
  matcher: ["/((?!api|auth|_next|_vercel|.*\\..*).*)"]
};
