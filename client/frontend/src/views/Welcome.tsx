// Account-app dashboard. After unlock the user lands here, signed in
// as `nickname` (which == the username they registered with). The
// screen is divided into:
//
//   1. Header bar — branding, lang toggle, settings, sign-out, user
//      chip. Stays visible the whole time so the user always knows
//      they're signed in.
//   2. Hero — "Xush kelibsiz, <name>" greeting + short subtitle.
//   3. Action area — two big cards (Create / Join). Create is
//      one-click; Join expands inline into the ID/code form.
//   4. Recent portals — proper list with avatar circles and badges.
//      The dashboard's center of gravity, not a footnote.
//   5. Footer chrome — NAT warning + signaling URL, muted so it
//      doesn't compete with the action area.

import React, { useEffect, useRef, useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import {
  Sparkles,
  LogIn,
  History as HistoryIcon,
  AlertTriangle,
  Settings as SettingsIcon,
  LogOut,
  ArrowRight,
  Crown,
  Pencil,
  X,
  Check,
} from "lucide-react";
import { Logo } from "../components/Logo";
import { app } from "../lib/wails";
import { usePortalStore } from "../stores/portalStore";
import { useT } from "../i18n";
import { parseInvite } from "../lib/deeplink";
import type { HistoryEntry } from "../types";

type Mode = "idle" | "create" | "join";

export function Welcome() {
  const { t, lang, setLang } = useT();
  const nickname = usePortalStore((s) => s.nickname);
  const setPortal = usePortalStore((s) => s.setPortal);
  const setScreen = usePortalStore((s) => s.setScreen);
  const setUnlocked = usePortalStore((s) => s.setUnlocked);
  const setRemembered = usePortalStore((s) => s.setRemembered);
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
  // Which recent-row is currently being rejoined. Used to show a
  // per-row spinner instead of a global one — the rest of the list
  // stays clickable in case the network call fails and the user wants
  // to try a different recent.
  const [rejoiningId, setRejoiningId] = useState<number | null>(null);

  // Generation counter so a late-arriving result from a cancelled
  // connect attempt can't drag the user back into a portal they
  // bailed out of.
  const genRef = useRef(0);

  useEffect(() => {
    app.SignalingURL().then(setSignalingUrl);
    app.RecentPortals(8).then(setHistory);
  }, [setSignalingUrl, setHistory]);

  // One-click rejoin from the Recent list. Mirrors what the mobile
  // client does (PortalViewModel.rejoinRecent):
  //
  //  - Owner rows: the server destroys an owner's portal the moment
  //    they disconnect, so the saved portalId is dead. Best-effort
  //    re-create with the same nickname; user gets a fresh ID + code.
  //  - Joiner rows: just call JoinPortal with the saved id + code.
  //    If the portal has since closed we surface the existing
  //    "no such portal" error and the user can try a different row.
  //
  // The previous behaviour (pre-fill the join form, make the user
  // click submit again) was a holdover from before the dashboard
  // redesign — pure friction now.
  const rejoinRow = async (h: typeof history[number]) => {
    if (busy) return;
    setError("");
    setRejoiningId(h.id);
    const myGen = ++genRef.current;
    setBusy(true);
    try {
      // Use the signed-in user's nickname (post-account work this
      // is always populated). Fall back to the historical handle if
      // the field is somehow empty so we never call into the backend
      // with a blank name.
      const nick = nickname.trim() || h.nickname;
      const p = h.isOwner
        ? await app.CreatePortal(nick, false)
        : await app.JoinPortal(nick, h.portalId, h.code);
      if (myGen !== genRef.current) return;
      setPortal(p);
      setScreen("portal");
    } catch (e: any) {
      if (myGen !== genRef.current) return;
      setError(e?.message || String(e));
    } finally {
      if (myGen === genRef.current) {
        setBusy(false);
        setRejoiningId(null);
      }
    }
  };

  // Smart-paste: pasting a full portal:// URL or "ID-CODE" string into
  // either join input splits across both fields.
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
    setPortalId("");
    setCode("");
    try {
      await app.Leave();
    } catch {}
  };

  // One-click Create. Skips the old idle→create→submit pivot — the
  // dashboard primary action shouldn't make the user click twice.
  const handleCreate = async () => {
    setError("");
    if (!nickname.trim()) {
      setError(t("welcome.error.empty_nickname"));
      return;
    }
    setMode("create");
    const myGen = ++genRef.current;
    setBusy(true);
    try {
      const p = await app.CreatePortal(nickname.trim(), false);
      if (myGen !== genRef.current) return;
      setPortal(p);
      setScreen("portal");
    } catch (e: any) {
      if (myGen !== genRef.current) return;
      setError(e?.message || String(e));
      setMode("idle");
    } finally {
      if (myGen === genRef.current) setBusy(false);
    }
  };

  const handleJoinSubmit = async () => {
    setError("");
    if (!portalId.trim() || !code.trim()) {
      setError(t("welcome.error.empty_id_or_code"));
      return;
    }
    const myGen = ++genRef.current;
    setBusy(true);
    try {
      const p = await app.JoinPortal(nickname.trim(), portalId.trim(), code.trim());
      if (myGen !== genRef.current) return;
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
    if (/code does not match|portal_code_wrong/i.test(raw))
      return t("welcome.error.wrong_code");
    if (/portal_full/i.test(raw)) return t("welcome.error.full");
    if (/portal_locked/i.test(raw)) return t("welcome.error.locked");
    return raw;
  };

  // Sign out flips the vault back to the locked state. We don't reset
  // the account or wipe history — the user can sign back in with the
  // same credentials. The remembered flag is also cleared, otherwise
  // the next cold start would skip the lock and we'd silently undo
  // the sign-out the user just asked for. If they're somehow
  // connecting when they hit sign-out, Leave() tears it down cleanly.
  const signOut = async () => {
    try {
      await app.Leave();
    } catch {}
    setRemembered(false);
    setUnlocked(false);
  };

  const initial = (nickname.trim()[0] || "?").toUpperCase();

  return (
    <div className="h-full flex flex-col">
      {/* Header bar — always present, signals "you're signed in".
          Drag region on the left half (around the brand), no-drag
          on the right half so the toolbar buttons stay clickable. */}
      <header
        className="draggable titlebar-pad px-4 flex items-center justify-between border-b border-white/[0.04]"
        style={{ height: 68 }}
      >
        <div className="no-drag flex items-center gap-2">
          <Logo size={26} />
          <span className="font-extrabold text-sm tracking-tight">Portal</span>
        </div>

        <div className="no-drag flex items-center gap-1.5">
          {/* Lang toggle stays compact in the header — same control
              the Lock screen uses. */}
          <div className="flex rounded overflow-hidden border border-white/10 text-[11px] font-mono">
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

          <div className="h-5 w-px bg-white/10 mx-1" />

          <button
            onClick={() => setScreen("settings")}
            title={t("common.tooltip.settings")}
            className="p-2 rounded hover:bg-white/[0.05] text-zinc-400 hover:text-zinc-200 transition"
          >
            <SettingsIcon className="w-4 h-4" strokeWidth={2} />
          </button>

          <button
            onClick={signOut}
            title={t("welcome.signout")}
            className="p-2 rounded hover:bg-rose-500/10 text-zinc-400 hover:text-rose-300 transition"
          >
            <LogOut className="w-4 h-4" strokeWidth={2} />
          </button>

          <div className="h-5 w-px bg-white/10 mx-1" />

          {/* User chip — letter avatar + handle. Subtly says "this is
              the account you're signed in as". */}
          <div className="flex items-center gap-2 pl-1.5 pr-2.5 py-1 rounded-full bg-white/[0.04] border border-white/[0.06]">
            <div className="w-6 h-6 rounded-full bg-gradient-to-br from-violet-500/40 to-cyan-500/30 flex items-center justify-center text-[11px] font-extrabold text-white">
              {initial}
            </div>
            <span className="text-xs font-mono text-zinc-200 max-w-[140px] truncate">
              {nickname || "—"}
            </span>
          </div>
        </div>
      </header>

      {/* Body */}
      <main className="flex-1 overflow-y-auto px-6 pb-6">
        <div className="max-w-2xl mx-auto pt-6">
          {/* Hero greeting */}
          <motion.div
            initial={{ opacity: 0, y: 6 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.25 }}
          >
            <h1 className="text-2xl font-extrabold tracking-tight">
              {t("welcome.greeting")}, <span className="gradient-text">{nickname || "?"}</span>
            </h1>
            <p className="text-sm text-zinc-400 mt-1.5">{t("welcome.subtitle")}</p>
          </motion.div>

          {/* Recent portals — promoted ABOVE the action cards because
              for returning users (who have history) rejoining is the
              primary intent, not creating a new portal. New users
              with no recents still see the action cards prominently
              after the empty-state placeholder. */}
          {history.length > 0 && (
            <section className="mt-6">
              <div className="flex items-center gap-1.5 text-xs uppercase tracking-widest text-zinc-500 mb-2">
                <HistoryIcon className="w-3 h-3" />
                {t("welcome.recent")}
              </div>
              <div className="space-y-1.5">
                {history.slice(0, 6).map((h) => (
                  <RecentRow
                    key={h.id}
                    h={h}
                    busy={busy}
                    rejoining={rejoiningId === h.id}
                    onRejoin={() => rejoinRow(h)}
                    onRenamed={async () => {
                      // Pull a fresh list so the row's label updates
                      // in place without forcing the parent to track
                      // edits manually.
                      setHistory(await app.RecentPortals(8));
                    }}
                    onRemoved={async () => {
                      setHistory(await app.RecentPortals(8));
                    }}
                  />
                ))}
              </div>
            </section>
          )}

          {/* Action area — Idle: 2 cards. Join: inline form. Create
              busy state shows on the create card itself, no view swap.
              When the user already has recent portals these are the
              "make something new" path; when they don't, this is the
              only thing that matters. */}
          <div className="mt-6">
            <AnimatePresence mode="wait" initial={false}>
              {mode !== "join" ? (
                <motion.div
                  key="cards"
                  initial={{ opacity: 0, y: 4 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -4 }}
                  transition={{ duration: 0.18 }}
                  className="grid grid-cols-1 sm:grid-cols-2 gap-3"
                >
                  {/* Create card — primary, gradient border */}
                  <button
                    onClick={handleCreate}
                    disabled={busy}
                    className="group relative text-left rounded-input p-4 panel hover:bg-white/[0.06] transition disabled:opacity-60 disabled:cursor-wait overflow-hidden"
                  >
                    <div className="absolute inset-0 opacity-0 group-hover:opacity-100 transition pointer-events-none bg-gradient-to-br from-violet-500/10 to-cyan-500/5" />
                    <div className="relative">
                      <div className="w-9 h-9 rounded-md bg-violet-500/20 border border-violet-500/30 flex items-center justify-center">
                        <Sparkles className="w-4 h-4 text-violet-300" />
                      </div>
                      <div className="mt-3 flex items-center gap-2">
                        <span className="font-semibold text-sm">{t("welcome.create.card_title")}</span>
                        {busy && mode === "create" ? (
                          <span className="text-xs text-violet-300 ml-auto animate-pulse">
                            {t("common.connecting")}
                          </span>
                        ) : (
                          <ArrowRight className="w-4 h-4 ml-auto text-zinc-500 group-hover:text-violet-300 transition group-hover:translate-x-0.5" />
                        )}
                      </div>
                      <p className="text-xs text-zinc-400 mt-1.5 leading-relaxed">
                        {t("welcome.create.card_hint")}
                      </p>
                    </div>
                  </button>

                  {/* Join card — secondary */}
                  <button
                    onClick={() => {
                      setError("");
                      setMode("join");
                    }}
                    disabled={busy}
                    className="group relative text-left rounded-input p-4 panel hover:bg-white/[0.06] transition disabled:opacity-60 disabled:cursor-wait"
                  >
                    <div className="w-9 h-9 rounded-md bg-cyan-500/15 border border-cyan-500/25 flex items-center justify-center">
                      <LogIn className="w-4 h-4 text-cyan-300" />
                    </div>
                    <div className="mt-3 flex items-center gap-2">
                      <span className="font-semibold text-sm">{t("welcome.join.card_title")}</span>
                      <ArrowRight className="w-4 h-4 ml-auto text-zinc-500 group-hover:text-cyan-300 transition group-hover:translate-x-0.5" />
                    </div>
                    <p className="text-xs text-zinc-400 mt-1.5 leading-relaxed">
                      {t("welcome.join.card_hint")}
                    </p>
                  </button>
                </motion.div>
              ) : (
                /* Inline join form — replaces the cards in place rather
                   than navigating to a separate screen, so the mental
                   model "I'm joining a portal" is uninterrupted. */
                <motion.div
                  key="joinform"
                  initial={{ opacity: 0, y: 4 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -4 }}
                  transition={{ duration: 0.18 }}
                  className="panel rounded-input p-4 space-y-3"
                >
                  <div className="flex items-center gap-2">
                    <div className="w-8 h-8 rounded-md bg-cyan-500/15 border border-cyan-500/25 flex items-center justify-center">
                      <LogIn className="w-4 h-4 text-cyan-300" />
                    </div>
                    <div>
                      <div className="text-sm font-semibold">{t("welcome.join.card_title")}</div>
                      <div className="text-[11px] text-zinc-500">{t("welcome.join.card_hint")}</div>
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-2">
                    <div>
                      <label className="text-[11px] uppercase tracking-widest text-zinc-500 ml-1">
                        {t("welcome.portalId.label")}
                      </label>
                      <input
                        type="text"
                        placeholder={t("welcome.portalId.placeholder")}
                        value={portalId}
                        onChange={(e) =>
                          setPortalId(e.target.value.replace(/[^0-9]/g, "").slice(0, 6))
                        }
                        onPaste={handleInvitePaste}
                        className="input-base w-full mt-1.5 font-mono text-center text-lg tracking-widest"
                        maxLength={6}
                        autoFocus
                      />
                    </div>
                    <div>
                      <label className="text-[11px] uppercase tracking-widest text-zinc-500 ml-1">
                        {t("welcome.code.label")}
                      </label>
                      <input
                        type="text"
                        placeholder={t("welcome.code.placeholder")}
                        value={code}
                        onChange={(e) =>
                          setCode(e.target.value.replace(/[^0-9]/g, "").slice(0, 6))
                        }
                        onPaste={handleInvitePaste}
                        className="input-base w-full mt-1.5 font-mono text-center text-lg tracking-widest"
                        maxLength={6}
                      />
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-2 pt-1">
                    <button
                      onClick={busy ? cancel : () => setMode("idle")}
                      className="panel rounded-btn h-10 text-sm hover:bg-white/[0.07]"
                    >
                      {busy ? t("welcome.cancel") : t("welcome.back")}
                    </button>
                    <button
                      onClick={handleJoinSubmit}
                      disabled={busy}
                      className="btn-primary rounded-btn h-10 text-sm font-semibold disabled:opacity-50"
                    >
                      {busy ? t("common.connecting") : t("welcome.joining")}
                    </button>
                  </div>
                </motion.div>
              )}
            </AnimatePresence>

            {error && (
              <motion.div
                initial={{ opacity: 0, y: -4 }}
                animate={{ opacity: 1, y: 0 }}
                className="mt-3 rounded-input border border-rose-500/30 bg-rose-500/5 p-3 text-xs space-y-2"
              >
                <div className="text-rose-300">{localizedError(error)}</div>
                {/no such portal/i.test(error) && (
                  <button
                    onClick={() => {
                      setMode("idle");
                      setError("");
                      setPortalId("");
                      setCode("");
                      handleCreate();
                    }}
                    className="text-xs text-emerald-300 hover:text-emerald-200 underline"
                  >
                    {t("welcome.create_new_link")}
                  </button>
                )}
              </motion.div>
            )}
          </div>

          {/* First-time empty state. Only shown for users with no
              history yet — once they have any recents the section
              above takes over. The dashed-border hint teaches what
              the section is for, not a 'broken' look. */}
          {history.length === 0 && (
            <section className="mt-8">
              <div className="flex items-center gap-1.5 text-xs uppercase tracking-widest text-zinc-500 mb-2">
                <HistoryIcon className="w-3 h-3" />
                {t("welcome.recent")}
              </div>
              <div className="rounded-input border border-dashed border-white/[0.08] p-6 text-center text-xs text-zinc-500">
                {t("welcome.recent.empty")}
              </div>
            </section>
          )}

          {/* Footer chrome — NAT warning + signaling URL, muted so it
              fades into the background rather than competing with
              the action area. */}
          <footer className="mt-8 pt-4 border-t border-white/[0.05] space-y-2.5">
            {nat?.type === 3 && (
              <div className="rounded-input p-3 flex gap-2 items-start border border-amber-500/20 bg-amber-500/5">
                <AlertTriangle className="w-4 h-4 text-amber-400 shrink-0 mt-0.5" strokeWidth={2} />
                <div className="text-[11px] text-amber-200/90 leading-relaxed">
                  {t("welcome.symmetric_nat_warning")}
                </div>
              </div>
            )}

            <div className="text-[10px] text-zinc-600 font-mono text-center truncate">
              {signalingUrl}
            </div>
          </footer>
        </div>
      </main>
    </div>
  );
}

// RecentRow — one entry on the dashboard's "Recent portals" list.
//
// Three modes share one row:
//
//   1. Display — clicking the row body rejoins, the pencil reveals
//      the rename input, the X removes the entry.
//   2. Rename  — text input + check / cancel. Enter submits, Escape
//      cancels. The check is disabled while the value matches the
//      current label so accidental no-op saves don't blip the UI.
//   3. Busy    — spinner replacing the chevron while a rejoin call
//      is in flight; the controls stay clickable so the user can
//      still cancel if they meant a different row.
function RecentRow({
  h,
  busy,
  rejoining,
  onRejoin,
  onRenamed,
  onRemoved,
}: {
  h: HistoryEntry;
  busy: boolean;
  rejoining: boolean;
  onRejoin: () => void;
  onRenamed: () => void | Promise<void>;
  onRemoved: () => void | Promise<void>;
}) {
  const { t } = useT();
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(h.label || "");
  const [saving, setSaving] = useState(false);

  // When the underlying row's label changes (e.g. another window
  // edited it) reflect that in the draft as long as the user isn't
  // mid-edit. We don't want to clobber unsaved input.
  useEffect(() => {
    if (!editing) setDraft(h.label || "");
  }, [h.label, editing]);

  const peerInitial = (h.label || h.nickname || "?").trim()[0]?.toUpperCase() || "?";
  const canRejoin = h.isOwner || !!h.code;
  const displayLabel = h.label?.trim() || "";

  const submitRename = async () => {
    const next = draft.trim();
    if (next === (h.label || "")) {
      setEditing(false);
      return;
    }
    setSaving(true);
    try {
      await app.RenamePortal(h.id, next);
      await onRenamed();
      setEditing(false);
    } catch {
      // Leave editing open so the user can retry. Errors are usually
      // length-validation; the input keeps the offending text so they
      // can shorten it.
    } finally {
      setSaving(false);
    }
  };

  const cancelRename = () => {
    setDraft(h.label || "");
    setEditing(false);
  };

  // Container is a div, not a button, while editing — nesting buttons
  // breaks click handling and bubbling pencil/X clicks up to a row-
  // level rejoin would be hostile.
  if (editing) {
    return (
      <div className="w-full panel rounded-input px-3 py-2.5 flex items-center gap-3 text-sm">
        <div className="w-8 h-8 rounded-full bg-gradient-to-br from-violet-500/30 to-indigo-500/20 flex items-center justify-center text-[11px] font-bold shrink-0">
          {peerInitial}
        </div>
        <div className="min-w-0 flex-1 flex items-center gap-2">
          <input
            type="text"
            autoFocus
            value={draft}
            onChange={(e) => setDraft(e.target.value.slice(0, 60))}
            onKeyDown={(e) => {
              if (e.key === "Enter") submitRename();
              if (e.key === "Escape") cancelRename();
            }}
            placeholder={t("welcome.rename.placeholder")}
            className="input-base flex-1 text-sm py-1.5"
            maxLength={60}
            disabled={saving}
          />
        </div>
        <button
          type="button"
          onClick={submitRename}
          disabled={saving}
          title={t("welcome.rename.save")}
          className="p-1.5 rounded-md text-emerald-300 hover:bg-emerald-500/15 disabled:opacity-50"
        >
          <Check className="w-4 h-4" />
        </button>
        <button
          type="button"
          onClick={cancelRename}
          disabled={saving}
          title={t("welcome.rename.cancel")}
          className="p-1.5 rounded-md text-zinc-400 hover:bg-white/[0.07] disabled:opacity-50"
        >
          <X className="w-4 h-4" />
        </button>
      </div>
    );
  }

  return (
    <div className="w-full panel rounded-input pl-3 pr-1.5 py-2.5 flex items-center gap-3 text-sm hover:bg-white/[0.07] group">
      <button
        type="button"
        onClick={() => canRejoin && onRejoin()}
        disabled={!canRejoin || (busy && !rejoining)}
        className="flex items-center gap-3 min-w-0 flex-1 text-left disabled:opacity-50 disabled:cursor-not-allowed"
      >
        <div className="w-8 h-8 rounded-full bg-gradient-to-br from-violet-500/30 to-indigo-500/20 flex items-center justify-center text-[11px] font-bold shrink-0">
          {peerInitial}
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            {/* If the user labelled this portal we lead with the label
                and demote the numeric id to a small monospace tail.
                Unlabelled rows still show the id prominently — labels
                are optional, the row should never feel empty. */}
            {displayLabel ? (
              <>
                <span className="font-medium truncate">{displayLabel}</span>
                <span className="font-mono text-[10px] text-zinc-500 shrink-0">
                  #{h.portalId}
                </span>
              </>
            ) : (
              <span className="font-mono text-violet-300 text-sm">{h.portalId}</span>
            )}
            {h.isOwner && (
              <span className="inline-flex items-center gap-1 text-[10px] uppercase tracking-wider text-amber-400 bg-amber-500/10 border border-amber-500/20 rounded px-1.5 py-0.5 shrink-0">
                <Crown className="w-2.5 h-2.5" />
                {t("common.owner")}
              </span>
            )}
          </div>
          <div className="text-[11px] text-zinc-500 truncate">{h.nickname}</div>
        </div>
      </button>
      {/* Inline tools — only visible on hover/focus to keep the row
          quiet at rest. Pencil opens rename, X drops the entry. */}
      <div className="flex items-center opacity-0 group-hover:opacity-100 focus-within:opacity-100 transition shrink-0">
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation();
            setEditing(true);
          }}
          title={t("welcome.rename.title")}
          className="p-1.5 rounded-md text-zinc-400 hover:text-violet-300 hover:bg-white/[0.07]"
        >
          <Pencil className="w-3.5 h-3.5" />
        </button>
        <button
          type="button"
          onClick={async (e) => {
            e.stopPropagation();
            await app.RemoveRecentPortal(h.id);
            await onRemoved();
          }}
          title={t("welcome.recent.remove")}
          className="p-1.5 rounded-md text-zinc-400 hover:text-rose-300 hover:bg-rose-500/10"
        >
          <X className="w-3.5 h-3.5" />
        </button>
      </div>
      {rejoining ? (
        <div className="w-4 h-4 border-2 border-violet-300/30 border-t-violet-300 rounded-full animate-spin shrink-0 mr-2" />
      ) : (
        <ArrowRight className="w-4 h-4 text-zinc-600 group-hover:text-zinc-300 transition shrink-0 mr-2" />
      )}
    </div>
  );
}
