import React, { useState } from "react";
import { createPortal } from "react-dom";
import { Copy, QrCode, LogOut, Check, Eye, EyeOff } from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import { QRCodeSVG } from "qrcode.react";
import type { PortalView } from "../types";

type Props = {
  portal: PortalView;
  onLeave: () => void;
};

const HIDDEN_PLACEHOLDER = "••••••";

export function PortalHeader({ portal, onLeave }: Props) {
  const [copiedField, setCopiedField] = useState<"id" | "code" | "both" | null>(null);
  const [qrOpen, setQrOpen] = useState(false);
  // Default to hidden so the code isn't shoulder-surfed in screen-shares
  // or screenshots. The user reveals deliberately.
  const [codeVisible, setCodeVisible] = useState(false);

  const copy = async (text: string, field: "id" | "code" | "both") => {
    try {
      await navigator.clipboard.writeText(text);
      setCopiedField(field);
      setTimeout(() => setCopiedField(null), 1500);
    } catch {}
  };

  const inviteText = portal.code
    ? `Portal\nID:   ${portal.portalId}\nCode: ${portal.code}`
    : `Portal\nID:   ${portal.portalId}`;

  return (
    <div className="draggable titlebar-pad px-5 h-[96px] flex items-center justify-between border-b border-white/5 bg-black/10 backdrop-blur-md">
      <div className="no-drag flex items-center gap-5">
        <Stat
          label="ID"
          value={portal.portalId}
          field="id"
          copiedField={copiedField}
          onCopy={() => copy(portal.portalId, "id")}
          revealed
        />
        {portal.code && (
          <Stat
            label="KOD"
            value={portal.code}
            field="code"
            copiedField={copiedField}
            onCopy={() => copy(portal.code, "code")}
            revealed={codeVisible}
            onToggleVisibility={() => setCodeVisible((v) => !v)}
          />
        )}
        <button
          onClick={() => copy(inviteText, "both")}
          title="ID + KOD ni birga nusxa olish"
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

      {/* Modal is rendered via createPortal to document.body. The
          parent header has a backdrop-filter, which establishes a
          containing block for descendants — that breaks `position:
          fixed` and made the QR card render inside the 96px header
          stripe. Portaling out of that subtree restores viewport
          coordinates. */}
      {createPortal(
        <AnimatePresence>
          {qrOpen && (
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              className="fixed inset-0 z-[60] bg-black/70 backdrop-blur-md overflow-y-auto"
              onClick={() => setQrOpen(false)}
            >
              <div className="min-h-full flex items-start sm:items-center justify-center px-6 pb-8 titlebar-pad pt-12">
                <motion.div
                  initial={{ scale: 0.9, opacity: 0, y: 12 }}
                  animate={{ scale: 1, opacity: 1, y: 0 }}
                  exit={{ scale: 0.9, opacity: 0, y: 12 }}
                  transition={{ type: "spring", stiffness: 320, damping: 26 }}
                  className="bg-[#0d1322] panel rounded-card p-6 flex flex-col items-center gap-4 max-w-sm w-full my-auto"
                  onClick={(e) => e.stopPropagation()}
                >
                  <div className="bg-white p-3 rounded-lg">
                    <QRCodeSVG
                      value={inviteText}
                      size={200}
                      bgColor="#ffffff"
                      fgColor="#0a0e1a"
                      level="M"
                      includeMargin={false}
                    />
                  </div>
                  <div className="text-center w-full">
                    <div className="text-xs uppercase tracking-widest text-zinc-500 mb-1.5">
                      ID + KOD
                    </div>
                    <div className="font-mono text-xl tracking-wider gradient-text font-semibold break-all">
                      {portal.portalId} · {portal.code}
                    </div>
                    <div className="text-xs text-zinc-500 mt-3 leading-relaxed">
                      QR ni do'stingizning kamerasiga tutsangiz,<br />
                      portalga to'g'ridan-to'g'ri kiradi.
                    </div>
                  </div>
                  <button
                    onClick={() => setQrOpen(false)}
                    className="text-xs text-zinc-400 hover:text-white mt-1 px-4 py-1.5 rounded-md hover:bg-white/[0.05]"
                  >
                    yopish
                  </button>
                </motion.div>
              </div>
            </motion.div>
          )}
        </AnimatePresence>,
        document.body
      )}
    </div>
  );
}

function Stat({
  label,
  value,
  field,
  copiedField,
  onCopy,
  revealed,
  onToggleVisibility,
}: {
  label: string;
  value: string;
  field: "id" | "code";
  copiedField: "id" | "code" | "both" | null;
  onCopy: () => void;
  revealed: boolean;
  onToggleVisibility?: () => void;
}) {
  const copied = copiedField === field;
  const display = revealed ? value : HIDDEN_PLACEHOLDER;
  return (
    <div className="no-drag flex flex-col items-start group">
      <div className="text-[10px] uppercase tracking-widest text-zinc-500">{label}</div>
      <div className="flex items-center gap-1.5">
        <button
          onClick={onCopy}
          title={revealed ? "Nusxa olish" : "Avval ko'rsatish kerak"}
          disabled={!revealed && onToggleVisibility !== undefined}
          className="font-mono text-lg gradient-text font-semibold tracking-wide flex items-center gap-1.5 disabled:cursor-not-allowed"
        >
          <span style={{ minWidth: "5.5em" }} className="tabular-nums">
            {display}
          </span>
          {copied ? (
            <Check className="w-3.5 h-3.5 text-emerald-400" />
          ) : (
            revealed && (
              <Copy className="w-3 h-3 text-zinc-500 opacity-0 group-hover:opacity-100 transition-opacity" />
            )
          )}
        </button>
        {onToggleVisibility && (
          <button
            onClick={onToggleVisibility}
            title={revealed ? "Yashirish" : "Ko'rsatish"}
            className="p-1 rounded-md text-zinc-500 hover:text-zinc-200 hover:bg-white/[0.05]"
          >
            {revealed ? (
              <EyeOff className="w-3.5 h-3.5" strokeWidth={2} />
            ) : (
              <Eye className="w-3.5 h-3.5" strokeWidth={2} />
            )}
          </button>
        )}
      </div>
    </div>
  );
}
