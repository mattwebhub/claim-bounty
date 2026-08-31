import Image from "next/image";
import {Link} from "@/i18n/navigation";

export function Brand({inverse = false}: {inverse?: boolean}) {
  return (
    <Link href="/" className={`brand ${inverse ? "brand--inverse" : ""}`} aria-label="Peer2Paper home">
      <Image
        className="brand-logo"
        src="/peer2paper-fox-loupe.png"
        width={512}
        height={512}
        alt=""
        aria-hidden="true"
      />
      <span>Peer2Paper</span>
    </Link>
  );
}
