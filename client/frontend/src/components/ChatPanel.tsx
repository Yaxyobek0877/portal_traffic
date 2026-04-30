import React, { useEffect, useRef, useState } from "react";
import { Send } from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import type { ChatMessage } from "../types";
import { app } from "../lib/wails";
import { avatarColor, avatarInitial } from "../lib/avatar";
import { timeOfDay } from "../lib/format";

type Props = {
  messages: ChatMessage[];
  myPeerId: string;
};

export function ChatPanel({ messages, myPeerId }: Props) {
  const [draft, setDraft] = useState("");
  const scrollerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const el = scrollerRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [messages.length]);

  const send = async () => {
    const text = draft.trim();
    if (!text) return;
    setDraft("");
    await app.SendChat(text);
  };

  return (
    <div className="h-full flex flex-col">
      <div ref={scrollerRef} className="flex-1 overflow-y-auto p-4 space-y-3">
        {messages.length === 0 && (
          <div className="text-center text-sm text-zinc-500 py-12">
            Hozircha xabarlar yo'q.<br/>Birinchi bo'lib salom yozing.
          </div>
        )}
        <AnimatePresence initial={false}>
          {messages.map((m, i) => (
            <Bubble key={i} msg={m} mine={m.isLocal || m.from === myPeerId} />
          ))}
        </AnimatePresence>
      </div>

      <div className="p-3 border-t border-white/5 flex gap-2 items-end">
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
