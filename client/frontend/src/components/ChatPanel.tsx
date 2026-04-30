import React, { useEffect, useRef, useState } from "react";
import { Paperclip, Send } from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import type { ChatMessage, PeerView, TransferProgress } from "../types";
import { app } from "../lib/wails";
import { avatarColor, avatarInitial } from "../lib/avatar";
import { timeOfDay } from "../lib/format";
import { TransferRow } from "./TransferRow";

type Props = {
  messages: ChatMessage[];
  myPeerId: string;
  peers: PeerView[];
  transfers: TransferProgress[];
};

export function ChatPanel({ messages, myPeerId, peers, transfers }: Props) {
  const [draft, setDraft] = useState("");
  const [dragOver, setDragOver] = useState(false);
  const scrollerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const el = scrollerRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [messages.length, transfers.length]);

  const send = async () => {
    const text = draft.trim();
    if (!text) return;
    setDraft("");
    await app.SendChat(text);
  };

  const sendFile = async () => {
    if (peers.length === 0) return;
    // For v1 we offer "send to first connected peer" via the picker.
    // Multi-peer broadcast would multi-stream; UI picker is Phase 5.
    const first = peers.find((p) => p.state === "connected") ?? peers[0];
    try {
      await app.SendFile(first.peerId);
    } catch {}
  };

  // Drag-and-drop. Wails on macOS bridges OS file drops via window
  // dragenter/drop events on the webview. The dropped File objects
  // have a `path` property under Wails (non-standard but documented).
  const onDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(true);
  };
  const onDragLeave = () => setDragOver(false);
  const onDrop = async (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(false);
    if (peers.length === 0) return;
    const first = peers.find((p) => p.state === "connected") ?? peers[0];
    const files = Array.from(e.dataTransfer.files) as Array<File & { path?: string }>;
    for (const f of files) {
      const path = (f as any).path as string | undefined;
      if (path) {
        try {
          await app.SendFilePath(first.peerId, path);
        } catch {}
      }
    }
  };

  return (
    <div
      className={`h-full flex flex-col relative ${
        dragOver ? "ring-2 ring-violet-400/60 ring-inset" : ""
      }`}
      onDragOver={onDragOver}
      onDragLeave={onDragLeave}
      onDrop={onDrop}
    >
      {dragOver && (
        <div className="absolute inset-0 bg-violet-500/10 backdrop-blur-sm z-20 flex items-center justify-center pointer-events-none">
          <div className="text-sm text-violet-200 font-medium">
            Tashlang — meshda yuborish boshlanadi
          </div>
        </div>
      )}

      <div ref={scrollerRef} className="flex-1 overflow-y-auto p-4 space-y-3">
        {messages.length === 0 && transfers.length === 0 && (
          <div className="text-center text-sm text-zinc-500 py-12">
            Hozircha xabarlar yo'q.<br />
            Birinchi bo'lib salom yozing yoki faylni shu yerga tashlang.
          </div>
        )}
        <AnimatePresence initial={false}>
          {messages.map((m, i) => (
            <Bubble key={`m-${i}`} msg={m} mine={m.isLocal || m.from === myPeerId} />
          ))}
          {transfers.map((t) => (
            <TransferRow key={`t-${t.peerId}-${t.xferId}-${t.direction}`} t={t} />
          ))}
        </AnimatePresence>
      </div>

      <div className="p-3 border-t border-white/5 flex gap-2 items-end">
        <button
          onClick={sendFile}
          disabled={peers.length === 0}
          title="Fayl yuborish"
          className="h-10 w-10 rounded-btn flex items-center justify-center text-zinc-300 hover:bg-white/[0.05] disabled:opacity-40"
        >
          <Paperclip className="w-4 h-4" strokeWidth={2} />
        </button>
        <textarea
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              send();
            }
          }}
          rows={1}
          placeholder="Xabar yozing..."
          className="input-base flex-1 resize-none min-h-[40px] max-h-[120px]"
        />
        <button
          onClick={send}
          className="btn-primary rounded-btn px-4 py-2.5 font-medium flex items-center gap-2 disabled:opacity-40"
          disabled={!draft.trim()}
        >
          <Send className="w-4 h-4" strokeWidth={2} />
        </button>
      </div>
    </div>
  );
}

function Bubble({ msg, mine }: { msg: ChatMessage; mine: boolean }) {
  const color = avatarColor(msg.nickname);
  const initial = avatarInitial(msg.nickname);
  return (
    <motion.div
      layout
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0 }}
      transition={{ type: "spring", stiffness: 320, damping: 28 }}
      className={`flex gap-2 ${mine ? "flex-row-reverse" : ""}`}
    >
      <div
        className="w-7 h-7 rounded-full flex items-center justify-center text-xs font-semibold text-white shrink-0"
        style={{ background: `linear-gradient(135deg, ${color}, #6366f1)` }}
      >
        {initial}
      </div>
      <div className={`min-w-0 max-w-[75%] ${mine ? "items-end" : "items-start"} flex flex-col`}>
        <div className="flex items-baseline gap-2 mb-0.5 px-0.5">
          <div className="text-xs text-zinc-400">{msg.nickname || "—"}</div>
          <div className="text-[10px] text-zinc-600 font-mono">{timeOfDay(msg.at)}</div>
        </div>
        <div
          className={`px-3 py-2 rounded-2xl text-sm break-words ${
            mine
              ? "bg-gradient-to-br from-violet-500/90 to-indigo-500/90 text-white"
              : "bg-white/[0.06] border border-white/10"
          }`}
        >
          {msg.text}
        </div>
      </div>
    </motion.div>
  );
}
