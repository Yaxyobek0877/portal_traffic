import type { Dict } from "./strings";

export const en: Dict = {
  // Welcome
  "welcome.greeting": "Welcome back",
  "welcome.subtitle": "Open a new portal or join your friend's.",
  "welcome.tagline": "Direct connections.\nNo servers in between.",
  "welcome.nickname.label": "Nickname",
  "welcome.nickname.placeholder": "alice",
  "welcome.portalId.label": "Portal ID",
  "welcome.portalId.placeholder": "123456",
  "welcome.code.label": "Code",
  "welcome.code.placeholder": "654321",
  "welcome.create": "Create portal",
  "welcome.create.card_title": "New portal",
  "welcome.create.card_hint": "One click — share the ID + code with your friend.",
  "welcome.join": "Join",
  "welcome.join.card_title": "Join",
  "welcome.join.card_hint": "Enter the portal ID and 6-digit code you were given.",
  "welcome.creating": "Create",
  "welcome.joining": "Join",
  "welcome.cancel": "Cancel",
  "welcome.back": "Back",
  "welcome.recent": "Recent portals",
  "welcome.recent.empty": "No portals yet — yours will show up here.",
  "welcome.recent.remove": "Remove from history",
  "welcome.recent.background": "Connect in background (don't enter)",
  "welcome.rename.title": "Rename",
  "welcome.rename.placeholder": "e.g. Family portal",
  "welcome.rename.save": "Save",
  "welcome.rename.cancel": "Cancel",
  "welcome.signout": "Sign out",
  "welcome.active": "Active connections",
  "welcome.active.foreground": "showing",
  "welcome.active.peers": "peers",
  "welcome.active.leave": "Leave this portal",
  "welcome.create.background": "Create in background (don't enter)",
  "welcome.join.background": "Connect in background — don't enter",
  "welcome.join.background_short": "Background",
  "welcome.error.empty_nickname": "Please enter a nickname first",
  "welcome.error.empty_id_or_code": "Both ID and code are required",
  "welcome.error.no_such_portal":
    "This portal has been closed. The owner may have left — your saved ID + code is no longer valid.",
  "welcome.error.wrong_code": "Wrong code. Double-check it or ask the owner.",
  "welcome.error.full": "Portal is full (16-peer max).",
  "welcome.error.locked": "Owner has locked this portal. Wait for it to open.",
  "welcome.error.nickname_taken": "That nickname is already in use in this portal. Pick another.",
  "welcome.create_new_link": "Create a new portal →",
  "welcome.symmetric_nat_warning":
    "Symmetric NAT detected. Direct connections may not work; the app will fall back to a free TURN relay automatically — nothing to configure. Latency may be slightly higher.",

  // Banner messages
  "banner.portal_closed": "Portal closed.",
  "banner.signaling_disconnected": "Signaling disconnected — reconnecting...",
  "banner.signaling_reconnected": "Reconnected ✓",
  "banner.reconnect_failed": "Reconnect failed. Please create the portal again.",

  // Settings
  "settings.title": "Settings",
  "settings.tab.profile": "Profile",
  "settings.tab.network": "Network",
  "settings.tab.identity": "Identity",
  "settings.tab.diagnostics": "Diagnostics",
  "settings.tab.files": "Files",
  "settings.tab.activity": "Activity",
  "settings.tab.history": "History",
  "settings.tab.about": "About",
  "settings.tab.logs": "Logs",
  "settings.back": "Back",
  "settings.language": "Language",
  "settings.language.uz": "O'zbek",
  "settings.language.en": "English",

  // Profile
  "settings.profile.title": "Profile",
  "settings.profile.username": "Username",
  "settings.profile.account_status": "Account status",
  "settings.profile.account_local": "Local account",
  "settings.profile.account_local_hint":
    "Account info is stored on this computer only. Nothing is synced to a server.",
  "settings.profile.signout": "Sign out",
  "settings.profile.reset": "Reset account",
  "settings.profile.reset_warn":
    "Reset clears the username, password and portal history. Network settings are kept. This cannot be undone.",
  "settings.profile.reset_confirm": "Yes, reset",
  "settings.profile.reset_cancel": "Cancel",
  "settings.profile.member_since": "Account created",
  "settings.profile.device_name": "Device name",
  "settings.profile.device_name_placeholder": "home, work, mac, win 64…",
  "settings.profile.device_name_hint":
    "When the same account signs in from multiple devices, room peers see which one you're on:",
  "settings.profile.autorun": "Run on system startup",
  "settings.profile.autorun_on": "On — Portal launches when your computer boots",
  "settings.profile.autorun_off": "Off — launch manually",
  "settings.profile.autorun_hint":
    "When on, an OS startup hook is installed (macOS LaunchAgent / Windows Run registry / Linux .desktop). Saved sessions auto-reconnect after reboot.",

  // Network
  "settings.network.title": "Network",
  "settings.network.signaling_label": "Signaling server URL",
  "settings.network.signaling_hint": "Default:",
  "settings.network.save": "Save",
  "settings.network.saved": "Saved ✓",
  "settings.network.turn_title": "TURN relay",
  "settings.network.turn_managed":
    "TURN is provisioned automatically by our signaling server.",
  "settings.network.turn_managed_hint":
    "If you're behind symmetric NAT or CGNAT, the signaling server hands out short-lived TURN credentials per session — there is nothing for you to configure.",
  "settings.network.turn_status_label": "Status",
  "settings.network.turn_status_active": "Active — server-managed",
  "settings.network.turn_status_idle": "Ready — engages on demand",

  // Diagnostics
  "settings.diag.title": "Diagnostics",
  "settings.diag.nat_label": "NAT type",
  "settings.diag.detecting": "Detecting...",
  "settings.diag.local": "local",
  "settings.diag.public": "public",
  "settings.diag.refresh": "Refresh",

  // Files
  "settings.files.title": "Files",
  "settings.files.save_dir_label": "Received files location",
  "settings.files.open": "Open",

  // Activity
  "settings.activity.title": "Peer connections",
  "settings.activity.empty":
    "No one has connected to your services yet. Once a guest hits \"Connect\" you'll see entries here.",
  "settings.activity.loading": "Loading…",

  // Logs
  "settings.logs.title": "Logs",
  "settings.logs.hint":
    "In-app events keep the last 500 lines, and they're also rotated to disk daily.",
  "settings.logs.show": "Show last 200 lines",
  "settings.logs.open_folder": "Open log folder",
  "settings.logs.copy": "Copy",
  "settings.logs.clear": "Clear",
  "settings.logs.close": "Close",
  "settings.logs.empty": "(empty)",

  // History
  "settings.history.title": "Recent portals",
  "settings.history.empty": "No portal history yet.",
  "settings.history.clear": "Clear history",

  // About
  "settings.about.title": "About",
  "settings.about.version": "Portal version",
  "settings.about.check_now": "Check now",
  "settings.about.checking": "Checking...",
  "settings.about.up_to_date": "✓ You're on the latest version",
  "settings.about.new_available": "New version available:",
  "settings.about.open_download": "Open download page",
  "settings.about.crashes_label": "Crash reports",
  "settings.about.crashes_hint":
    "Reports are stored locally only. They contain stack traces with home directory redacted, no portal IDs, peer IDs, IPs, or message content.",
  "settings.about.crashes_empty": "No crash reports.",
  "settings.about.crashes_more": "+ {0} more",
  "settings.about.refresh": "Refresh",
  "settings.about.clear_all": "Clear all",
  "settings.about.docs": "Documentation",

  // Common
  "common.owner": "owner",
  "common.connecting": "Connecting...",
  "common.connected": "Connected",
  "common.failed": "Failed",
  "common.tooltip.settings": "Settings",

  // Updater
  "update.available": "New version available:",
  "update.download": "Page",
  "update.install": "Install",
  "update.installing": "Installing…",
  "update.install_failed": "Install failed: ",
  "update.dismiss": "Dismiss",

  // Portal header
  "header.copy_both_tooltip": "Copy both ID and code",
  "header.copied": "Copied",
  "header.leave": "Leave",
  "header.dashboard": "Back to dashboard (portal stays connected)",
  "header.dashboard_short": "Dashboard",
  "header.close": "Close this portal (disconnect)",
  "header.switcher": "Switch portal",
  "header.switcher.title": "Active portals",
  "header.qr.close": "close",
  "header.qr.help": "Point your friend's camera at the QR\nto join the portal directly.",
  "header.copy_tooltip": "Copy",
  "header.reveal_first": "Reveal it first",
  "header.hide": "Hide",
  "header.show": "Show",

  // Chat
  "chat.placeholder": "Type a message...",
  "chat.empty": "No messages yet.",
  "chat.empty_hint": "Say hi first, or drop a file here.",
  "chat.send": "Send",
  "chat.send_file": "Send file",
  "chat.drop_to_send": "Drop — sending into the mesh starts now",

  // Lock screen
  "lock.signup.title": "Sign up",
  "lock.signup.subtitle":
    "Create an account — you'll sign in with the password next time.",
  "lock.signup.username.label": "Username",
  "lock.signup.username.placeholder": "alice",
  "lock.signup.password.label": "Password",
  "lock.signup.password.placeholder": "8+ chars, strong password",
  "lock.signup.confirm.label": "Confirm password",
  "lock.signup.confirm.placeholder": "Type it again",
  "lock.signup.submit": "Continue",
  "lock.signup.submitting": "Saving...",
  "lock.signup.replace_warning":
    "An account already exists on this device. Creating a new one will replace it (portal history is preserved).",
  "lock.signup.error.username_empty": "Please enter a username.",
  "lock.signup.error.username_too_long": "Username can't be longer than 24 characters.",
  "lock.signup.error.password_too_short": "Password must be at least 8 characters.",
  "lock.signup.error.password_weak":
    "Password is weak. Meet every rule below.",
  "lock.signup.error.mismatch": "Passwords don't match.",
  "lock.signup.error.failed": "Couldn't save. Try again.",
  "lock.signup.hint":
    "Heads up: if you forget the password, there's no recovery — only a local reset that wipes history and starts over.",
  "lock.signup.strength.title": "Password rules:",
  "lock.signup.strength.length": "At least 8 characters",
  "lock.signup.strength.lower": "Lowercase letter (a-z)",
  "lock.signup.strength.upper": "Uppercase letter (A-Z)",
  "lock.signup.strength.digit": "Digit (0-9)",
  "lock.signup.strength.special": "Special character (!@#$...)",
  "lock.signin.title": "Sign in",
  "lock.signin.subtitle": "Enter your details to access your account.",
  "lock.signin.username.label": "Username",
  "lock.signin.username.placeholder": "alice",
  "lock.signin.password.label": "Password",
  "lock.signin.password.placeholder": "••••",
  "lock.signin.submit": "Sign in",
  "lock.signin.checking": "Checking...",
  "lock.signin.error.wrong": "Wrong username or password.",
  "lock.signin.error.locked":
    "Too many attempts. Try again in {0} seconds.",
  "lock.signin.forgot": "Forgot password?",
  "lock.remember": "Remember me on this device",
  "lock.remember.hint":
    "When on, the app skips the lock on next launch. Only enable on your own computer.",
  "lock.reset.title": "Reset account?",
  "lock.reset.body":
    "There's no recovery for a forgotten password — but you can wipe the local data and start fresh. Portal history, username and password are deleted; network settings (signaling, TURN) are kept.",
  "lock.reset.confirm": "Yes, wipe and reset",
  "lock.reset.cancel": "Cancel",
};
