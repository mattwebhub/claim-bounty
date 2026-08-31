import {Link} from "@/i18n/navigation";

export function Brand({inverse = false}: {inverse?: boolean}) {
  return (
    <Link href="/" className={`brand ${inverse ? "brand--inverse" : ""}`} aria-label="Peer2Paper home">
      <span className="brand-mark" aria-hidden="true">
        <span />
      </span>
      <span>Peer2Paper</span>
    </Link>
  );
}
