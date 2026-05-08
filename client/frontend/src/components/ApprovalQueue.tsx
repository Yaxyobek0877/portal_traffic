// ApprovalQueue — global modal that pops up when a peer tries to
// open one of our services that's flagged require_approval.
//
// Lives at the App.tsx level so it overlays whichever screen the
// user is on. Subscribes to the `service:approval-request` event,
// queues incoming requests (so two peers dialing simultaneously
// don't lose one), and presents them one at a time. The user clicks
// Allow / Deny / Allow always (cache a permanent allow for the
// session via the Go-side approvalDecided cache).
//
// Decisions go back to Go via app.ApproveServiceRequest /
// DenyServiceRequest with the requestId. Timeouts (5s TCP / 3s UDP)
// fire `service:approval-cancelled` from Go so we drop the modal
// when the peer's dial would have given up anyway.

import React, { useEffect, useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { ShieldQuestion, Check, X, Globe } from "lucide-react";
import { app, subscribe } from "../lib/wails";
import type { ApprovalRequest } from "../types";

export function ApprovalQueue() {
  const [queue, setQueue] = useState<ApprovalRequest[]>([]);
  const current = queue[0] ?? null;

  useEffect(() => {
    const offs: Array<() => void> = [];
    offs.push(
      subscribe<ApprovalRequest>("service:approval-request", (req) => {
        if (!req || !req.requestId) return;
        setQueue((q) => {
          // Dedup by requestId in case the event fires twice.
          if (q.some((r) => r.requestId === req.requestId)) return q;
          return [...q, req];
        });
      }),
    );
    offs.push(
      subscribe<{ requestId: string }>("service:approval-cancelled", (p) => {
        if (!p || !p.requestId) return;
        setQueue((q) => q.filter((r) => r.requestId !== p.requestId));
      }),
    );
    return () => offs.forEach((off) => off());
  }, []);

  const decide = async (allow: boolean) => {
    if (!current) return;
    setQueue((q) => q.slice(1));
    try {
      if (allow) {
        await app.ApproveServiceRequest(current.requestId);
      } else {
        await app.DenyServiceRequest(current.requestId);
      }
    } catch {}
  };

  return (
    <AnimatePresence>
      {current && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          className="fixed inset-0 z-[70] bg-black/70 backdrop-blur-md flex items-center justify-center p-6 titlebar-pad"
        >
          <motion.div
            initial={{ scale: 0.92, opacity: 0, y: 8 }}
            animate={{ scale: 1, opacity: 1, y: 0 }}
            exit={{ scale: 0.92, opacity: 0, y: 8 }}
            transition={{ type: "spring", stiffness: 320, damping: 26 }}
            className="bg-[#0d1322] panel rounded-card p-6 w-full max-w-md space-y-4"
          >
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-full bg-amber-500/15 border border-amber-500/30 flex items-center justify-center shrink-0">
                <ShieldQuestion className="w-5 h-5 text-amber-300" strokeWidth={2} />
              </div>
              <div className="min-w-0 flex-1">
                <div className="font-semibold text-base">Ulanish so'rovi</div>
                <div className="text-xs text-zinc-500 mt-0.5">
                  Peer servisingizga ulanmoqchi — ruxsat berasizmi?
                </div>
              </div>
            </div>

            <div className="panel rounded-input p-3 space-y-2 text-sm">
              <div className="flex items-center gap-2">
                <span className="text-xs text-zinc-500 w-20 shrink-0">Kim</span>
                <span className="font-medium truncate">
                  {current.nickname || current.peerId.slice(0, 8)}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-xs text-zinc-500 w-20 shrink-0">Servis</span>
                <Globe className="w-3.5 h-3.5 text-cyan-400 shrink-0" strokeWidth={2} />
                <span className="font-medium truncate">{current.serviceName}</span>
                <span
                  className={`text-[10px] font-mono px-1.5 py-0.5 rounded ${
                    current.protocol === "udp"
                      ? "bg-cyan-400/10 text-cyan-300"
                      : "bg-violet-400/10 text-violet-300"
                  }`}
                >
                  {current.protocol.toUpperCase()}
                </span>
                <span className="font-mono text-xs text-zinc-500">:{current.port}</span>
              </div>
            </div>

            <p className="text-xs text-zinc-500 leading-relaxed">
              Ruxsat berishingiz bilan ushbu peer shu sessiya davomida bu portni qayta-qayta
              tasdiqlatmasdan ulashi mumkin bo'ladi. Boshqalar uchun yangi modal chiqadi.
              Hozir 5 soniyada javob bermasangiz, avtomatik rad etiladi.
            </p>

            <div className="grid grid-cols-2 gap-2">
              <button
                onClick={() => decide(false)}
                className="rounded-btn px-4 py-2.5 text-sm flex items-center justify-center gap-2 bg-rose-500/15 text-rose-300 border border-rose-500/25 hover:bg-rose-500/25"
              >
                <X className="w-4 h-4" strokeWidth={2.5} />
                Rad etish
              </button>
              <button
                onClick={() => decide(true)}
                className="btn-primary rounded-btn px-4 py-2.5 text-sm font-semibold flex items-center justify-center gap-2"
              >
                <Check className="w-4 h-4" strokeWidth={2.5} />
                Ruxsat berish
              </button>
            </div>

            {queue.length > 1 && (
              <div className="text-[11px] text-zinc-600 text-center">
                Yana {queue.length - 1} ta ulanish so'rovi navbatda…
              </div>
            )}
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}
