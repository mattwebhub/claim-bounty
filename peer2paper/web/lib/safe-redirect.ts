import {siteUrl} from "./site";

const validationOrigin = "https://peer2paper.invalid";
const unsafeCharacter = /[\\\u0000-\u001f\u007f]/;
const encodedSeparator = /%(?:2f|5c)/i;

function normalizeLocalPath(value: unknown) {
  if (
    typeof value !== "string" ||
    !value.startsWith("/") ||
    value.startsWith("//") ||
    unsafeCharacter.test(value) ||
    encodedSeparator.test(value)
  ) {
    return null;
  }

  try {
    const target = new URL(value, validationOrigin);
    if (target.origin !== validationOrigin) return null;
    return `${target.pathname}${target.search}${target.hash}`;
  } catch {
    return null;
  }
}

export function safeLocalPath(value: unknown, fallback = "/en/dashboard") {
  return normalizeLocalPath(value) ?? normalizeLocalPath(fallback) ?? "/en/dashboard";
}

export function authCallbackUrl(next: string) {
  const callback = new URL("/auth/callback", siteUrl);
  callback.searchParams.set("next", safeLocalPath(next));
  return callback.toString();
}
