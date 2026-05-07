// Local-account gate. Sits in front of Welcome on every launch:
//   - First launch (no account configured) → Sign-Up mode: username +
//     password + confirm. SignUp creates the row, flips unlocked, and
//     pre-fills Welcome's nickname with the username.
//   - Subsequent launches → Sign-In mode: username + password. SignIn
//     verifies both; on failure the password field is cleared and
//     refocused. "Forgot password?" runs ResetVault and bumps the
//     user back into Sign-Up.
//
// The whole component is gated by App.tsx — when usePortalStore.unlocked
// flips true, this view is unmounted and Welcome takes over.

import React, { useEffect, useRef, useState } from "react";
import { motion } from "framer-motion";
import { Lock as LockIcon, ShieldCheck, KeyRound, AlertTriangle, User } from "lucide-react";
import { Logo } from "../components/Logo";
import { app } from "../lib/wails";
import { usePortalStore } from "../stores/portalStore";
import { useT } from "../i18n";

type Mode = "loading" | "signup" | "signin";

export function Lock() {
  const { t, lang, setLang } = useT();
  const setUnlocked = usePortalStore((s) => s.setUnlocked);
  const setNickname = usePortalStore((s) => s.setNickname);

  const [mode, setMode] = useState<Mode>("loading");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [resetOpen, setResetOpen] = useState(false);

  const usernameRef = useRef<HTMLInputElement>(null);
  const passwordRef = useRef<HTMLInputElement>(null);

  // Probe the backend once on mount: existing account → Sign-In;
  // otherwise → Sign-Up. We also pre-fill the username on Sign-In so
  // returning users only have to type their password.
  useEffect(() => {
    let cancelled = false;
    app
      .HasAccount()
      .then(async (has) => {
        if (cancelled) return;
        if (has) {
          const u = await app.CurrentUsername().catch(() => "");
          if (cancelled) return;
          setUsername(u);
          setMode("signin");
        } else {
          setMode("signup");
        }
      })
      .catch(() => {
        // Backend not reachable: default to Sign-Up so the user can
        // at least proceed past the screen.
        if (!cancelled) setMode("signup");
      });
    return () => {
      cancelled = true;
    };
  }, []);

  // Auto-focus the right field when the mode resolves. On Sign-In the
  // username is already pre-filled, so the password field gets focus;
  // on Sign-Up we start at the username field.
  useEffect(() => {
    if (mode === "signup") usernameRef.current?.focus();
    if (mode === "signin") passwordRef.current?.focus();
  }, [mode]);

  const submit = async () => {
    setError("");
    if (mode === "signup") {
      const uname = username.trim();
      if (!uname) {
        setError(t("lock.signup.error.username_empty"));
        return;
      }
      if ([...uname].length > 24) {
        setError(t("lock.signup.error.username_too_long"));
        return;
      }
      if (password.length < 4) {
        setError(t("lock.signup.error.password_too_short"));
        return;
      }
      if (password !== confirm) {
        setError(t("lock.signup.error.mismatch"));
        return;
      }
      setBusy(true);
      try {
        await app.SignUp(uname, password);
        setNickname(uname);
        setUnlocked(true);
      } catch (e: any) {
        const msg = (e?.message || String(e)).toLowerCase();
        if (msg.includes("username_empty")) {
          setError(t("lock.signup.error.username_empty"));
        } else if (msg.includes("username_too_long")) {
          setError(t("lock.signup.error.username_too_long"));
        } else if (msg.includes("password_too_short")) {
          setError(t("lock.signup.error.password_too_short"));
        } else {
          setError(t("lock.signup.error.failed"));
        }
      } finally {
        setBusy(false);
      }
      return;
    }
    // signin
    if (!username.trim() || !password) {
      setError(t("lock.signin.error.wrong"));
      return;
    }
    setBusy(true);
    try {
      const ok = await app.SignIn(username.trim(), password);
      if (ok) {
        setNickname(username.trim());
        setUnlocked(true);
      } else {
        setError(t("lock.signin.error.wrong"));
        setPassword("");
        passwordRef.current?.focus();
      }
    } catch {
      setError(t("lock.signin.error.wrong"));
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
      setUsername("");
      setPassword("");
      setConfirm("");
      setError("");
      setResetOpen(false);
      setMode("signup");
    } catch {
      setResetOpen(false);
      setError(t("lock.signup.error.failed"));
    } finally {
      setBusy(false);
    }
  };

  // Toggle between modes (the "I already have an account" / "Create one"
  // links). Wipes form state so the previous screen's password text
  // doesn't bleed across.
  const switchMode = (next: Mode) => {
    setError("");
    setPassword("");
    setConfirm("");
    setMode(next);
  };

  return (
    <div className="h-full flex flex-col">
      {/* Titlebar drag region. The OS traffic lights live in the left
          padding; we keep the right side for the secondary lang toggle.
          A primary, prominent language switcher lives below the form
          so users don't miss it on the small titlebar instance. */}
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
            <Logo size={120} />
            <div className="absolute -bottom-1 -right-1 bg-zinc-900 border border-violet-500/40 rounded-full p-2 shadow-lg">
              {mode === "signup" ? (
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
            {mode === "signup" ? t("lock.signup.title") : t("lock.signin.title")}
          </motion.h1>

          <motion.p
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ delay: 0.3 }}
            className="text-xs text-zinc-400 mt-2 leading-relaxed"
          >
            {mode === "signup" ? t("lock.signup.subtitle") : t("lock.signin.subtitle")}
          </motion.p>

          {mode === "loading" && <div className="mt-10 text-xs text-zinc-500">…</div>}

          {(mode === "signup" || mode === "signin") && (
            <div className="mt-5 space-y-3 text-left">
              <div>
                <label className="text-xs uppercase tracking-widest text-zinc-500 ml-1">
                  {mode === "signup"
                    ? t("lock.signup.username.label")
                    : t("lock.signin.username.label")}
                </label>
                <div className="relative mt-1.5">
                  <User className="w-4 h-4 text-zinc-500 absolute left-3 top-1/2 -translate-y-1/2 pointer-events-none" />
                  <input
                    ref={usernameRef}
                    type="text"
                    placeholder={
                      mode === "signup"
                        ? t("lock.signup.username.placeholder")
                        : t("lock.signin.username.placeholder")
                    }
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    onKeyDown={onKey}
                    className="input-base w-full pl-9"
                    maxLength={24}
                    autoComplete="username"
                    disabled={busy}
                  />
                </div>
              </div>

              <div>
                <label className="text-xs uppercase tracking-widest text-zinc-500 ml-1">
                  {mode === "signup"
                    ? t("lock.signup.password.label")
                    : t("lock.signin.password.label")}
                </label>
                <div className="relative mt-1.5">
                  <KeyRound className="w-4 h-4 text-zinc-500 absolute left-3 top-1/2 -translate-y-1/2 pointer-events-none" />
                  <input
                    ref={passwordRef}
                    type="password"
                    placeholder={
                      mode === "signup"
                        ? t("lock.signup.password.placeholder")
                        : t("lock.signin.password.placeholder")
                    }
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    onKeyDown={onKey}
                    className="input-base w-full pl-9"
                    maxLength={128}
                    autoComplete={mode === "signup" ? "new-password" : "current-password"}
                    disabled={busy}
                  />
                </div>
              </div>

              {mode === "signup" && (
                <motion.div
                  initial={{ opacity: 0, y: -4 }}
                  animate={{ opacity: 1, y: 0 }}
                >
                  <label className="text-xs uppercase tracking-widest text-zinc-500 ml-1">
                    {t("lock.signup.confirm.label")}
                  </label>
                  <input
                    type="password"
                    placeholder={t("lock.signup.confirm.placeholder")}
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
                  ? mode === "signup"
                    ? t("lock.signup.submitting")
                    : t("lock.signin.checking")
                  : mode === "signup"
                  ? t("lock.signup.submit")
                  : t("lock.signin.submit")}
              </button>

              {mode === "signup" && (
                <p className="text-[11px] text-zinc-500 leading-relaxed pt-1">
                  {t("lock.signup.hint")}
                </p>
              )}

              {mode === "signin" && (
                <button
                  type="button"
                  onClick={() => setResetOpen(true)}
                  className="text-xs text-zinc-500 hover:text-zinc-300 underline pt-1"
                >
                  {t("lock.signin.forgot")}
                </button>
              )}
            </div>
          )}

          {/* Primary, can't-miss language switcher. The titlebar one is
              tiny and easy to overlook on a Sign-Up screen the user
              has never seen before — this row makes it obvious. */}
          {mode !== "loading" && (
            <div className="mt-8 flex items-center justify-center gap-3 text-xs">
              <span className="text-zinc-500 uppercase tracking-widest text-[10px]">
                {t("lock.language")}
              </span>
              <div className="flex rounded-full overflow-hidden border border-white/10 font-mono">
                <button
                  type="button"
                  onClick={() => setLang("uz")}
                  className={`px-3 py-1 ${lang === "uz" ? "bg-violet-500/30 text-white" : "text-zinc-400 hover:bg-white/[0.06]"}`}
                >
                  O'zbek
                </button>
                <button
                  type="button"
                  onClick={() => setLang("en")}
                  className={`px-3 py-1 ${lang === "en" ? "bg-violet-500/30 text-white" : "text-zinc-400 hover:bg-white/[0.06]"}`}
                >
                  English
                </button>
              </div>
            </div>
          )}

          {/* Toggle between sign-up and sign-in. Useful both ways:
              after a Reset the user lands in Sign-Up but might just
              want to retry; on Sign-In they may want to reconsider. */}
          {mode !== "loading" && (
            <div className="mt-4">
              {mode === "signup" ? (
                <button
                  type="button"
                  onClick={() => switchMode("signin")}
                  className="text-xs text-zinc-500 hover:text-zinc-300 underline"
                >
                  {t("lock.signup.switch")}
                </button>
              ) : null}
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
