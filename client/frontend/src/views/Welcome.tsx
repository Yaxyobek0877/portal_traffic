import React, { useEffect, useRef, useState } from "react";
import { motion } from "framer-motion";
import { Sparkles, LogIn, History as HistoryIcon, AlertTriangle } from "lucide-react";
import { Logo } from "../components/Logo";
import { app } from "../lib/wails";
import { usePortalStore } from "../stores/portalStore";
import { useT } from "../i18n";
import { parseInvite } from "../lib/deeplink";

type Mode = "idle" | "create" | "join";

export function Welcome() {
  const { t, lang, setLang } = useT();
  const nickname = usePortalStore((s) => s.nickname);
  const setNickname = usePortalStore((s) => s.setNickname);
  const setPortal = usePortalStore((s) => s.setPortal);
  const setScreen = usePortalStore((s) => s.setScreen);
  const setSignalingUrl = usePortalStore((s) => s.setSignalingUrl);
  const signalingUrl = usePortalStore((s) => s.signalingUrl);
  const history = usePortalStore((s) => s.history);
  const setHistory = usePortalStore((s) => s.setHistory);
  const nat = usePortalStore((s) => s.nat);

  const [mode, setMode] = useState<Mode>("idle");
  const [portalId, setPortalId] = useState("");
  const [code, setCode] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  // generation counter so we can ignore a late-arriving result from a
  // connect attempt the user already cancelled. Without this, clicking
  // "back" while the request is in flight just lets the eventual
  // success drag the user back to the portal view.
  const genRef = useRef(0);

  useEffect(() => {
    app.SignalingURL().then(setSignalingUrl);
    app.RecentPortals(5).then(setHistory);
  }, [setSignalingUrl, setHistory]);

  const reuse = (id: string, c: string) => {
    setMode("join");
    setPortalId(id);
    setCode(c);
  };

  // When the user pastes a portal:// URL or formatted invite text into
  // either of the join inputs, split it across both fields. Avoids the
  // "type the digits one at a time" friction.
  const handleInvitePaste = (e: React.ClipboardEvent<HTMLInputElement>) => {
    const raw = e.clipboardData.getData("text");
    const parsed = parseInvite(raw);
    if (parsed && (parsed.code || /portal:\/\//i.test(raw))) {
      e.preventDefault();
      setPortalId(parsed.portalId.replace(/[^0-9]/g, "").slice(0, 6));
      if (parsed.code) {
        setCode(parsed.code.replace(/[^0-9]/g, "").slice(0, 6));
      }
    }
  };

  const cancel = async () => {
    genRef.current++;
    setBusy(false);
    setMode("idle");
    setError("");
    try {
      await app.Leave();
    } catch {}
  };

  const submit = async () => {
    setError("");
    if (!nickname.trim()) {
      setError(t("welcome.error.empty_nickname"));
      return;
    }
    if (mode === "join" && (!portalId.trim() || !code.trim())) {
      setError(t("welcome.error.empty_id_or_code"));
      return;
    }
    const myGen = ++genRef.current;
    setBusy(true);
    try {
      const p =
        mode === "create"
          ? await app.CreatePortal(nickname.trim(), false)
          : await app.JoinPortal(nickname.trim(), portalId.trim(), code.trim());
      if (myGen !== genRef.current) {
        return;
      }
      setPortal(p);
      setScreen("portal");
    } catch (e: any) {
      if (myGen !== genRef.current) return;
      setError(e?.message || String(e));
    } finally {
      if (myGen === genRef.current) setBusy(false);
    }
  };

  // Translate known server-side errors to user-friendly localized
  // strings; fall through to the raw message otherwise.
  const localizedError = (raw: string): string => {
    if (/no such portal/i.test(raw)) return t("welcome.error.no_such_portal");
    if (/code does not match|portal_code_wrong/i.test(raw)) return t("welcome.error.wrong_code");
    if (/portal_full/i.test(raw)) return t("welcome.error.full");
    if (/portal_locked/i.test(raw)) return t("welcome.error.locked");
    return raw;
  };

  return (
    <div className="h-full flex flex-col">
      <div className="draggable titlebar-pad flex justify-end items-center px-3 gap-1" style={{ height: 68 }}>
        {/* Language toggle only — full Settings panel is reachable
            from the Portal screen, so the login stays uncluttered.
            A user landing in the wrong language can flip it here
            without going hunting through Settings. */}
        <div className="no-drag flex rounded overflow-hidden border border-white/10 text-[11px] font-mono">
          <button
            type="button"
            onClick={() => setLang("uz")}
            className={`px-2 py-1 ${lang === "uz" ? "bg-violet-500/30 text-white" : "text-zinc-500 hover:bg-white/[0.04]"}`}
          >
            UZ
          </button>
          <button
            type="button"
            onClick={() => setLang("en")}
            className={`px-2 py-1 ${lang === "en" ? "bg-violet-500/30 text-white" : "text-zinc-500 hover:bg-white/[0.04]"}`}
          >
            EN
          </button>
        </div>
      </div>

      <div className="flex-1 flex items-center justify-center px-6 -mt-6">
        <div className="w-full max-w-[440px] text-center">
          <motion.div
            initial={{ opacity: 0, scale: 0.85 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ type: "spring", stiffness: 200, damping: 22 }}
          >
            <Logo size={170} />
          </motion.div>
          <motion.h1
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.15 }}
            className="text-4xl font-extrabold tracking-tight mt-2"
          >
            Portal
          </motion.h1>
          <motion.p
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ delay: 0.3 }}
            className="text-sm text-zinc-400 mt-2 whitespace-pre-line"
          >
            {t("welcome.tagline")}
          </motion.p>

          <div className="mt-8 space-y-3 text-left">
            <div>
              <label className="text-xs uppercase tracking-widest text-zinc-500 ml-1">
                {t("welcome.nickname.label")}
              </label>
              <input
                type="text"
                placeholder={t("welcome.nickname.placeholder")}
                value={nickname}
                onChange={(e) => setNickname(e.target.value)}
                className="input-base w-full mt-1.5"
                maxLength={24}
                autoFocus
              />
            </div>

            {mode === "join" && (
              <motion.div
                initial={{ opacity: 0, y: -6 }}
                animate={{ opacity: 1, y: 0 }}
                className="grid grid-cols-2 gap-2"
              >
                <div>
                  <label className="text-xs uppercase tracking-widest text-zinc-500 ml-1">
                    {t("welcome.portalId.label")}
                  </label>
                  <input
                    type="text"
                    placeholder={t("welcome.portalId.placeholder")}
                    value={portalId}
                    onChange={(e) => setPortalId(e.target.value.replace(/[^0-9]/g, "").slice(0, 6))}
                    onPaste={handleInvitePaste}
                    className="input-base w-full mt-1.5 font-mono text-center text-lg tracking-widest"
                    maxLength={6}
                  />
                </div>
                <div>
                  <label className="text-xs uppercase tracking-widest text-zinc-500 ml-1">
                    {t("welcome.code.label")}
                  </label>
                  <input
                    type="text"
                    placeholder={t("welcome.code.placeholder")}
                    value={code}
                    onChange={(e) => setCode(e.target.value.replace(/[^0-9]/g, "").slice(0, 6))}
                    onPaste={handleInvitePaste}
                    className="input-base w-full mt-1.5 font-mono text-center text-lg tracking-widest"
                    maxLength={6}
                  />
                </div>
              </motion.div>
            )}

            {error && (
              <div className="rounded-input border border-rose-500/30 bg-rose-500/5 p-3 text-xs space-y-2">
                <div className="text-rose-300">{localizedError(error)}</div>
                {/no such portal/i.test(error) && (
                  <div className="flex gap-2">
                    <button
                      onClick={() => {
                        setMode("create");
                        setError("");
                        setPortalId("");
                        setCode("");
                      }}
                      className="text-xs text-emerald-300 hover:text-emerald-200 underline"
                    >
                      {t("welcome.create_new_link")}
                    </button>
                  </div>
                )}
              </div>
            )}

            {mode === "idle" ? (
              <div className="grid grid-cols-2 gap-3 pt-3">
                <button
                  onClick={() => setMode("create")}
                  className="btn-primary rounded-btn h-12 font-semibold flex items-center justify-center gap-2"
                >
                  <Sparkles className="w-4 h-4" />
                  {t("welcome.create")}
                </button>
                <button
                  onClick={() => setMode("join")}
                  className="panel rounded-btn h-12 font-semibold flex items-center justify-center gap-2 hover:bg-white/[0.07]"
                >
                  <LogIn className="w-4 h-4" />
                  {t("welcome.join")}
                </button>
              </div>
            ) : (
              <div className="grid grid-cols-2 gap-3 pt-3">
                <button
                  onClick={busy ? cancel : () => setMode("idle")}
                  className="panel rounded-btn h-11 text-sm hover:bg-white/[0.07]"
                >
                  {busy ? t("welcome.cancel") : t("welcome.back")}
                </button>
                <button
                  onClick={submit}
                  disabled={busy}
                  className="btn-primary rounded-btn h-11 text-sm font-semibold disabled:opacity-50"
                >
                  {busy
                    ? t("common.connecting")
                    : mode === "create"
                    ? t("welcome.creating")
                    : t("welcome.joining")}
                </button>
              </div>
            )}
          </div>

          {history.length > 0 && mode === "idle" && (
            <div className="mt-8 text-left">
              <div className="flex items-center gap-1.5 text-xs uppercase tracking-widest text-zinc-500 mb-2">
                <HistoryIcon className="w-3 h-3" />
                {t("welcome.recent")}
              </div>
              <div className="space-y-1.5">
                {history.slice(0, 4).map((h) => (
                  <button
                    key={h.id}
                    onClick={() => h.code && reuse(h.portalId, h.code)}
                    disabled={!h.code}
                    className="w-full panel rounded-input px-3 py-2 flex items-center justify-between text-sm hover:bg-white/[0.07] disabled:opacity-50"
                  >
                    <div className="flex items-center gap-2 min-w-0">
                      <span className="font-mono text-violet-300">{h.portalId}</span>
                      <span className="text-zinc-500 truncate">{h.nickname}</span>
                    </div>
                    {h.isOwner && (
                      <span className="text-[10px] uppercase tracking-wider text-amber-400">
                        {t("common.owner")}
                      </span>
                    )}
                  </button>
                ))}
              </div>
            </div>
          )}

          {nat?.type === 3 && (
            <div className="mt-6 panel rounded-input p-3 text-left flex gap-2 items-start border-amber-500/20 bg-amber-500/5">
              <AlertTriangle className="w-4 h-4 text-amber-400 shrink-0 mt-0.5" strokeWidth={2} />
              <div className="text-[11px] text-amber-200/90 leading-relaxed">
                {t("welcome.symmetric_nat_warning")}
              </div>
            </div>
          )}

          <div className="text-xs text-zinc-600 mt-6 font-mono truncate">{signalingUrl}</div>
        </div>
      </div>
    </div>
  );
}
