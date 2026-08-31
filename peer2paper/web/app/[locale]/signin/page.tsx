import {AuthPanel} from "@/components/auth-panel";
import type {Metadata} from "next";
export const metadata: Metadata = {robots: {index: false, follow: false}};
export default async function Page({params, searchParams}: {params: Promise<{locale: string}>; searchParams: Promise<{error?: string; notice?: string; next?: string}>}) {
  const [{locale}, query] = await Promise.all([params, searchParams]);
  return <AuthPanel mode="signin" locale={locale} {...query} />;
}
