// Local vault unlock screen. Sits in front of Welcome on every launch:
//   - First launch (no password set yet) → "setup" mode, two fields,
//     validate match, call SetPassword, flip unlocked.
//   - Subsequent launches → "unlock" mode, one field, VerifyPassword,
//     flip unlocked on success. "Forgot?" runs ResetVault and bumps
//     the user back into setup mode.
//
// The whole component is gated by App.tsx — when usePortalStore.unlocked
// is true, this view is unmounted and Welcome takes over.

import React, { useEffect, useRef, useState } from "react";
import { motion } from "framer-motion";
import { Lock as LockIcon, ShieldCheck, KeyRound, AlertTriangle } from "lucide-react";
import { Logo } from "../components/Logo";
import { app } from "../lib/wails";
import { usePortalStore } from "../stores/portalStore";
import { useT } from "../i18n";

type Mode = "loading" | "setup" | "unlock";

export function Lock() {
  const { t, lang, setLang } = useT();
  const setUnlocked = usePortalStore((s) => s.setUnlocked);

  const [mode, setMode] = useState<Mode>("loading");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [resetOpen, setResetOpen] = useState(false);

  const inputRef = useRef<HTMLInputElement>(null);

  // Probe the backend once on mount to pick the mode. We do this in
  // a useEffect rather than at App.tsx level so the lock screen stays
  // self-contained — App.tsx only needs to know "is the vault open?".
  useEffect(() => {
    let cancelled = false;
    app
      .HasPassword()
      .then((has) => {
        if (cancelled) return;
        setMode(has ? "unlock" : "setup");
      })
      .catch(() => {
        // Defensive: if the backend isn't reachable, default to setup
        // so the user can at least proceed past the screen.
        if (!cancelled) setMode("setup");
      });
    return () => {
      cancelled = true;
    };
  }, []);

  // Re-focus the password input when the mode flips (e.g. from
  // loading→setup, or after a reset). Without this the user has to
  // click into the field on first render which is annoying for a
  // password screen they'll see every launch.
  useEffect(() => {
    if (mode === "setup" || mode === "unlock") {
      inputRef.current?.focus();
    }
  }, [mode]);

  const submit = async () => {
    setError("");
    if (mode === "setup") {
      if (password.trim().length < 4) {
        setError(t("lock.setup.error.too_short"));
        return;
      }
      if (password !== confirm) {
        setError(t("lock.setup.error.mismatch"));
        return;
      }
      setBusy(true);
      try {
        await app.SetPassword(password);
        setUnlocked(true);
      } catch (e: any) {
        const msg = e?.message || String(e);
        if (/password_too_short/.test(msg)) {
          setError(t("lock.setup.error.too_short"));
        } else {
          setError(t("lock.setup.error.failed"));
        }
      } finally {
        setBusy(false);
      }
      return;
    }
    // unlock
    if (!password) {
      setError(t("lock.unlock.error.wrong"));
      return;
    }
    setBusy(true);
    try {
      const ok = await app.VerifyPassword(password);
      if (ok) {
        setUnlocked(true);
      } else {
        setError(t("lock.unlock.error.wrong"));
        setPassword("");
        inputRef.current?.focus();
      }
    } catch {
      setError(t("lock.unlock.error.wrong"));
    } finally {
      setBusy(false);
    }
  };

  const onKey = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") submit();
  };

  const doReset = async () => {
    setBusy(true);
    try {
      await app.ResetVault();
      // Wipe local UI state too — leftover password text in the field
      // would survive the mode flip otherwise.
      setPassword("");
      setConfirm("");
      setError("");
      setResetOpen(false);
      setMode("setup");
    } catch {
      setResetOpen(false);
      setError(t("lock.setup.error.failed"));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="h-full flex flex-col">
      {/* Same titlebar pattern as Welcome — drag region + UZ/EN toggle. */}
      <div className="draggable titlebar-pad flex justify-end items-center px-3 gap-1" style={{ height: 68 }}>
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
        <div className="w-full max-w-[420px] text-center">
          <motion.div
            initial={{ opacity: 0, scale: 0.85 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ type: "spring", stiffness: 200, damping: 22 }}
            className="relative inline-block"
          >
            <Logo size={140} />
            {/* Lock chip layered over the logo so the user reads the
                screen as "Portal — locked" at a glance. */}
            <div className="absolute -bottom-1 -right-1 bg-zinc-900 border border-violet-500/40 rounded-full p-2 shadow-lg">
              {mode === "setup" ? (
                <ShieldCheck className="w-4 h-4 text-violet-300" />
              ) : (
                <LockIcon className="w-4 h-4 text-violet-300" />
              )}
            </div>
          </motion.div>

          <motion.h1
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.15 }}
            className="text-2xl font-extrabold tracking-tight mt-4"
          >
            {mode === "setup" ? t("lock.setup.title") : t("lock.unlock.title")}
          </motion.h1>

          <motion.p
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ delay: 0.3 }}
            className="text-xs text-zinc-400 mt-2 leading-relaxed"
          >
            {mode === "setup" ? t("lock.setup.subtitle") : t("lock.unlock.subtitle")}
          </motion.p>

          {mode === "loading" && (
            <div className="mt-10 text-xs text-zinc-500">…</div>
          )}

          {(mode === "setup" || mode === "unlock") && (
            <div className="mt-6 space-y-3 text-left">
              <div>
                <label className="text-xs uppercase tracking-widest text-zinc-500 ml-1">
                  {mode === "setup"
                    ? t("lock.setup.password.label")
                    : t("lock.unlock.password.label")}
                </label>
                <input
                  ref={inputRef}
                  type="password"
                  placeholder={
                    mode === "setup"
                      ? t("lock.setup.password.placeholder")
                      : t("lock.unlock.password.placeholder")
                  }
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  onKeyDown={onKey}
                  className="input-base w-full mt-1.5"
                  maxLength={128}
                  autoComplete={mode === "setup" ? "new-password" : "current-password"}
                  disabled={busy}
                />
              </div>

              {mode === "setup" && (
                <motion.div
                  initial={{ opacity: 0, y: -4 }}
                  animate={{ opacity: 1, y: 0 }}
                >
                  <label className="text-xs uppercase tracking-widest text-zinc-500 ml-1">
                    {t("lock.setup.confirm.label")}
                  </label>
                  <input
                    type="password"
                    placeholder={t("lock.setup.confirm.placeholder")}
                    value={confirm}
                    onChange={(e) => setConfirm(e.target.value)}
                    onKeyDown={onKey}
                    className="input-base w-full mt-1.5"
                    maxLength={128}
                    autoComplete="new-password"
                    disabled={busy}
                  />
                </motion.div>
              )}

              {error && (
                <div className="rounded-input border border-rose-500/30 bg-rose-500/5 p-2.5 text-xs text-rose-300">
                  {error}
                </div>
              )}

              <button
                onClick={submit}
                disabled={busy}
                className="btn-primary rounded-btn h-11 w-full text-sm font-semibold flex items-center justify-center gap-2 disabled:opacity-50"
              >
                <KeyRound className="w-4 h-4" />
                {busy
                  ? mode === "setup"
                    ? t("lock.setup.submitting")
                    : t("lock.unlock.checking")
                  : mode === "setup"
                  ? t("lock.setup.submit")
                  : t("lock.unlock.submit")}
              </button>

              {mode === "setup" && (
                <p className="text-[11px] text-zinc-500 leading-relaxed pt-1">
                  {t("lock.setup.hint")}
                </p>
              )}

              {mode === "unlock" && (
                <button
                  type="button"
                  onClick={() => setResetOpen(true)}
                  className="text-xs text-zinc-500 hover:text-zinc-300 underline pt-1"
                >
                  {t("lock.unlock.forgot")}
                </button>
              )}
            </div>
          )}
        </div>
      </div>

      {resetOpen && (
        <div className="absolute inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <motion.div
            initial={{ opacity: 0, scale: 0.95 }}
            animate={{ opacity: 1, scale: 1 }}
            className="panel rounded-input p-5 max-w-[360px] mx-4"
          >
            <div className="flex gap-3 items-start">
              <AlertTriangle className="w-5 h-5 text-amber-400 shrink-0 mt-0.5" />
              <div className="flex-1">
                <div className="font-semibold text-sm">{t("lock.reset.title")}</div>
                <p className="text-xs text-zinc-400 mt-2 leading-relaxed">
                  {t("lock.reset.body")}
                </p>
              </div>
            </div>
            <div className="grid grid-cols-2 gap-2 mt-4">
              <button
                onClick={() => setResetOpen(false)}
                disabled={busy}
                className="panel rounded-btn h-10 text-xs hover:bg-white/[0.07] disabled:opacity-50"
              >
                {t("lock.reset.cancel")}
              </button>
              <button
                onClick={doReset}
                disabled={busy}
                className="rounded-btn h-10 text-xs font-semibold bg-rose-500/30 hover:bg-rose-500/50 text-rose-100 disabled:opacity-50"
              >
                {t("lock.reset.confirm")}
              </button>
            </div>
          </motion.div>
        </div>
      )}
    </div>
  );
}
