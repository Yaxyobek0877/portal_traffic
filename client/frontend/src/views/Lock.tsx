// Local-account gate — Sign-In / Sign-Up tabbed card. Sits in front
// of Welcome on every launch.
//
// Layout: a single auth card with a two-tab segmented control at the
// top ([Kirish] | [Ro'yxatdan o'tish]) and the form below. Tab choice
// resolves on mount from app.HasAccount() — returning users land on
// Sign-In with the username pre-filled; first-time users land on
// Sign-Up. Free switching between tabs from there. Signing up while
// an account already exists overwrites it — we warn before submitting.
//
// Language toggle lives only in the titlebar (consistent with Welcome);
// no bottom pill anymore.

import React, { useEffect, useRef, useState } from "react";
import { motion } from "framer-motion";
import {
  Lock as LockIcon,
  ShieldCheck,
  KeyRound,
  AlertTriangle,
  User,
  ArrowRight,
  Check,
  X as XIcon,
} from "lucide-react";
import { Logo } from "../components/Logo";
import { app } from "../lib/wails";
import { usePortalStore } from "../stores/portalStore";
import { useT } from "../i18n";

type Tab = "signin" | "signup";

export function Lock() {
  const { t, lang, setLang } = useT();
  const setUnlocked = usePortalStore((s) => s.setUnlocked);
  const setNickname = usePortalStore((s) => s.setNickname);
  const setRemembered = usePortalStore((s) => s.setRemembered);

  // Initial state is "loading" so we don't flash the wrong tab while
  // the HasAccount probe is in flight. The form is rendered behind a
  // skeleton; tab buttons fade in once we know the right default.
  const [tab, setTab] = useState<Tab | "loading">("loading");
  const [hasAccount, setHasAccount] = useState(false);

  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  // Default ON — most users want to skip the lock on subsequent
  // launches. Anyone worried about a shared laptop can untick it.
  const [rememberMe, setRememberMe] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [resetOpen, setResetOpen] = useState(false);
  // Lockout countdown after 5 failed sign-ins. The submit button
  // disables and a timer ticks down. Backend tracks the absolute
  // expiry — we just decrement to show progress.
  const [lockoutSec, setLockoutSec] = useState(0);

  // Tick the lockout countdown once a second.
  useEffect(() => {
    if (lockoutSec <= 0) return;
    const timer = setTimeout(() => setLockoutSec((s) => Math.max(0, s - 1)), 1000);
    return () => clearTimeout(timer);
  }, [lockoutSec]);

  // Live password-strength evaluation. Mirrors validatePasswordStrength
  // in client/auth.go so the checklist matches what the backend will
  // accept on submit. unicode regex categories cover non-ASCII letters
  // for usernames that bring those in.
  const strength = {
    length: password.length >= 8,
    lower: /\p{Ll}/u.test(password),
    upper: /\p{Lu}/u.test(password),
    digit: /[0-9]/.test(password),
    special: /[^\p{L}\p{N}\s]/u.test(password),
  };
  const allStrong =
    strength.length && strength.lower && strength.upper && strength.digit && strength.special;

  const usernameRef = useRef<HTMLInputElement>(null);
  const passwordRef = useRef<HTMLInputElement>(null);

  // Probe the backend once on mount: existing account → Sign-In tab,
  // pre-filled username; otherwise → Sign-Up tab.
  useEffect(() => {
    let cancelled = false;
    app
      .HasAccount()
      .then(async (has) => {
        if (cancelled) return;
        setHasAccount(has);
        if (has) {
          const u = await app.CurrentUsername().catch(() => "");
          if (cancelled) return;
          setUsername(u);
          setTab("signin");
        } else {
          setTab("signup");
        }
      })
      .catch(() => {
        // Backend not reachable: default to Sign-Up so the user can
        // at least proceed past the screen.
        if (!cancelled) setTab("signup");
      });
    return () => {
      cancelled = true;
    };
  }, []);

  // Auto-focus the right field when the tab changes. On Sign-In the
  // username is usually already filled, so focus jumps to password;
  // on Sign-Up we start at the username field.
  useEffect(() => {
    if (tab === "signup") usernameRef.current?.focus();
    if (tab === "signin") passwordRef.current?.focus();
  }, [tab]);

  // Switch tab. Wipes the password fields so the previous mode's
  // text doesn't bleed into the next one (common phishing-style
  // foot-gun). Username is preserved — it's not sensitive and the
  // user often wants to re-use it on the other tab.
  const switchTab = (next: Tab) => {
    if (next === tab || busy) return;
    setError("");
    setPassword("");
    setConfirm("");
    setTab(next);
  };

  const submit = async () => {
    setError("");
    if (tab === "signup") {
      const uname = username.trim();
      if (!uname) {
        setError(t("lock.signup.error.username_empty"));
        return;
      }
      if ([...uname].length > 24) {
        setError(t("lock.signup.error.username_too_long"));
        return;
      }
      if (password.length < 8) {
        setError(t("lock.signup.error.password_too_short"));
        return;
      }
      if (!allStrong) {
        setError(t("lock.signup.error.password_weak"));
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
        setRemembered(rememberMe);
        setUnlocked(true);
      } catch (e: any) {
        const msg = (e?.message || String(e)).toLowerCase();
        if (msg.includes("username_empty")) {
          setError(t("lock.signup.error.username_empty"));
        } else if (msg.includes("username_too_long")) {
          setError(t("lock.signup.error.username_too_long"));
        } else if (msg.includes("password_too_short")) {
          setError(t("lock.signup.error.password_too_short"));
        } else if (msg.includes("password_weak")) {
          setError(t("lock.signup.error.password_weak"));
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
      const result = await app.SignIn(username.trim(), password);
      if (result.ok) {
        setLockoutSec(0);
        setNickname(username.trim());
        setRemembered(rememberMe);
        setUnlocked(true);
      } else if (result.lockoutSeconds > 0) {
        setLockoutSec(result.lockoutSeconds);
        setError(
          t("lock.signin.error.locked").replace("{0}", String(result.lockoutSeconds))
        );
        setPassword("");
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
      setHasAccount(false);
      // The remembered flag is tied to the previous account. Clearing
      // it here means the next launch falls back through the Lock
      // setup flow as expected.
      setRemembered(false);
      setUsername("");
      setPassword("");
      setConfirm("");
      setError("");
      setResetOpen(false);
      setTab("signup");
    } catch {
      setResetOpen(false);
      setError(t("lock.signup.error.failed"));
    } finally {
      setBusy(false);
    }
  };

  // Styling helpers — keep the JSX below readable.
  const tabClass = (t: Tab) =>
    `flex-1 h-9 text-xs font-semibold rounded-md transition ${
      tab === t
        ? "bg-violet-500/30 text-white shadow-inner"
        : "text-zinc-400 hover:bg-white/[0.04]"
    } disabled:opacity-50 disabled:cursor-not-allowed`;

  return (
    <div className="h-full flex flex-col">
      {/* Titlebar drag region with the corner UZ/EN toggle. The bottom
          language pill from the previous iteration is gone — this one
          is the only language switcher now. */}
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

      <div className="flex-1 flex items-center justify-center px-6 -mt-4">
        <div className="w-full max-w-[420px]">
          {/* Logo header */}
          <div className="text-center">
            <motion.div
              initial={{ opacity: 0, scale: 0.85 }}
              animate={{ opacity: 1, scale: 1 }}
              transition={{ type: "spring", stiffness: 200, damping: 22 }}
              className="relative inline-block"
            >
              <Logo size={96} />
              <div className="absolute -bottom-0.5 -right-0.5 bg-zinc-900 border border-violet-500/40 rounded-full p-1.5 shadow-lg">
                {tab === "signup" ? (
                  <ShieldCheck className="w-3.5 h-3.5 text-violet-300" />
                ) : (
                  <LockIcon className="w-3.5 h-3.5 text-violet-300" />
                )}
              </div>
            </motion.div>
            <motion.h1
              initial={{ opacity: 0, y: 6 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.15 }}
              className="text-xl font-extrabold tracking-tight mt-3"
            >
              Portal
            </motion.h1>
          </div>

          {/* Auth card */}
          <motion.div
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.2 }}
            className="panel rounded-input mt-5 p-5"
          >
            {/* Tab segmented control. Disabled while a submit is in
                flight to keep the user from switching tabs mid-request. */}
            <div className="flex gap-1 p-1 bg-black/30 rounded-md mb-5">
              <button
                type="button"
                onClick={() => switchTab("signin")}
                disabled={busy || tab === "loading"}
                className={tabClass("signin")}
              >
                {t("lock.signin.title")}
              </button>
              <button
                type="button"
                onClick={() => switchTab("signup")}
                disabled={busy || tab === "loading"}
                className={tabClass("signup")}
              >
                {t("lock.signup.title")}
              </button>
            </div>

            {/* Subtitle that swaps with the tab. Same fixed line height
                across modes so the form below doesn't jump on switch. */}
            <p className="text-xs text-zinc-400 leading-relaxed text-center min-h-[3rem] flex items-center justify-center">
              {tab === "signup" ? t("lock.signup.subtitle") : t("lock.signin.subtitle")}
            </p>

            {/* Sign-Up-on-existing-account warning. Quietly informs
                rather than blocks; the user is the owner of the device
                and they may genuinely want to start over. */}
            {tab === "signup" && hasAccount && (
              <div className="mt-2 rounded-input border border-amber-500/30 bg-amber-500/5 p-2 text-[11px] text-amber-200/90 leading-relaxed">
                {t("lock.signup.replace_warning")}
              </div>
            )}

            {tab === "loading" ? (
              <div className="h-40 flex items-center justify-center text-xs text-zinc-500">
                …
              </div>
            ) : (
              <div className="mt-4 space-y-3">
                <div>
                  <label className="text-[11px] uppercase tracking-widest text-zinc-500 ml-1">
                    {tab === "signup"
                      ? t("lock.signup.username.label")
                      : t("lock.signin.username.label")}
                  </label>
                  <div className="relative mt-1.5">
                    <User className="w-4 h-4 text-zinc-500 absolute left-3 top-1/2 -translate-y-1/2 pointer-events-none" />
                    <input
                      ref={usernameRef}
                      type="text"
                      placeholder={
                        tab === "signup"
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
                  <label className="text-[11px] uppercase tracking-widest text-zinc-500 ml-1">
                    {tab === "signup"
                      ? t("lock.signup.password.label")
                      : t("lock.signin.password.label")}
                  </label>
                  <div className="relative mt-1.5">
                    <KeyRound className="w-4 h-4 text-zinc-500 absolute left-3 top-1/2 -translate-y-1/2 pointer-events-none" />
                    <input
                      ref={passwordRef}
                      type="password"
                      placeholder={
                        tab === "signup"
                          ? t("lock.signup.password.placeholder")
                          : t("lock.signin.password.placeholder")
                      }
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                      onKeyDown={onKey}
                      className="input-base w-full pl-9"
                      maxLength={128}
                      autoComplete={tab === "signup" ? "new-password" : "current-password"}
                      disabled={busy}
                    />
                  </div>
                </div>

                {tab === "signup" && (
                  <motion.div
                    initial={{ opacity: 0, y: -4 }}
                    animate={{ opacity: 1, y: 0 }}
                  >
                    <label className="text-[11px] uppercase tracking-widest text-zinc-500 ml-1">
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

                {/* Live strength checklist — only on Sign-Up. Mirrors the
                    backend rules so what the user sees is what SignUp
                    will accept. Items turn green as they pass; the
                    submit button only enables when all five do. */}
                {tab === "signup" && (
                  <motion.div
                    initial={{ opacity: 0, y: -4 }}
                    animate={{ opacity: 1, y: 0 }}
                    className="rounded-input border border-white/[0.06] bg-black/20 p-2.5 space-y-1"
                  >
                    <div className="text-[10px] uppercase tracking-widest text-zinc-500 mb-0.5">
                      {t("lock.signup.strength.title")}
                    </div>
                    {(
                      [
                        ["length", strength.length],
                        ["lower", strength.lower],
                        ["upper", strength.upper],
                        ["digit", strength.digit],
                        ["special", strength.special],
                      ] as const
                    ).map(([key, ok]) => (
                      <div
                        key={key}
                        className={`flex items-center gap-2 text-[11px] transition ${
                          ok ? "text-emerald-300" : "text-zinc-500"
                        }`}
                      >
                        {ok ? (
                          <Check className="w-3 h-3 shrink-0" strokeWidth={3} />
                        ) : (
                          <XIcon className="w-3 h-3 shrink-0 text-zinc-600" strokeWidth={2.5} />
                        )}
                        <span>{t(`lock.signup.strength.${key}` as any)}</span>
                      </div>
                    ))}
                  </motion.div>
                )}

                {/* Remember-me — kept compact above the submit button so
                    it reads as a property of the action rather than a
                    separate setting. Default ON; users on shared
                    machines untick it for a per-launch lock. */}
                <label className="flex items-start gap-2 cursor-pointer select-none pt-0.5">
                  <input
                    type="checkbox"
                    checked={rememberMe}
                    onChange={(e) => setRememberMe(e.target.checked)}
                    disabled={busy}
                    className="mt-0.5 w-3.5 h-3.5 rounded border-white/20 bg-white/[0.04] text-violet-500 focus:ring-1 focus:ring-violet-500 cursor-pointer accent-violet-500"
                  />
                  <span className="text-[11px] text-zinc-300 leading-snug">
                    {t("lock.remember")}
                    <span className="block text-zinc-500 text-[10px] mt-0.5">
                      {t("lock.remember.hint")}
                    </span>
                  </span>
                </label>

                {/* Error / lockout banner. The lockout text reflects the
                    live countdown so the user can see when they'll be
                    able to try again rather than guessing. */}
                {(error || lockoutSec > 0) && (
                  <div className="rounded-input border border-rose-500/30 bg-rose-500/5 p-2.5 text-xs text-rose-300">
                    {lockoutSec > 0
                      ? t("lock.signin.error.locked").replace("{0}", String(lockoutSec))
                      : error}
                  </div>
                )}

                <button
                  onClick={submit}
                  disabled={busy || lockoutSec > 0}
                  className="btn-primary rounded-btn h-11 w-full text-sm font-semibold flex items-center justify-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {busy ? (
                    tab === "signup" ? (
                      t("lock.signup.submitting")
                    ) : (
                      t("lock.signin.checking")
                    )
                  ) : lockoutSec > 0 ? (
                    `${t("lock.signin.submit")} (${lockoutSec}s)`
                  ) : (
                    <>
                      {tab === "signup" ? t("lock.signup.submit") : t("lock.signin.submit")}
                      <ArrowRight className="w-4 h-4" />
                    </>
                  )}
                </button>

                {/* Bottom row — context-sensitive secondary actions. */}
                <div className="pt-1 text-center">
                  {tab === "signin" ? (
                    <button
                      type="button"
                      onClick={() => setResetOpen(true)}
                      className="text-[11px] text-zinc-500 hover:text-zinc-300 underline"
                    >
                      {t("lock.signin.forgot")}
                    </button>
                  ) : (
                    <p className="text-[11px] text-zinc-500 leading-relaxed">
                      {t("lock.signup.hint")}
                    </p>
                  )}
                </div>
              </div>
            )}
          </motion.div>
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
