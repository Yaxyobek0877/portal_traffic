// Translation key registry. The single source of truth for what
// strings need to be translated; uz.ts and en.ts must implement
// every key in this type.
//
// Keep keys grouped by screen/component, in lowercase dotted form.
// New strings: add the key here, then add translations to both
// uz.ts and en.ts. TypeScript will fail compilation if either is
// out of date.

export type StringKey =
  // Welcome screen
  | "welcome.tagline"
  | "welcome.nickname.label"
  | "welcome.nickname.placeholder"
  | "welcome.portalId.label"
  | "welcome.portalId.placeholder"
  | "welcome.code.label"
  | "welcome.code.placeholder"
  | "welcome.create"
  | "welcome.join"
  | "welcome.creating"
  | "welcome.joining"
  | "welcome.cancel"
  | "welcome.back"
  | "welcome.recent"
  | "welcome.error.empty_nickname"
  | "welcome.error.empty_id_or_code"
  | "welcome.error.no_such_portal"
  | "welcome.error.wrong_code"
  | "welcome.error.full"
  | "welcome.error.locked"
  | "welcome.create_new_link"
  | "welcome.symmetric_nat_warning"

  // Banner messages
  | "banner.portal_closed"
  | "banner.signaling_disconnected"
  | "banner.signaling_reconnected"
  | "banner.reconnect_failed"

  // Settings
  | "settings.title"
  | "settings.tab.network"
  | "settings.tab.identity"
  | "settings.tab.diagnostics"
  | "settings.tab.files"
  | "settings.tab.history"
  | "settings.tab.about"
  | "settings.back"
  | "settings.language"
  | "settings.language.uz"
  | "settings.language.en"

  // Common
  | "common.owner"
  | "common.connecting"
  | "common.connected"
  | "common.failed"
  | "common.tooltip.settings"

  // Updater
  | "update.available"
  | "update.download"
  | "update.dismiss"

  // Portal header (during session)
  | "header.copy_both_tooltip"
  | "header.copied"
  | "header.leave"
  | "header.qr.close"
  | "header.qr.help"
  | "header.copy_tooltip"
  | "header.reveal_first"
  | "header.hide"
  | "header.show"

  // Chat
  | "chat.placeholder"
  | "chat.empty"
  | "chat.empty_hint"
  | "chat.send"
  | "chat.send_file"
  | "chat.drop_to_send";

// Helper: when the key is missing in the active dictionary, fall back
// to the en dictionary, then to the key itself. Catches dev-time
// gaps without crashing the UI.
export type Dict = Record<StringKey, string>;
