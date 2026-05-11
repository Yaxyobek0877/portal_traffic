// Tiny i18n module — same role as client/frontend/src/i18n in the desktop
// build, scaled down to two languages and a single-file dictionary. We
// keep the API surface intentionally small (one Map lookup, one Compose
// helper) because Portal's UI vocabulary is < 200 strings; pulling in
// the full Android resources XML pipeline for that would dwarf the
// payload it carries.
//
// Usage:
//   - Wrap the Compose tree in `LangScope { ... }` near the top of the
//     activity so `LocalLang.current` is set.
//   - Inside composables: `Text(t("welcome.create"))`. For interpolated
//     strings: `t("portal.title.reconnecting", attempt)` (uses Kotlin's
//     `String.format`, so use %s/%d markers in the dictionary).
//
// Keys live in a single Map per language (no nested objects). When a
// key is missing in the active language we fall through to English,
// then to the raw key — so the UI never crashes on an unknown key,
// and a missing translation is visible without making the screen
// blank.

package uz.aihealth.portal_mobile.i18n

import androidx.compose.runtime.Composable
import androidx.compose.runtime.compositionLocalOf

enum class Lang(val code: String, val label: String) {
    UZ("uz", "UZ"),
    EN("en", "EN");

    companion object {
        fun fromCode(code: String?): Lang =
            values().firstOrNull { it.code.equals(code, ignoreCase = true) } ?: UZ
    }
}

/**
 * CompositionLocal carrying the active language. The default is UZ —
 * matches the desktop client's "Toshkent-built app, default to Uzbek"
 * heuristic. Override at the activity root via [LangScope].
 */
val LocalLang = compositionLocalOf { Lang.UZ }

/**
 * Compose helper. Resolves [key] against the active language, falling
 * back to EN and then the raw key.
 */
@Composable
fun t(key: String): String = Strings.get(LocalLang.current, key)

/**
 * Same as [t] but applies [String.format] with [args]. Use %s / %d in
 * the dictionary entry. Off-thread safe; doesn't touch any state.
 */
@Composable
fun t(key: String, vararg args: Any?): String =
    Strings.get(LocalLang.current, key).format(*args)

object Strings {
    fun get(lang: Lang, key: String): String =
        (when (lang) {
            Lang.UZ -> uz[key] ?: en[key]
            Lang.EN -> en[key] ?: uz[key]
        }) ?: key

    private val uz: Map<String, String> = mapOf(
        // -------- common --------
        "common.save" to "Saqlash",
        "common.saved" to "Saqlandi",
        "common.cancel" to "Bekor qilish",
        "common.back" to "Orqaga",
        "common.close" to "Yopish",
        "common.clear" to "Tozalash",
        "common.now" to "hozir",
        "common.min_ago" to "%d daq oldin",
        "common.hour_ago" to "%d soat oldin",
        "common.day_ago" to "%d kun oldin",
        "common.host" to "host",
        "common.joiner" to "joiner",
        "common.copy" to "Nusxa",
        "common.lang" to "Til",

        // -------- welcome --------
        "welcome.tagline" to "To'g'ridan-to'g'ri ulanish.\nOrada server yo'q.",
        "welcome.nickname.label" to "Taxallusingiz",
        "welcome.create" to "Yangi portal yaratish",
        "welcome.join" to "Mavjud portalga qo'shilish",
        "welcome.settings" to "Sozlamalar",
        "welcome.recent" to "Yaqindagi portallar",

        // -------- join --------
        "join.title" to "Portalga qo'shilish",
        "join.scan_qr" to "QR kodni skanerlash",
        "join.qr_error" to "QR ichida 6 raqamli ID va kod topilmadi",
        "join.qr_prompt" to "QR ni kameraga tuting",
        "join.or_manual" to "yoki qo'l bilan kiriting",
        "join.portal_id_label" to "Portal ID (6 raqam)",
        "join.code_label" to "Kod (6 raqam)",
        "join.submit" to "Qo'shilish",

        // -------- portal --------
        "portal.title.owner" to "Portal: %s",
        "portal.title.joiner" to "Portal %s",
        "portal.title.connecting" to "Ulanmoqda…",
        "portal.title.reconnecting" to "Qayta ulanmoqda… (%d/8)",
        "portal.title.error" to "Xatolik",
        "portal.title.fallback" to "Portal",
        "portal.exit" to "Chiqish",
        "portal.id_label" to "Portal ID",
        "portal.code_label" to "Kod",
        "portal.your_ip" to "Sizning IP",
        "portal.share_code" to "Bu kodni do'stlaringiz bilan ulashing.",
        "portal.qr_button" to "QR",
        "portal.no_peers" to "Hech kim ulangan emas",
        "portal.peers_count" to "Peer'lar (%d)",
        "portal.peer.connected" to "ulangan",
        "portal.peer.connecting" to "ulanmoqda",
        "portal.peer.failed" to "xato",
        "portal.peer.closed" to "yopildi",
        "portal.transfers_count" to "Fayllar (%d)",
        "portal.transfer.error" to "xato: %s",
        "portal.transfer.done_with_path" to "tayyor — %s",
        "portal.transfer.done" to "tayyor",
        "portal.chat.placeholder" to "Xabar yozing…",
        "portal.chat.send" to "Yuborish",
        "portal.qr.title" to "Portal taklifi",
        "portal.qr.hint" to "QR ni do'stingizning kamerasiga tutsangiz, portalga to'g'ridan-to'g'ri kiradi.",
        "portal.connecting_signal" to "Signal serveriga ulanmoqda…",
        "portal.error" to "Xato: %s",
        "portal.reconnecting_msg" to "Tarmoq vaqtinchalik uzildi. Urinish %d/8…",

        // -------- settings --------
        "settings.title" to "Sozlamalar",
        "settings.network" to "Tarmoq",
        "settings.signal_url" to "Signal serveri URL",
        "settings.signal_url_hint" to "wss:// yoki ws:// bilan boshlanishi shart",
        "settings.default" to "Standart: %s",
        "settings.url_validation" to "URL wss:// yoki ws:// bilan boshlanishi kerak",
        "settings.reset" to "Standartga qaytarish",

        "settings.cf_turn" to "Cloudflare TURN (tavsiya etiladi)",
        "settings.cf_turn.intro" to "Simmetrik NAT (CGNAT, mobile internet) ortida bo'lsangiz — bu eng oson va ishonchli yo'l. Bepul tarif: 1 TB/oy.",
        "settings.cf_turn.steps" to "1. Cloudflare → Calls → TURN\n2. \"Create TURN Service\" → nom: portal\n3. \"View Credentials\"\n4. Token ID va API Token ni shu yerga yopishtiring",
        "settings.cf_turn.token_id" to "Token ID",
        "settings.cf_turn.api_token" to "API Token",
        "settings.cf_turn.test" to "Sinash",
        "settings.cf_turn.testing" to "Sinalmoqda…",
        "settings.cf_turn.ok" to "Cloudflare TURN ishlamoqda — short-lived credentials qaytarildi ✓",
        "settings.cf_turn.empty" to "Cloudflare TURN sozlanmagan",

        "settings.manual_turn" to "Qo'lda TURN (o'z serveringiz)",
        "settings.manual_turn.intro" to "Agar siz va do'stingiz har xil tarmoqlarda Simmetrik NAT ortida bo'lsangiz, to'g'ridan-to'g'ri ulanish ishlamaydi — TURN serveri ma'lumotni o'tkazib beradi.",
        "settings.manual_turn.url_label" to "TURN URL (har qatorga bittadan)",
        "settings.manual_turn.username" to "Foydalanuvchi nomi",
        "settings.manual_turn.credential" to "Parol / kredensial",
        "settings.manual_turn.free" to "Bepul TURN ni yoqish",
        "settings.manual_turn.free_hint" to "Open Relay Project — bepul, sekinroq, lekin tezda sinash uchun yetadi",
        "settings.manual_turn.applies_next" to "Sozlangandan keyin keyingi portal yaratish/qo'shilishda kuchga kiradi.",

        "settings.about" to "Haqida",
        "settings.about.version" to "Portal versiyasi",
        "settings.about.docs" to "Hujjatlar",

        // -------- auth --------
        "auth.signin.title" to "Hisobga kirish",
        "auth.signup.title" to "Yangi hisob ochish",
        "auth.username" to "Foydalanuvchi nomi",
        "auth.username_hint" to "3-24 belgi · A-Z, 0-9, _ va -",
        "auth.password" to "Parol",
        "auth.password_hint" to "8+ belgi · katta+kichik harf, raqam, belgi",
        "auth.signin.cta" to "Kirish",
        "auth.signup.cta" to "Ro'yxatdan o'tish",
        "auth.toggle_to_signup" to "Hisobingiz yo'qmi? Ro'yxatdan o'ting",
        "auth.toggle_to_signin" to "Allaqachon hisob bormi? Kirish",
        "auth.greeting" to "Salom, %s",
        "auth.signed_in_as" to "Kirgan: %s",
        "auth.signout" to "Chiqish",
        "auth.banner.required" to "Davom etish uchun hisobingizga kiring yoki yangi hisob oching. Hisob — P2P portallaringiz va boshqa cloud xizmatlar uchun.",
        "auth.busy" to "Yuborilmoqda…",
        "auth.connecting_to" to "Server: %s",
        "auth.show_password" to "Ko'rsatish",
        "auth.hide_password" to "Yashirish",
        "auth.password_confirm" to "Parolni qayta yozing",
        "auth.password_confirm_hint" to "Yuqoridagi parolni takrorlang",
        "auth.password_mismatch" to "Parollar mos kelmadi",

        // Server-side error codes (mirroring server/userstore.go)
        "auth.err.username_taken" to "Bu foydalanuvchi nomi allaqachon band",
        "auth.err.username_invalid" to "Foydalanuvchi nomi noto'g'ri (3-24 belgi, ASCII)",
        "auth.err.password_too_short" to "Parol kamida 8 belgi bo'lishi kerak",
        "auth.err.password_weak" to "Parol kuchsiz: katta+kichik harf, raqam va maxsus belgi kerak",
        "auth.err.invalid_credentials" to "Foydalanuvchi nomi yoki parol noto'g'ri",
        "auth.err.locked_out" to "Bloklangan — %d soniyadan keyin urinib ko'ring",
        "auth.err.no_session" to "Sessiya tugadi — qaytadan kirish kerak",
        "auth.err.bad_request" to "So'rov noto'g'ri",
        "auth.err.server_outdated" to "Server bu API ni qo'llab-quvvatlamaydi — server yangi versiyaga yangilanishi kerak",
        "auth.err.unknown" to "Noma'lum xato",
        "auth.err.network" to "Server bilan ulanish bo'lmadi: %s",

        // -------- errors --------
        "error.empty_nickname" to "Taxallus bo'sh bo'la olmaydi",
    )

    private val en: Map<String, String> = mapOf(
        "common.save" to "Save",
        "common.saved" to "Saved",
        "common.cancel" to "Cancel",
        "common.back" to "Back",
        "common.close" to "Close",
        "common.clear" to "Clear",
        "common.now" to "now",
        "common.min_ago" to "%d min ago",
        "common.hour_ago" to "%d hr ago",
        "common.day_ago" to "%d days ago",
        "common.host" to "host",
        "common.joiner" to "joiner",
        "common.copy" to "Copy",
        "common.lang" to "Lang",

        "welcome.tagline" to "Direct connections.\nNo middleman.",
        "welcome.nickname.label" to "Your nickname",
        "welcome.create" to "Create new portal",
        "welcome.join" to "Join existing portal",
        "welcome.settings" to "Settings",
        "welcome.recent" to "Recent portals",

        "join.title" to "Join portal",
        "join.scan_qr" to "Scan QR code",
        "join.qr_error" to "QR didn't contain a 6-digit ID and code",
        "join.qr_prompt" to "Hold the QR in front of camera",
        "join.or_manual" to "or enter manually",
        "join.portal_id_label" to "Portal ID (6 digits)",
        "join.code_label" to "Code (6 digits)",
        "join.submit" to "Join",

        "portal.title.owner" to "Portal: %s",
        "portal.title.joiner" to "Portal %s",
        "portal.title.connecting" to "Connecting…",
        "portal.title.reconnecting" to "Reconnecting… (%d/8)",
        "portal.title.error" to "Error",
        "portal.title.fallback" to "Portal",
        "portal.exit" to "Exit",
        "portal.id_label" to "Portal ID",
        "portal.code_label" to "Code",
        "portal.your_ip" to "Your IP",
        "portal.share_code" to "Share this code with your friends.",
        "portal.qr_button" to "QR",
        "portal.no_peers" to "No peers connected",
        "portal.peers_count" to "Peers (%d)",
        "portal.peer.connected" to "connected",
        "portal.peer.connecting" to "connecting",
        "portal.peer.failed" to "failed",
        "portal.peer.closed" to "closed",
        "portal.transfers_count" to "Files (%d)",
        "portal.transfer.error" to "error: %s",
        "portal.transfer.done_with_path" to "done — %s",
        "portal.transfer.done" to "done",
        "portal.chat.placeholder" to "Write a message…",
        "portal.chat.send" to "Send",
        "portal.qr.title" to "Portal invite",
        "portal.qr.hint" to "If your friend scans this with their camera, they'll join the portal directly.",
        "portal.connecting_signal" to "Connecting to the signal server…",
        "portal.error" to "Error: %s",
        "portal.reconnecting_msg" to "Network temporarily lost. Attempt %d/8…",

        "settings.title" to "Settings",
        "settings.network" to "Network",
        "settings.signal_url" to "Signal server URL",
        "settings.signal_url_hint" to "Must start with wss:// or ws://",
        "settings.default" to "Default: %s",
        "settings.url_validation" to "URL must start with wss:// or ws://",
        "settings.reset" to "Reset to default",

        "settings.cf_turn" to "Cloudflare TURN (recommended)",
        "settings.cf_turn.intro" to "Behind symmetric NAT (CGNAT, mobile data)? This is the easiest and most reliable path. Free tier: 1 TB/mo.",
        "settings.cf_turn.steps" to "1. Cloudflare → Calls → TURN\n2. \"Create TURN Service\" → name: portal\n3. \"View Credentials\"\n4. Paste Token ID and API Token here",
        "settings.cf_turn.token_id" to "Token ID",
        "settings.cf_turn.api_token" to "API Token",
        "settings.cf_turn.test" to "Test",
        "settings.cf_turn.testing" to "Testing…",
        "settings.cf_turn.ok" to "Cloudflare TURN works — short-lived credentials returned ✓",
        "settings.cf_turn.empty" to "Cloudflare TURN not configured",

        "settings.manual_turn" to "Manual TURN (your own server)",
        "settings.manual_turn.intro" to "If both you and your peer are behind symmetric NAT, direct connection won't work — a TURN server relays the traffic.",
        "settings.manual_turn.url_label" to "TURN URL (one per line)",
        "settings.manual_turn.username" to "Username",
        "settings.manual_turn.credential" to "Password / credential",
        "settings.manual_turn.free" to "Enable free TURN",
        "settings.manual_turn.free_hint" to "Open Relay Project — free, slower, fine for quick tests",
        "settings.manual_turn.applies_next" to "Takes effect on the next portal create / join.",

        "settings.about" to "About",
        "settings.about.version" to "Portal version",
        "settings.about.docs" to "Documentation",

        "error.empty_nickname" to "Nickname can't be empty",

        "auth.signin.title" to "Sign in",
        "auth.signup.title" to "Create account",
        "auth.username" to "Username",
        "auth.username_hint" to "3-24 chars · A-Z, 0-9, _ and -",
        "auth.password" to "Password",
        "auth.password_hint" to "8+ chars · upper+lower+digit+special",
        "auth.signin.cta" to "Sign in",
        "auth.signup.cta" to "Create account",
        "auth.toggle_to_signup" to "No account? Sign up",
        "auth.toggle_to_signin" to "Already have an account? Sign in",
        "auth.greeting" to "Hi, %s",
        "auth.signed_in_as" to "Signed in as %s",
        "auth.signout" to "Sign out",
        "auth.banner.required" to "Sign in or create an account to continue. Your account is for your portals and other cloud features.",
        "auth.busy" to "Sending…",
        "auth.connecting_to" to "Server: %s",
        "auth.show_password" to "Show",
        "auth.hide_password" to "Hide",
        "auth.password_confirm" to "Confirm password",
        "auth.password_confirm_hint" to "Repeat the password above",
        "auth.password_mismatch" to "Passwords don't match",

        "auth.err.username_taken" to "That username is already taken",
        "auth.err.username_invalid" to "Username is invalid (3-24 ASCII chars)",
        "auth.err.password_too_short" to "Password must be at least 8 characters",
        "auth.err.password_weak" to "Password is weak — needs upper+lower+digit+special",
        "auth.err.invalid_credentials" to "Wrong username or password",
        "auth.err.locked_out" to "Locked — try again in %d seconds",
        "auth.err.no_session" to "Session expired — please sign in again",
        "auth.err.bad_request" to "Bad request",
        "auth.err.server_outdated" to "Server doesn't support this API — needs to be updated",
        "auth.err.unknown" to "Unknown error",
        "auth.err.network" to "Couldn't reach the server: %s",
    )
}
