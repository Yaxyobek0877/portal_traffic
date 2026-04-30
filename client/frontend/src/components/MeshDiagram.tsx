// Animated SVG diagram of the mesh: every peer (plus self) is laid
// out around the circumference; full-mesh edges between every pair;
// pulses travel along edges to evoke "data flowing".

import React, { useEffect, useRef, useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import type { PeerView } from "../types";
import { avatarColor, avatarInitial } from "../lib/avatar";
import { rttLabel, shortId } from "../lib/format";

type Props = {
  selfNickname: string;
  selfVip: string;
  peers: PeerView[];
  hovered?: string | null;
  onHover?: (peerId: string | null) => void;
};

type Node = {
  id: string;
  label: string;
  vip: string;
  isSelf: boolean;
  isOwner: boolean;
  state: PeerView["state"];
  rttMs: number;
  x: number;
  y: number;
};

const W = 560;
const H = 460;
const CX = W / 2;
const CY = H / 2;

export function MeshDiagram({ selfNickname, selfVip, peers, hovered, onHover }: Props) {
  const [pulses, setPulses] = useState<Pulse[]>([]);
  const idCtr = useRef(0);

  // Position: self in the centre; peers around a circle.
  const others = peers.filter((p) => p.peerId);
  const radius = others.length === 0 ? 0 : Math.min(190, 80 + others.length * 12);
  const nodes: Node[] = [
    {
      id: "self",
      label: selfNickname || "you",
      vip: selfVip,
      isSelf: true,
      isOwner: false,
      state: "connected",
      rttMs: 0,
      x: CX,
      y: CY,
    },
    ...others.map((p, i) => {
      const a = (i / others.length) * Math.PI * 2 - Math.PI / 2;
      return {
        id: p.peerId,
        label: p.nickname || shortId(p.peerId),
        vip: p.virtualIp,
        isSelf: false,
        isOwner: p.isOwner,
        state: p.state,
        rttMs: p.rttMs,
        x: CX + Math.cos(a) * radius,
        y: CY + Math.sin(a) * radius,
      };
    }),
  ];

  // Edges: self → every peer, plus every peer pair (full mesh).
  const edges: Array<[Node, Node]> = [];
  for (let i = 0; i < nodes.length; i++) {
    for (let j = i + 1; j < nodes.length; j++) {
      edges.push([nodes[i], nodes[j]]);
    }
  }

  // Periodic pulse: pick a random edge, animate a dot along it.
  useEffect(() => {
    if (edges.length === 0) return;
    let cancelled = false;
    const tick = () => {
      if (cancelled) return;
      const e = edges[Math.floor(Math.random() * edges.length)];
      const reverse = Math.random() > 0.5;
      const a = reverse ? e[1] : e[0];
      const b = reverse ? e[0] : e[1];
      const id = ++idCtr.current;
      const pulse: Pulse = { id, ax: a.x, ay: a.y, bx: b.x, by: b.y };
      setPulses((cur) => [...cur, pulse]);
      setTimeout(() => setPulses((cur) => cur.filter((p) => p.id !== id)), 800);
      const next = 250 + Math.random() * 400;
      timer = window.setTimeout(tick, next);
    };
    let timer = window.setTimeout(tick, 600);
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [edges.length]);

  const isHighlighted = (n: Node) =>
    hovered != null && (n.id === hovered || n.id === "self");

  return (
    <div className="relative w-full max-w-[640px] mx-auto">
      {others.length === 0 && (
        <motion.div
          initial={{ opacity: 0, y: 8 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.3, duration: 0.6 }}
          className="absolute left-1/2 -translate-x-1/2"
          style={{ top: 60 }}
        >
          <div className="text-center text-zinc-500">
            <div className="text-xs uppercase tracking-widest text-violet-300/70 mb-1.5">
              Kutilmoqda
            </div>
            <div className="text-sm">
              Hech kim hali qo'shilmagan.<br />
              <span className="text-zinc-600 text-xs">
                ID + KOD ni do'stingizga ulashing — meshda ko'rinadi.
              </span>
            </div>
          </div>
        </motion.div>
      )}
    <svg
      viewBox={`0 0 ${W} ${H}`}
      className="max-w-full h-auto"
      style={{ minHeight: 360 }}
    >
      <defs>
        <linearGradient id="edgeGradMesh" x1="0" y1="0" x2="1" y2="0">
          <stop offset="0%" stopColor="#8b5cf6" />
          <stop offset="100%" stopColor="#22d3ee" />
        </linearGradient>
        <radialGradient id="nodeFillSelf" cx="50%" cy="50%" r="50%">
          <stop offset="0%" stopColor="#fff" />
          <stop offset="60%" stopColor="#a78bfa" />
          <stop offset="100%" stopColor="#6366f1" />
        </radialGradient>
        <radialGradient id="nodeFillPeer" cx="50%" cy="50%" r="50%">
          <stop offset="0%" stopColor="#22d3ee" />
          <stop offset="100%" stopColor="#6366f1" />
        </radialGradient>
      </defs>

      {/* Edges */}
      {edges.map(([a, b], i) => {
        const active =
          hovered != null && (a.id === hovered || b.id === hovered);
        const dim = hovered != null && !active;
        return (
          <line
            key={i}
            x1={a.x}
            y1={a.y}
            x2={b.x}
            y2={b.y}
            stroke="url(#edgeGradMesh)"
            strokeWidth={active ? 2 : 1.1}
            opacity={dim ? 0.08 : active ? 0.85 : 0.32}
            style={{
              transition: "all .25s ease",
              animation: !hovered ? `meshPulse 4s ease-in-out infinite ${i * 0.15}s` : undefined,
            }}
          />
        );
      })}

      {/* Pulses */}
      {pulses.map((p) => (
        <motion.circle
          key={p.id}
          r={3}
          fill="#22d3ee"
          initial={{ cx: p.ax, cy: p.ay, opacity: 1 }}
          animate={{ cx: p.bx, cy: p.by, opacity: 0.5 }}
          transition={{ duration: 0.8, ease: "easeOut" }}
          style={{ filter: "drop-shadow(0 0 6px #22d3ee)" }}
        />
      ))}

      {/* Nodes */}
      <AnimatePresence>
        {nodes.map((n) => {
          const r = n.isSelf ? 22 : 17;
          const color = avatarColor(n.label);
          const initial = avatarInitial(n.label);
          const highlighted = isHighlighted(n);
          return (
            <motion.g
              key={n.id}
              layout
              initial={{ scale: 0.4, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              exit={{ scale: 0.4, opacity: 0 }}
              transition={{ type: "spring", stiffness: 300, damping: 26 }}
              onMouseEnter={() => !n.isSelf && onHover?.(n.id)}
              onMouseLeave={() => !n.isSelf && onHover?.(null)}
              style={{ cursor: n.isSelf ? "default" : "pointer" }}
            >
              <circle
                cx={n.x}
                cy={n.y}
                r={r + 6}
                fill="none"
                stroke="url(#edgeGradMesh)"
                strokeWidth={1}
                opacity={highlighted ? 0.9 : 0.4}
              />
              <circle
                cx={n.x}
                cy={n.y}
                r={r}
                fill={n.isSelf ? "url(#nodeFillSelf)" : `url(#nodeFillPeer)`}
                style={{ animation: `nodePulse 3s ease-in-out infinite` }}
              />
              <text
                x={n.x}
                y={n.y + 4}
                textAnchor="middle"
                fill="#0a0e1a"
                fontFamily="Inter"
                fontWeight={700}
                fontSize={n.isSelf ? 14 : 12}
              >
                {initial}
              </text>
              {/* Label */}
              <text
                x={n.x}
                y={n.y + r + 18}
                textAnchor="middle"
                fill="#e6e9ef"
                fontFamily="Inter"
                fontSize="11"
                fontWeight={500}
              >
                {n.label}
              </text>
              <text
                x={n.x}
                y={n.y + r + 32}
                textAnchor="middle"
                fill="#5b6479"
                fontFamily="JetBrains Mono"
                fontSize="10"
              >
                {n.vip}
                {n.isSelf ? "" : ` · ${rttLabel(n.rttMs)}`}
              </text>
              {/* fallback color hint */}
              <circle cx={n.x + r - 4} cy={n.y - r + 4} r={3} fill={color} opacity={0.7} />
            </motion.g>
          );
        })}
      </AnimatePresence>
    </svg>
    </div>
  );
}

type Pulse = { id: number; ax: number; ay: number; bx: number; by: number };
