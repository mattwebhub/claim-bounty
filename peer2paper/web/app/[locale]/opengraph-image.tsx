import {ImageResponse} from "next/og";
import {readFile} from "node:fs/promises";
import {join} from "node:path";

export const alt = "Peer2Paper — independent scientific claim audits";
export const size = {width: 1200, height: 630};
export const contentType = "image/png";
export const runtime = "nodejs";

export default async function Image() {
  const logoData = await readFile(join(process.cwd(), "public/peer2paper-fox-loupe.png"), "base64");
  const logoDataUrl = `data:image/png;base64,${logoData}`;
  return new ImageResponse(
    <div style={{width: "100%", height: "100%", display: "flex", flexDirection: "column", justifyContent: "space-between", padding: 76, background: "#f4f1e8", color: "#17221d", fontFamily: "Arial"}}>
      <div style={{display: "flex", alignItems: "center", gap: 18, fontSize: 28, fontWeight: 700}}>
        {/* ImageResponse renders plain image elements; next/image is not supported here. */}
        <img src={logoDataUrl} width={52} height={52} alt="" />
        Peer2Paper
      </div>
      <div style={{display: "flex", flexDirection: "column", maxWidth: 980}}>
        <div style={{fontSize: 76, lineHeight: 1.02, letterSpacing: -4, fontWeight: 700}}>Know whether a scientific claim holds up.</div>
        <div style={{marginTop: 28, fontSize: 26, color: "#4d5c55"}}>Independent reproduction, robustness testing and evidence-backed verdicts.</div>
      </div>
      <div style={{fontSize: 18, color: "#4d5c55"}}>Evidence before certainty.</div>
    </div>,
    size
  );
}
