import {createServerClient} from "@supabase/ssr";
import {cookies} from "next/headers";
import {supabaseAnonKey, supabaseUrl} from "./config";

export async function createClient() {
  if (!supabaseUrl || !supabaseAnonKey) throw new Error("Supabase is not configured");
  const cookieStore = await cookies();
  return createServerClient(supabaseUrl, supabaseAnonKey, {
    cookies: {
      getAll: () => cookieStore.getAll(),
      setAll: (items) => {
        try {
          items.forEach(({name, value, options}) => cookieStore.set(name, value, options));
        } catch {
          // Session refresh is persisted by proxy when this runs in a Server Component.
        }
      }
    }
  });
}
