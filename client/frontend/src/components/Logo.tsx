import React from "react";

// Animated wormhole logo — the same motif as the landing page.
export function Logo({ size = 220 }: { size?: number }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 200 200"
      aria-hidden="true"
      style={{ filter: "drop-shadow(0 0 40px rgba(139,92,246,0.45))" }}
    >
      <defs>
        <linearGradient id="wgClient" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0%" stopColor="#8b5cf6" />
          <stop offset="60%" stopColor="#6366f1" />
          <stop offset="100%" stopColor="#22d3ee" />
        </linearGradient>
        <radialGradient id="wgGlowClient" cx="50%" cy="50%" r="50%">
          <stop offset="0%" stopColor="#8b5cf6" stopOpacity="0.9" />
          <stop offset="100%" stopColor="#6366f1" stopOpacity="0" />
        </radialGradient>
      </defs>
      <g style={{ animation: "nodePulse 2.4s ease-in-out infinite", transformOrigin: "100px 100px" }}>
        <circle cx="100" cy="100" r="14" fill="url(#wgGlowClient)" opacity={0.85} />
        <circle cx="100" cy="100" r="6" fill="#fff" opacity={0.95} />
      </g>
      {[
        { r: 92, w: 1.2, dash: "3 9", op: 0.55, dur: 14, dir: 1 },
        { r: 78, w: 1.4, dash: "6 4", op: 0.7, dur: 9, dir: -1 },
        { r: 62, w: 1.7, dash: "2 6", op: 0.82, dur: 6, dir: 1 },
        { r: 46, w: 2.1, dash: "14 6", op: 1, dur: 4, dir: -1 },
        { r: 30, w: 1.6, dash: "", op: 0.65, dur: 2.5, dir: 1 },
      ].map((ring, i) => (
        <g
          key={i}
          style={{
            animation: `spin360 ${ring.dur}s linear infinite ${ring.dir < 0 ? "reverse" : ""}`,
            transformOrigin: "100px 100px",
          }}
        >
          <circle
            cx="100"
            cy="100"
            r={ring.r}
            stroke="url(#wgClient)"
            strokeWidth={ring.w}
            strokeDasharray={ring.dash || undefined}
            fill="none"
            opacity={ring.op}
          />
        </g>
      ))}
    </svg>
  );
}
