// Settings — proper app-style preferences screen.
//
// Layout: left rail with category tabs, right pane with the active
// category's content. Same shape as macOS System Settings, VS Code
// preferences, etc. — familiar enough that no one has to learn a new
// pattern.
//
// What's intentionally NOT here:
//
//   - Cloudflare TURN credentials. Removed: the signaling server hands
//     out short-lived TURN credentials per session in PortalCreated /
//     PortalJoined events, so the user no longer has to provision their
//     own Cloudflare account. mesh.applyServerICE picks them up
//     automatically. The Go-side GetCloudflareTurn / SetCloudflareTurn
//     methods are kept for backwards compatibility but no longer reached
//     from the UI.
//
//   - Manual TURN URL ("o'z serveringiz"). Same reason. A user who
//     genuinely wants to plug their own coturn in can edit the SQLite
//     row by hand — that's an advanced enough task that surfacing it
//     in the dashboard was confusing newcomers more than it was helping
//     anybody.
//
// What IS here, organised by tab:
//
//   - Profile — username, account status, sign-out, reset.
//   - Network — signaling URL, read-only TURN status indicator.
//   - Diagnostics — NAT classification.
//   - Files — receive directory.
//   - Activity — recent peer dials on exposed services.
//   - History — recent portals.
//   - Logs — view/copy/clear in-app logs.
//   - About — version, updates, crash reports, docs.

import React, { useEffect, useState } from "react";
import { motion } from "framer-motion";
import {
  ArrowLeft,
  Activity,
  Check,
  Copy,
  FileText,
  Folder,
  Globe,
  History,
  Info,
  Pencil,
  RefreshCcw,
  Trash2,
  User,
  X,
  Zap,
  LogOut,
  ShieldCheck,
} from "lucide-react";
import { app } from "../lib/wails";
import type { UpdateResult, CrashReport } from "../lib/wails";
import { usePortalStore } from "../stores/portalStore";
import { useT } from "../i18n";
import type { HistoryEntry, ActivityEntry } from "../types";

// One canonical list of tabs. The id is a stable string key both the
// state and the rail use; the icon + label are presentation-only.
type TabId =
  | "profile"
  | "network"
  | "diagnostics"
  | "files"
  | "activity"
  | "history"
  | "logs"
  | "about";

export function Settings() {
  const { t, lang, setLang } = useT();
  const setScreen = usePortalStore((s) => s.setScreen);
  const setUnlocked = usePortalStore((s) => s.setUnlocked);
  const setRemembered = usePortalStore((s) => s.setRemembered);
  const nickname = usePortalStore((s) => s.nickname);
  const setNickname = usePortalStore((s) => s.setNickname);

  const [tab, setTab] = useState<TabId>("profile");

  return (
    <div className="h-full flex flex-col">
      {/* Top bar — back button + title + lang toggle. Stays put while
          the user clicks through tabs. */}
      <div
        className="draggable titlebar-pad flex items-center gap-3 px-4 border-b border-white/5"
        style={{ height: 88 }}
      >
        <button
          onClick={() =>
            setScreen(usePortalStore.getState().portal ? "portal" : "welcome")
          }
          className="no-drag p-2 rounded-md hover:bg-white/5 text-zinc-400"
          title={t("settings.back")}
        >
          <ArrowLeft className="w-4 h-4" />
        </button>
        <h2 className="text-base font-semibold">{t("settings.title")}</h2>

        <div className="no-drag ml-auto flex items-center gap-2">
          <span className="text-[10px] uppercase tracking-widest text-zinc-500">
            {t("settings.language")}
          </span>
          <div className="flex rounded-md border border-white/10 overflow-hidden">
            <button
              onClick={() => setLang("uz")}
              className={`px-2 py-1 text-xs ${
                lang === "uz"
                  ? "bg-violet-500/20 text-violet-200"
                  : "text-zinc-400 hover:text-white hover:bg-white/5"
              }`}
            >
              UZ
            </button>
            <button
              onClick={() => setLang("en")}
              className={`px-2 py-1 text-xs border-l border-white/10 ${
                lang === "en"
                  ? "bg-violet-500/20 text-violet-200"
                  : "text-zinc-400 hover:text-white hover:bg-white/5"
              }`}
            >
              EN
            </button>
          </div>
        </div>
      </div>

      {/* Two-column shell — sidebar (≤200px) + content. The content
          column owns its own scroll so the rail stays anchored on
          long-running sections like Logs or History. */}
      <div className="flex-1 flex overflow-hidden">
        <aside className="w-56 shrink-0 border-r border-white/5 py-4 overflow-y-auto">
          <SidebarTab id="profile" current={tab} onSelect={setTab} icon={<User className="w-4 h-4" />} label={t("settings.tab.profile")} />
          <SidebarTab id="network" current={tab} onSelect={setTab} icon={<Globe className="w-4 h-4" />} label={t("settings.tab.network")} />
          <SidebarTab id="diagnostics" current={tab} onSelect={setTab} icon={<Info className="w-4 h-4" />} label={t("settings.tab.diagnostics")} />
          <SidebarTab id="files" current={tab} onSelect={setTab} icon={<Folder className="w-4 h-4" />} label={t("settings.tab.files")} />
          <SidebarTab id="activity" current={tab} onSelect={setTab} icon={<Activity className="w-4 h-4" />} label={t("settings.tab.activity")} />
          <SidebarTab id="history" current={tab} onSelect={setTab} icon={<History className="w-4 h-4" />} label={t("settings.tab.history")} />
          <SidebarTab id="logs" current={tab} onSelect={setTab} icon={<FileText className="w-4 h-4" />} label={t("settings.tab.logs")} />
          <SidebarTab id="about" current={tab} onSelect={setTab} icon={<Info className="w-4 h-4" />} label={t("settings.tab.about")} />
        </aside>

        <div className="flex-1 overflow-y-auto">
          <div className="max-w-2xl mx-auto p-6">
            {tab === "profile" && (
              <ProfileTab
                nickname={nickname}
                setNickname={setNickname}
                onSignOut={async () => {
                  try {
                    await app.Leave();
                  } catch {}
                  setRemembered(false);
                  setUnlocked(false);
                }}
                onReset={async () => {
                  try {
                    await app.Leave();
                  } catch {}
                  await app.ResetVault();
                  setRemembered(false);
                  setUnlocked(false);
                }}
              />
            )}
            {tab === "network" && <NetworkTab />}
            {tab === "diagnostics" && <DiagnosticsTab />}
            {tab === "files" && <FilesTab />}
            {tab === "activity" && <ActivityTab />}
            {tab === "history" && <HistoryTab />}
            {tab === "logs" && <LogsTab />}
            {tab === "about" && <AboutTab />}
          </div>
        </div>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Sidebar
// ---------------------------------------------------------------------------

function SidebarTab({
  id,
  current,
  onSelect,
  icon,
  label,
}: {
  id: TabId;
  current: TabId;
  onSelect: (id: TabId) => void;
  icon: React.ReactNode;
  label: string;
}) {
  const active = id === current;
  return (
    <button
      onClick={() => onSelect(id)}
      className={`w-full text-left px-4 py-2 text-sm flex items-center gap-2.5 transition relative ${
        active
          ? "bg-violet-500/10 text-violet-200"
          : "text-zinc-400 hover:text-zinc-200 hover:bg-white/[0.03]"
      }`}
    >
      {active && (
        <span className="absolute left-0 top-1.5 bottom-1.5 w-[2px] rounded-full bg-violet-400" />
      )}
      <span className={active ? "text-violet-300" : "text-zinc-500"}>{icon}</span>
      {label}
    </button>
  );
}

// ---------------------------------------------------------------------------
// Profile
// ---------------------------------------------------------------------------

function ProfileTab({
  nickname,
  setNickname,
  onSignOut,
  onReset,
}: {
  nickname: string;
  setNickname: (n: string) => void;
  onSignOut: () => void;
  onReset: () => void;
}) {
  const { t } = useT();
  const [confirmReset, setConfirmReset] = useState(false);
  const [accountUser, setAccountUser] = useState("");
  // Device-label state — pulled from the Go side which falls back to
  // a platform default ('mac' / 'win 64' / 'linux') when nothing is
  // saved yet. Editing this changes how the device is announced in
  // the room: 'texuz · uy' instead of plain 'texuz'.
  const [deviceName, setDeviceName] = useState("");
  const [deviceDraft, setDeviceDraft] = useState("");
  const [deviceSavedAt, setDeviceSavedAt] = useState<number | null>(null);
  // Auto-run state — checkbox-style toggle. The Go side writes the
  // OS-level hook (LaunchAgent / Run / .desktop) and persists the
  // flag; we read it on mount so the toggle reflects current truth.
  const [autoRun, setAutoRun] = useState(false);
  const [autoRunErr, setAutoRunErr] = useState("");

  // Pull the actual stored username on mount. nickname in the store
  // tracks the *display* nickname (which defaults to the username but
  // can drift if the user edits it inside a portal); the real auth
  // identity lives in storage and we want to surface that on profile.
  useEffect(() => {
    app.CurrentUsername().then((u) => {
      if (u) {
        setAccountUser(u);
        if (!nickname.trim()) setNickname(u);
      }
    });
    app.CurrentDeviceName().then((d) => {
      setDeviceName(d);
      setDeviceDraft(d);
    });
    app.IsAutoRun().then(setAutoRun);
  }, [nickname, setNickname]);

  const handle = accountUser || nickname || "—";
  const initial = (handle.trim()[0] || "?").toUpperCase();

  const saveDeviceName = async () => {
    const next = deviceDraft.trim();
    try {
      await app.SetDeviceName(next);
      const fresh = await app.CurrentDeviceName();
      setDeviceName(fresh);
      setDeviceDraft(fresh);
      setDeviceSavedAt(Date.now());
    } catch {}
  };

  const toggleAutoRun = async () => {
    setAutoRunErr("");
    const next = !autoRun;
    try {
      await app.SetAutoRun(next);
      setAutoRun(next);
    } catch (e: any) {
      setAutoRunErr(e?.message || String(e));
    }
  };

  return (
    <Section
      icon={<User className="w-4 h-4" />}
      title={t("settings.profile.title")}
    >
      {/* Identity card — large avatar + handle + status badge. Sets the
          tone that this is a proper account screen, not an afterthought. */}
      <div className="flex items-center gap-4">
        <div className="w-16 h-16 rounded-full bg-gradient-to-br from-violet-500/40 to-cyan-500/30 flex items-center justify-center text-2xl font-extrabold text-white border border-white/10">
          {initial}
        </div>
        <div className="min-w-0">
          <div className="text-lg font-semibold truncate">{handle}</div>
          <div className="mt-1 inline-flex items-center gap-1.5 text-[11px] uppercase tracking-wider text-emerald-300 bg-emerald-500/10 border border-emerald-500/20 rounded px-2 py-0.5">
            <ShieldCheck className="w-3 h-3" />
            {t("settings.profile.account_local")}
          </div>
        </div>
      </div>

      <p className="text-xs text-zinc-500 leading-relaxed">
        {t("settings.profile.account_local_hint")}
      </p>

      {/* Username row — read-only here. Mutation requires a re-sign-up
          flow which lives behind ResetVault, since changing the username
          would orphan the bcrypt hash. Showing it as an inert input
          rather than a chip makes the field feel familiar without
          implying it's editable. */}
      <Field label={t("settings.profile.username")}>
        <input
          type="text"
          value={handle}
          readOnly
          className="input-base w-full font-mono text-sm bg-white/[0.02] cursor-default"
        />
      </Field>

      {/* Device label — multi-device disambiguation. Default is
          platform-derived ('mac' / 'win 64' / 'linux'); rename to
          'uy' / 'ish' / 'serverim' for distinct rooms. The mesh
          nickname becomes 'username · device' so peers see which
          install they're talking to. Editable inline; Save persists
          and refreshes the displayed value. */}
      <Field label={t("settings.profile.device_name")}>
        <div className="flex gap-2 items-stretch">
          <input
            type="text"
            value={deviceDraft}
            onChange={(e) => setDeviceDraft(e.target.value)}
            placeholder={t("settings.profile.device_name_placeholder")}
            maxLength={32}
            className="input-base flex-1 text-sm"
          />
          <button
            onClick={saveDeviceName}
            disabled={deviceDraft.trim() === deviceName.trim()}
            className="btn-primary rounded-btn px-4 text-sm font-medium disabled:opacity-50"
          >
            {t("settings.network.save")}
          </button>
        </div>
        {deviceSavedAt && (
          <div className="text-xs text-emerald-400 mt-1.5">
            {t("settings.network.saved")}
          </div>
        )}
        <p className="text-[11px] text-zinc-500 mt-2 leading-relaxed">
          {t("settings.profile.device_name_hint")}{" "}
          <span className="font-mono text-zinc-400">
            {handle} · {deviceName || "—"}
          </span>
        </p>
      </Field>

      {/* Auto-run on system startup — checkbox toggle. Wails-side
          writes the platform-specific entry (LaunchAgent on macOS,
          HKCU\…\Run on Windows, ~/.config/autostart on Linux);
          flipping the toggle off removes the entry. */}
      <Field label={t("settings.profile.autorun")}>
        <button
          onClick={toggleAutoRun}
          className={`w-full rounded-input px-3 py-2.5 flex items-center gap-3 transition border ${
            autoRun
              ? "bg-violet-500/[0.08] border-violet-500/30"
              : "bg-white/[0.02] border-white/[0.06] hover:bg-white/[0.04]"
          }`}
        >
          <span
            className={`w-9 h-5 rounded-full relative transition ${
              autoRun ? "bg-violet-500" : "bg-zinc-700"
            }`}
          >
            <span
              className={`absolute top-0.5 w-4 h-4 rounded-full bg-white transition-all ${
                autoRun ? "left-4" : "left-0.5"
              }`}
            />
          </span>
          <span className="text-sm">
            {autoRun
              ? t("settings.profile.autorun_on")
              : t("settings.profile.autorun_off")}
          </span>
        </button>
        <p className="text-[11px] text-zinc-500 mt-2 leading-relaxed">
          {t("settings.profile.autorun_hint")}
        </p>
        {autoRunErr && (
          <div className="text-xs text-rose-400 mt-1.5">{autoRunErr}</div>
        )}
      </Field>

      <div className="pt-2 border-t border-white/5 space-y-2">
        <button
          onClick={onSignOut}
          className="w-full panel rounded-btn px-4 py-2.5 text-sm flex items-center justify-center gap-2 hover:bg-white/[0.07]"
        >
          <LogOut className="w-4 h-4" />
          {t("settings.profile.signout")}
        </button>

        {!confirmReset ? (
          <button
            onClick={() => setConfirmReset(true)}
            className="w-full rounded-btn px-4 py-2.5 text-sm flex items-center justify-center gap-2 text-rose-300 hover:text-rose-200 hover:bg-rose-500/10 border border-rose-500/20"
          >
            <Trash2 className="w-4 h-4" />
            {t("settings.profile.reset")}
          </button>
        ) : (
          <div className="rounded-input border border-rose-500/30 bg-rose-500/5 p-3 space-y-2">
            <p className="text-xs text-rose-200/90 leading-relaxed">
              {t("settings.profile.reset_warn")}
            </p>
            <div className="flex gap-2">
              <button
                onClick={() => setConfirmReset(false)}
                className="flex-1 panel rounded-btn px-3 py-2 text-xs hover:bg-white/[0.07]"
              >
                {t("settings.profile.reset_cancel")}
              </button>
              <button
                onClick={onReset}
                className="flex-1 rounded-btn px-3 py-2 text-xs font-semibold bg-rose-500/30 hover:bg-rose-500/50 text-rose-100"
              >
                {t("settings.profile.reset_confirm")}
              </button>
            </div>
          </div>
        )}
      </div>
    </Section>
  );
}

// ---------------------------------------------------------------------------
// Network
// ---------------------------------------------------------------------------

function NetworkTab() {
  const { t } = useT();
  const signalingUrl = usePortalStore((s) => s.signalingUrl);
  const setSignalingUrl = usePortalStore((s) => s.setSignalingUrl);

  const [draftUrl, setDraftUrl] = useState(signalingUrl);
  const [savedAt, setSavedAt] = useState<number | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    setDraftUrl(signalingUrl);
  }, [signalingUrl]);

  const saveUrl = async () => {
    setError("");
    try {
      await app.SetSignalingURL(draftUrl);
      setSignalingUrl(draftUrl);
      setSavedAt(Date.now());
    } catch (e: any) {
      setError(e?.message || String(e));
    }
  };

  // The TURN status badge cares about whether direct connection is
  // viable. We treat type=3 (symmetric NAT) as "active — relay path is
  // the live one"; everything else as "idle but ready" because pion
  // will request relay candidates anyway, just rarely uses them.
  const nat = usePortalStore((s) => s.nat);
  const turnActive = nat?.type === 3 || nat?.needsTurn;

  return (
    <div className="space-y-6">
      <Section icon={<Globe className="w-4 h-4" />} title={t("settings.network.title")}>
        <Field label={t("settings.network.signaling_label")}>
          <div className="flex gap-2">
            <input
              type="text"
              value={draftUrl}
              onChange={(e) => setDraftUrl(e.target.value)}
              className="input-base flex-1 font-mono text-sm"
            />
            <button
              onClick={saveUrl}
              className="btn-primary rounded-btn px-4 text-sm font-medium"
            >
              {t("settings.network.save")}
            </button>
          </div>
          {error && <div className="text-xs text-rose-400 mt-1.5">{error}</div>}
          {savedAt && !error && (
            <div className="text-xs text-emerald-400 mt-1.5">
              {t("settings.network.saved")}
            </div>
          )}
          <p className="text-xs text-zinc-500 mt-2">
            {t("settings.network.signaling_hint")}{" "}
            <span className="font-mono">wss://signaling.1pro.uz/ws</span>
          </p>
        </Field>
      </Section>

      {/* TURN — read-only status panel. Replaces the previous
          Cloudflare/manual TURN forms. The signaling server provisions
          credentials per session via PortalCreated / PortalJoined; the
          desktop app applies them through mesh.applyServerICE. Nothing
          for the user to touch here. */}
      <Section icon={<Zap className="w-4 h-4" />} title={t("settings.network.turn_title")}>
        <div className="flex items-start gap-3">
          <div className="w-9 h-9 rounded-md bg-violet-500/20 border border-violet-500/30 flex items-center justify-center shrink-0">
            <ShieldCheck className="w-4 h-4 text-violet-300" />
          </div>
          <div className="flex-1 min-w-0">
            <div className="text-sm font-medium">
              {t("settings.network.turn_managed")}
            </div>
            <p className="text-xs text-zinc-500 mt-1.5 leading-relaxed">
              {t("settings.network.turn_managed_hint")}
            </p>
          </div>
        </div>

        <Field label={t("settings.network.turn_status_label")}>
          <div
            className={`panel rounded-input px-3 py-2 text-xs flex items-center gap-2 ${
              turnActive
                ? "border-emerald-500/30 bg-emerald-500/5"
                : ""
            }`}
          >
            <span
              className={`w-2 h-2 rounded-full ${
                turnActive ? "bg-emerald-400" : "bg-zinc-500"
              }`}
            />
            <span className={turnActive ? "text-emerald-300" : "text-zinc-300"}>
              {turnActive
                ? t("settings.network.turn_status_active")
                : t("settings.network.turn_status_idle")}
            </span>
          </div>
        </Field>
      </Section>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Diagnostics
// ---------------------------------------------------------------------------

function DiagnosticsTab() {
  const { t } = useT();
  const nat = usePortalStore((s) => s.nat);

  const refresh = async () => {
    const r = await app.NATInfo();
    usePortalStore.getState().setNat(r);
  };

  return (
    <Section icon={<Info className="w-4 h-4" />} title={t("settings.diag.title")}>
      <Field label={t("settings.diag.nat_label")}>
        {nat ? (
          <div className="space-y-1.5">
            <div
              className={`text-base font-semibold ${
                nat.type === 1
                  ? "text-emerald-400"
                  : nat.type === 2
                  ? "text-amber-400"
                  : nat.type === 3
                  ? "text-rose-400"
                  : "text-zinc-300"
              }`}
            >
              {nat.label}
            </div>
            <div className="text-xs text-zinc-500 font-mono space-y-0.5">
              <div>
                {t("settings.diag.local")}: {nat.localAddr}
              </div>
              {nat.reflexiveAddrs.map((a, i) => (
                <div key={i}>
                  {t("settings.diag.public")}: {a}
                </div>
              ))}
            </div>
          </div>
        ) : (
          <div className="text-sm text-zinc-500">{t("settings.diag.detecting")}</div>
        )}
      </Field>
      <button
        onClick={refresh}
        className="panel rounded-btn px-3 py-2 text-xs flex items-center gap-1.5 hover:bg-white/[0.07]"
      >
        <RefreshCcw className="w-3.5 h-3.5" />
        {t("settings.diag.refresh")}
      </button>
    </Section>
  );
}

// ---------------------------------------------------------------------------
// Files
// ---------------------------------------------------------------------------

function FilesTab() {
  const { t } = useT();
  const saveDir = usePortalStore((s) => s.saveDir);
  const setSaveDir = usePortalStore((s) => s.setSaveDir);

  useEffect(() => {
    app.SaveDir().then(setSaveDir);
  }, [setSaveDir]);

  return (
    <Section icon={<Folder className="w-4 h-4" />} title={t("settings.files.title")}>
      <Field label={t("settings.files.save_dir_label")}>
        <div className="flex gap-2 items-center">
          <code className="font-mono text-sm flex-1 panel rounded-input px-3 py-2 truncate">
            {saveDir || "—"}
          </code>
          <button
            onClick={() => app.OpenSaveDir()}
            className="panel rounded-btn px-3 py-2 text-sm hover:bg-white/[0.07]"
          >
            {t("settings.files.open")}
          </button>
        </div>
      </Field>
    </Section>
  );
}

// ---------------------------------------------------------------------------
// Activity (was inline in old Settings.tsx)
// ---------------------------------------------------------------------------

function ActivityTab() {
  const { t } = useT();
  const [entries, setEntries] = useState<ActivityEntry[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    const refresh = async () => {
      try {
        const list = await app.ProxyActivity();
        if (!cancelled) setEntries(list);
      } finally {
        if (!cancelled) setLoading(false);
      }
    };
    refresh();
    const tt = window.setInterval(refresh, 5000);
    return () => {
      cancelled = true;
      window.clearInterval(tt);
    };
  }, []);

  return (
    <Section icon={<Activity className="w-4 h-4" />} title={t("settings.activity.title")}>
      {loading ? (
        <div className="text-xs text-zinc-500">{t("settings.activity.loading")}</div>
      ) : entries.length === 0 ? (
        <div className="text-xs text-zinc-500">{t("settings.activity.empty")}</div>
      ) : (
        <div className="space-y-1.5 max-h-[360px] overflow-y-auto">
          {entries
            .slice()
            .reverse()
            .map((e, i) => {
              const ok = e.result === "ok";
              const tone = ok ? "text-emerald-300/90" : "text-rose-300/90";
              const dot = ok ? "bg-emerald-400" : "bg-rose-400";
              const when = (() => {
                try {
                  return new Date(e.time).toLocaleTimeString();
                } catch {
                  return e.time;
                }
              })();
              return (
                <div
                  key={`${e.time}-${i}`}
                  className="panel rounded-input px-3 py-2 flex items-start gap-2 text-xs"
                >
                  <span className={`w-2 h-2 rounded-full mt-1 shrink-0 ${dot}`} />
                  <div className="min-w-0 flex-1 leading-snug">
                    <div className="flex items-center gap-2">
                      <span className="font-medium truncate">
                        {e.nickname || e.peerId.slice(0, 8)}
                      </span>
                      <span className="text-zinc-500">→</span>
                      <span className="font-mono text-[10px] uppercase text-zinc-400 shrink-0">
                        {e.protocol}:{e.port}
                      </span>
                      {e.target && e.target !== `127.0.0.1:${e.port}` && (
                        <span className="font-mono text-[10px] text-zinc-500 truncate">
                          → {e.target}
                        </span>
                      )}
                    </div>
                    <div className={`text-[10px] ${tone}`}>{e.result}</div>
                  </div>
                  <span className="font-mono text-[10px] text-zinc-500 shrink-0">{when}</span>
                </div>
              );
            })}
        </div>
      )}
    </Section>
  );
}

// ---------------------------------------------------------------------------
// History
// ---------------------------------------------------------------------------

function HistoryTab() {
  const { t } = useT();
  const history = usePortalStore((s) => s.history);
  const setHistory = usePortalStore((s) => s.setHistory);

  const refresh = async () => setHistory(await app.RecentPortals(20));

  useEffect(() => {
    refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <Section icon={<History className="w-4 h-4" />} title={t("settings.history.title")}>
      {history.length === 0 ? (
        <div className="text-sm text-zinc-500 py-4 text-center">
          {t("settings.history.empty")}
        </div>
      ) : (
        <div className="space-y-1.5">
          {history.map((h) => (
            <HistoryRow key={h.id} h={h} onChanged={refresh} />
          ))}
        </div>
      )}
      {history.length > 0 && (
        <button
          onClick={async () => {
            await app.ClearHistory();
            setHistory([]);
          }}
          className="mt-3 text-xs text-rose-400 hover:text-rose-300 flex items-center gap-1"
        >
          <Trash2 className="w-3 h-3" />
          {t("settings.history.clear")}
        </button>
      )}
    </Section>
  );
}

// HistoryRow renders one portal_history row, with an inline rename
// affordance that mirrors the Welcome dashboard's. Settings is the
// "settle in" surface for editing labels, so the pencil is visible
// at all times here (not hover-only).
function HistoryRow({
  h,
  onChanged,
}: {
  h: HistoryEntry;
  onChanged: () => void | Promise<void>;
}) {
  const { t } = useT();
  const dt = new Date(h.lastSeen).toLocaleString();
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(h.label || "");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!editing) setDraft(h.label || "");
  }, [h.label, editing]);

  const submit = async () => {
    setSaving(true);
    try {
      await app.RenamePortal(h.id, draft.trim());
      await onChanged();
      setEditing(false);
    } finally {
      setSaving(false);
    }
  };

  if (editing) {
    return (
      <div className="panel rounded-input px-3 py-2 flex items-center gap-2 text-sm">
        <span className="font-mono text-[10px] text-zinc-500 shrink-0">#{h.portalId}</span>
        <input
          type="text"
          autoFocus
          value={draft}
          onChange={(e) => setDraft(e.target.value.slice(0, 60))}
          onKeyDown={(e) => {
            if (e.key === "Enter") submit();
            if (e.key === "Escape") setEditing(false);
          }}
          placeholder={t("welcome.rename.placeholder")}
          className="input-base flex-1 text-sm py-1"
          maxLength={60}
          disabled={saving}
        />
        <button
          onClick={submit}
          disabled={saving}
          className="p-1 text-emerald-300 hover:bg-emerald-500/15 rounded"
          title={t("welcome.rename.save")}
        >
          <Check className="w-4 h-4" />
        </button>
        <button
          onClick={() => setEditing(false)}
          disabled={saving}
          className="p-1 text-zinc-400 hover:bg-white/[0.07] rounded"
          title={t("welcome.rename.cancel")}
        >
          <X className="w-4 h-4" />
        </button>
      </div>
    );
  }

  const label = h.label?.trim();
  return (
    <div className="panel rounded-input px-3 py-2 flex items-center justify-between text-sm group">
      <div className="flex items-center gap-3 min-w-0 flex-1">
        {label ? (
          <>
            <span className="font-medium truncate">{label}</span>
            <span className="font-mono text-[10px] text-zinc-500 shrink-0">
              #{h.portalId}
            </span>
          </>
        ) : (
          <span className="font-mono text-violet-300">{h.portalId}</span>
        )}
        <span className="text-zinc-500">·</span>
        <span className="truncate">{h.nickname}</span>
        {h.isOwner && (
          <span className="text-[10px] uppercase tracking-wider text-amber-400">
            {t("common.owner")}
          </span>
        )}
      </div>
      <div className="flex items-center gap-1 shrink-0 ml-3">
        <span className="text-xs text-zinc-500">{dt}</span>
        <button
          onClick={() => setEditing(true)}
          className="p-1 text-zinc-500 hover:text-violet-300 hover:bg-white/[0.07] rounded opacity-0 group-hover:opacity-100 transition"
          title={t("welcome.rename.title")}
        >
          <Pencil className="w-3.5 h-3.5" />
        </button>
        <button
          onClick={async () => {
            await app.RemoveRecentPortal(h.id);
            await onChanged();
          }}
          className="p-1 text-zinc-500 hover:text-rose-300 hover:bg-rose-500/10 rounded opacity-0 group-hover:opacity-100 transition"
          title={t("welcome.recent.remove")}
        >
          <X className="w-3.5 h-3.5" />
        </button>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Logs
// ---------------------------------------------------------------------------

function LogsTab() {
  const { t } = useT();
  const [logs, setLogs] = useState<string[]>([]);
  const [logPath, setLogPath] = useState("");
  const [logsOpen, setLogsOpen] = useState(false);

  const showLogs = async () => {
    const [lines, path] = await Promise.all([app.LogLines(200), app.LogFilePath()]);
    setLogs(lines);
    setLogPath(path);
    setLogsOpen(true);
  };
  const copyLogs = async () => {
    try {
      await navigator.clipboard.writeText(logs.join("\n"));
    } catch {}
  };
  const clearLogs = async () => {
    await app.ClearLogs();
    setLogs([]);
  };

  return (
    <>
      <Section icon={<FileText className="w-4 h-4" />} title={t("settings.logs.title")}>
        <p className="text-xs text-zinc-500">{t("settings.logs.hint")}</p>
        <div className="flex gap-2 flex-wrap">
          <button
            onClick={showLogs}
            className="btn-primary rounded-btn px-4 py-2 text-sm font-medium"
          >
            {t("settings.logs.show")}
          </button>
          <button
            onClick={() => app.OpenLogFolder()}
            className="panel rounded-btn px-3 py-2 text-xs flex items-center gap-1.5 hover:bg-white/[0.07]"
          >
            <Folder className="w-3.5 h-3.5" />
            {t("settings.logs.open_folder")}
          </button>
        </div>
      </Section>

      {logsOpen && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="fixed inset-0 z-[60] bg-black/70 backdrop-blur-md flex items-center justify-center p-6 titlebar-pad"
          onClick={() => setLogsOpen(false)}
        >
          <motion.div
            initial={{ scale: 0.95, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            transition={{ type: "spring", stiffness: 320, damping: 26 }}
            className="bg-[#0d1322] panel rounded-card flex flex-col w-full max-w-3xl max-h-[80vh] overflow-hidden"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="px-5 py-3 border-b border-white/5 flex items-center justify-between">
              <div className="flex items-center gap-2 min-w-0">
                <FileText className="w-4 h-4 text-violet-300 shrink-0" />
                <span className="font-mono text-xs text-zinc-400 truncate">{logPath}</span>
              </div>
              <div className="flex items-center gap-2 shrink-0">
                <button
                  onClick={copyLogs}
                  className="text-xs text-zinc-400 hover:text-white px-2 py-1 rounded hover:bg-white/[0.05] flex items-center gap-1"
                >
                  <Copy className="w-3 h-3" /> {t("settings.logs.copy")}
                </button>
                <button
                  onClick={clearLogs}
                  className="text-xs text-zinc-400 hover:text-rose-300 px-2 py-1 rounded hover:bg-rose-500/10"
                >
                  {t("settings.logs.clear")}
                </button>
                <button
                  onClick={() => setLogsOpen(false)}
                  className="text-xs text-zinc-400 hover:text-white px-2 py-1 rounded hover:bg-white/[0.05]"
                >
                  {t("settings.logs.close")}
                </button>
              </div>
            </div>
            <pre className="flex-1 overflow-auto px-5 py-4 font-mono text-[11px] text-zinc-300 leading-relaxed whitespace-pre-wrap">
              {logs.length === 0 ? t("settings.logs.empty") : logs.join("\n")}
            </pre>
          </motion.div>
        </motion.div>
      )}
    </>
  );
}

// ---------------------------------------------------------------------------
// About
// ---------------------------------------------------------------------------

function AboutTab() {
  const { t } = useT();
  const [version, setVersion] = useState("");
  const [update, setUpdate] = useState<UpdateResult | null>(null);
  const [updateChecking, setUpdateChecking] = useState(false);
  const [crashes, setCrashes] = useState<CrashReport[]>([]);

  useEffect(() => {
    app.AppVersion().then(setVersion);
    app.CheckForUpdate(false).then(setUpdate).catch(() => {});
    app.CrashReports().then(setCrashes).catch(() => {});
  }, []);

  const recheckUpdate = async () => {
    setUpdateChecking(true);
    try {
      setUpdate(await app.CheckForUpdate(true));
    } finally {
      setUpdateChecking(false);
    }
  };

  const refreshCrashes = async () => {
    try {
      setCrashes(await app.CrashReports());
    } catch {}
  };

  const clearCrashes = async () => {
    await app.ClearCrashReports();
    setCrashes([]);
  };

  return (
    <Section icon={<Info className="w-4 h-4" />} title={t("settings.about.title")}>
      <Field label={t("settings.about.version")}>
        <div className="flex items-center gap-3">
          <code className="font-mono text-base panel rounded-input px-3 py-2">
            v{version || "—"}
          </code>
          <button
            onClick={recheckUpdate}
            disabled={updateChecking}
            className="panel rounded-btn px-3 py-2 text-xs flex items-center gap-1.5 hover:bg-white/[0.07] disabled:opacity-50"
          >
            <RefreshCcw className={`w-3.5 h-3.5 ${updateChecking ? "animate-spin" : ""}`} />
            {updateChecking ? t("settings.about.checking") : t("settings.about.check_now")}
          </button>
        </div>
        {update && (
          <div className="mt-3 text-xs">
            {update.error ? (
              <div className="text-rose-400">{update.error}</div>
            ) : update.available ? (
              <div className="panel rounded-input p-3 border-violet-500/30 bg-violet-500/5 space-y-2">
                <div>
                  <span className="text-violet-200">
                    {t("settings.about.new_available")}
                  </span>{" "}
                  <span className="font-mono font-semibold">v{update.latestVersion}</span>
                </div>
                {update.releaseNotes && (
                  <pre className="text-[11px] text-zinc-400 whitespace-pre-wrap font-mono leading-relaxed max-h-32 overflow-y-auto">
                    {update.releaseNotes}
                  </pre>
                )}
                <button
                  onClick={() => app.OpenReleasePage(update.releaseUrl)}
                  className="btn-primary rounded-btn px-3 py-1.5 text-xs font-semibold"
                >
                  {t("settings.about.open_download")}
                </button>
              </div>
            ) : (
              <div className="text-emerald-400">{t("settings.about.up_to_date")}</div>
            )}
          </div>
        )}
      </Field>

      <Field label={t("settings.about.crashes_label")}>
        <p className="text-xs text-zinc-500 -mt-1 mb-2">
          {t("settings.about.crashes_hint")}
        </p>
        {crashes.length === 0 ? (
          <div className="text-sm text-zinc-500 panel rounded-input px-3 py-2">
            {t("settings.about.crashes_empty")}
          </div>
        ) : (
          <div className="space-y-1.5">
            {crashes.slice(0, 5).map((c) => (
              <div
                key={c.id}
                className="panel rounded-input px-3 py-2 text-xs flex items-center gap-3"
              >
                <span className="font-mono text-rose-300 shrink-0">{c.id}</span>
                <span className="flex-1 truncate text-zinc-400">{c.panicMessage}</span>
                <span className="text-zinc-600 shrink-0">
                  {c.os}/{c.arch}
                </span>
              </div>
            ))}
            {crashes.length > 5 && (
              <div className="text-xs text-zinc-500">
                {t("settings.about.crashes_more").replace("{0}", String(crashes.length - 5))}
              </div>
            )}
          </div>
        )}
        <div className="flex gap-2 mt-2 flex-wrap">
          <button
            onClick={() => app.OpenCrashFolder()}
            className="panel rounded-btn px-3 py-1.5 text-xs flex items-center gap-1.5 hover:bg-white/[0.07]"
          >
            <Folder className="w-3.5 h-3.5" />
            {t("settings.files.open")}
          </button>
          <button
            onClick={refreshCrashes}
            className="panel rounded-btn px-3 py-1.5 text-xs flex items-center gap-1.5 hover:bg-white/[0.07]"
          >
            <RefreshCcw className="w-3.5 h-3.5" />
            {t("settings.about.refresh")}
          </button>
          {crashes.length > 0 && (
            <button
              onClick={clearCrashes}
              className="panel rounded-btn px-3 py-1.5 text-xs flex items-center gap-1.5 hover:bg-white/[0.07] text-rose-300"
            >
              <Trash2 className="w-3.5 h-3.5" />
              {t("settings.about.clear_all")}
            </button>
          )}
        </div>
      </Field>

      <Field label={t("settings.about.docs")}>
        <div className="flex gap-2 flex-wrap text-xs">
          {[
            ["GitHub", "https://github.com/Yaxyobek0877/portal_traffic"],
            [
              "PCP-1 Spec",
              "https://github.com/Yaxyobek0877/portal_traffic/blob/main/client/crypt/pcp/SPEC.md",
            ],
            ["Privacy", "https://github.com/Yaxyobek0877/portal_traffic/blob/main/PRIVACY.md"],
            ["Terms", "https://github.com/Yaxyobek0877/portal_traffic/blob/main/TERMS.md"],
            ["License (MIT)", "https://github.com/Yaxyobek0877/portal_traffic/blob/main/LICENSE"],
          ].map(([label, url]) => (
            <button
              key={url}
              onClick={() => app.OpenReleasePage(url)}
              className="panel rounded-input px-3 py-1.5 hover:bg-white/[0.07]"
            >
              {label}
            </button>
          ))}
        </div>
      </Field>
    </Section>
  );
}

// ---------------------------------------------------------------------------
// Shared bits
// ---------------------------------------------------------------------------

function Section({
  title,
  icon,
  children,
}: {
  title: string;
  icon: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      className="panel rounded-card p-5"
    >
      <h3 className="text-sm font-semibold flex items-center gap-2 mb-4">
        <span className="text-violet-300">{icon}</span>
        {title}
      </h3>
      <div className="space-y-4">{children}</div>
    </motion.div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <label className="text-xs uppercase tracking-widest text-zinc-500 mb-1.5 block">
        {label}
      </label>
      {children}
    </div>
  );
}
