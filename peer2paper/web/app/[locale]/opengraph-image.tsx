import {ImageResponse} from "next/og";

export const alt = "Peer2Paper — independent scientific claim audits";
export const size = {width: 1200, height: 630};
export const contentType = "image/png";

export default function Image() {
  return new ImageResponse(
    <div style={{width: "100%", height: "100%", display: "flex", flexDirection: "column", justifyContent: "space-between", padding: 76, background: "#f4f1e8", color: "#17221d", fontFamily: "Arial"}}>
      <div style={{display: "flex", alignItems: "center", gap: 18, fontSize: 28, fontWeight: 700}}>
        <div style={{width: 44, height: 44, display: "flex", alignItems: "center", justifyContent: "center", border: "4px solid #173e31", borderRadius: 12, transform: "rotate(45deg)"}}><div style={{width: 12, height: 12, borderRadius: 99, background: "#d7653b"}} /></div>
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
