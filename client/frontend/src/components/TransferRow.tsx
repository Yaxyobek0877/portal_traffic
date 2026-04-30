import React from "react";
import { motion } from "framer-motion";
import { CheckCircle2, AlertCircle, ArrowDown, ArrowUp, FileIcon, FolderOpen } from "lucide-react";
import type { TransferProgress } from "../types";
import { app } from "../lib/wails";

export function TransferRow({ t }: { t: TransferProgress }) {
  const pct = t.total > 0 ? Math.min(100, (t.bytes / t.total) * 100) : 0;
  const status = t.error
    ? "error"
    : t.done
    ? "done"
    : "active";

  const ArrowIcon = t.direction === "send" ? ArrowUp : ArrowDown;
  const dirLabel = t.direction === "send" ? "yuborilmoqda" : "kelmoqda";

  return (
    <motion.div
      layout
      initial={{ opacity: 0, y: 4 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0 }}
      className="panel rounded-card p-3 flex items-center gap-3"
    >
      <div className="w-9 h-9 rounded-md bg-violet-500/10 flex items-center justify-center shrink-0">
        <FileIcon className="w-4 h-4 text-violet-300" strokeWidth={2} />
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <ArrowIcon className="w-3 h-3 text-zinc-400 shrink-0" strokeWidth={2.5} />
          <span className="text-sm font-medium truncate">{t.manifest.name}</span>
          <span className="text-xs text-zinc-500 font-mono ml-auto shrink-0">
            {humanBytes(t.bytes)} / {humanBytes(t.total)}
          </span>
        </div>
        <div className="mt-1.5 h-1 rounded-full bg-white/[0.05] overflow-hidden">
          <motion.div
            className={`h-full ${
              status === "error"
                ? "bg-rose-500"
                : status === "done"
                ? "bg-emerald-500"
                : "bg-gradient-to-r from-violet-500 to-cyan-400"
            }`}
            initial={false}
            animate={{ width: `${pct}%` }}
            transition={{ ease: "easeOut", duration: 0.2 }}
          />
        </div>
        <div className="mt-1 flex items-center gap-2 text-[10px] text-zinc-500">
          {status === "error" && (
            <span className="text-rose-300 flex items-center gap-1">
              <AlertCircle className="w-3 h-3" /> {t.error}
            </span>
          )}
          {status === "done" && !t.error && (
            <span className="text-emerald-300 flex items-center gap-1">
              <CheckCircle2 className="w-3 h-3" /> tugadi
            </span>
          )}
          {status === "active" && <span>{dirLabel} ({pct.toFixed(0)}%)</span>}
          {t.savePath && (
            <button
              onClick={() => app.OpenSaveDir()}
              className="ml-auto flex items-center gap-1 hover:text-zinc-200"
              title={t.savePath}
            >
              <FolderOpen className="w-3 h-3" /> ochish
            </button>
          )}
        </div>
      </div>
    </motion.div>
  );
}

function humanBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  if (n < 1024 * 1024 * 1024) return `${(n / 1024 / 1024).toFixed(1)} MB`;
  return `${(n / 1024 / 1024 / 1024).toFixed(2)} GB`;
}
