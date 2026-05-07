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
  "welcome.signout": "Sign out",
  "welcome.error.empty_nickname": "Please enter a nickname first",
  "welcome.error.empty_id_or_code": "Both ID and code are required",
  "welcome.error.no_such_portal":
    "This portal has been closed. The owner may have left — your saved ID + code is no longer valid.",
  "welcome.error.wrong_code": "Wrong code. Double-check it or ask the owner.",
  "welcome.error.full": "Portal is full (16-peer max).",
  "welcome.error.locked": "Owner has locked this portal. Wait for it to open.",
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
  "settings.tab.network": "Network",
  "settings.tab.identity": "Identity",
  "settings.tab.diagnostics": "Diagnostics",
  "settings.tab.files": "Files",
  "settings.tab.history": "History",
  "settings.tab.about": "About",
  "settings.back": "Back",
  "settings.language": "Language",
  "settings.language.uz": "O'zbek",
  "settings.language.en": "English",

  // Common
  "common.owner": "owner",
  "common.connecting": "Connecting...",
  "common.connected": "Connected",
  "common.failed": "Failed",
  "common.tooltip.settings": "Settings",

  // Updater
  "update.available": "New version available:",
  "update.download": "Download",
  "update.dismiss": "Dismiss",

  // Portal header
  "header.copy_both_tooltip": "Copy both ID and code",
  "header.copied": "Copied",
  "header.leave": "Leave",
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
