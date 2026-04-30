import React, { useState } from "react";
import { Copy, QrCode, LogOut, Check } from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import { QRCodeSVG } from "qrcode.react";
import type { PortalView } from "../types";
import { app } from "../lib/wails";

type Props = {
  portal: PortalView;
  onLeave: () => void;
};

export function PortalHeader({ portal, onLeave }: Props) {
  const [copiedField, setCopiedField] = useState<"id" | "code" | "both" | null>(null);
  const [qrOpen, setQrOpen] = useState(false);

  const copy = async (text: string, field: "id" | "code" | "both") => {
    try {
      await navigator.clipboard.writeText(text);
      setCopiedField(field);
      setTimeout(() => setCopiedField(null), 1500);
    } catch {}
  };

  const inviteText = `Portal\nID:   ${portal.portalId}\nCode: ${portal.code}`;

  return (
    <div className="draggable px-5 h-[68px] flex items-center justify-between border-b border-white/5 bg-black/10 backdrop-blur-md">
      <div className="no-drag flex items-center gap-5">
        <Stat
          label="ID"
          value={portal.portalId}
          field="id"
          copiedField={copiedField}
          onCopy={() => copy(portal.portalId, "id")}
        />
        {portal.code && (
          <Stat
            label="KOD"
            value={portal.code}
            field="code"
            copiedField={copiedField}
            onCopy={() => copy(portal.code, "code")}
          />
        )}
        <button
          onClick={() => copy(inviteText, "both")}
          title="Ikkalasini birga nusxa olish"
          className="no-drag h-9 px-3 panel rounded-btn flex items-center gap-1.5 text-xs hover:bg-white/[0.07]"
        >
          {copiedField === "both" ? (
            <>
              <Check className="w-3.5 h-3.5 text-emerald-400" /> Nusxalandi
            </>
          ) : (
            <>
              <Copy className="w-3.5 h-3.5" /> Taklif
            </>
          )}
        </button>
        {portal.code && (
          <button
            onClick={() => setQrOpen(true)}
            className="no-drag h-9 px-3 panel rounded-btn flex items-center gap-1.5 text-xs hover:bg-white/[0.07]"
          >
            <QrCode className="w-3.5 h-3.5" /> QR
          </button>
        )}
      </div>

      <button
        onClick={onLeave}
        className="no-drag h-9 px-3 rounded-btn text-xs flex items-center gap-1.5 text-rose-300 hover:bg-rose-500/10"
      >
        <LogOut className="w-3.5 h-3.5" /> Chiqish
      </button>

      <AnimatePresence>
        {qrOpen && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 z-30 bg-black/60 backdrop-blur-sm flex items-center justify-center"
            onClick={() => setQrOpen(false)}
          >
            <motion.div
              initial={{ scale: 0.9, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              exit={{ scale: 0.9, opacity: 0 }}
              transition={{ type: "spring", stiffness: 300, damping: 26 }}
              className="bg-[#0d1322] panel rounded-card p-8 flex flex-col items-center gap-4"
              onClick={(e) => e.stopPropagation()}
            >
              <QRCodeSVG
                value={inviteText}
                size={260}
                bgColor="#0d1322"
                fgColor="#ffffff"
                level="M"
                includeMargin={false}
              />
              <div className="text-center">
                <div className="font-mono text-2xl tracking-wider gradient-text">
                  {portal.portalId} · {portal.code}
                </div>
                <div className="text-xs text-zinc-500 mt-2">
                  QR ni kameraga tutsangiz, do'stlaringiz portalga kiradi.
                </div>
              </div>
              <button
                onClick={() => setQrOpen(false)}
                className="text-xs text-zinc-400 hover:text-white mt-2"
              >
                yopish
              </button>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}

function Stat({
  label,
  value,
  field,
  copiedField,
  onCopy,
}: {
  label: string;
  value: string;
  field: "id" | "code";
  copiedField: "id" | "code" | "both" | null;
  onCopy: () => void;
}) {
  const copied = copiedField === field;
  return (
    <button
      onClick={onCopy}
      className="no-drag flex flex-col items-start group"
      title="Nusxa olish"
    >
      <div className="text-[10px] uppercase tracking-widest text-zinc-500">{label}</div>
      <div className="font-mono text-lg gradient-text font-semibold tracking-wide flex items-center gap-1.5">
        {value}
        {copied ? (
          <Check className="w-3.5 h-3.5 text-emerald-400" />
        ) : (
          <Copy className="w-3 h-3 text-zinc-500 opacity-0 group-hover:opacity-100 transition-opacity" />
        )}
      </div>
    </button>
  );
}
