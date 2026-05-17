// Shared web-admin client. Two pages (login.html + dashboard.html)
// import this module via a <script> tag; nothing else does.
//
// Conventions:
//   - All API calls go through `api()` which sets Content-Type, sends
//     credentials (the session cookie) and parses JSON. Errors come
//     back as { error, lockoutSeconds } shaped per the server.
//   - i18n is a tiny key→object map; `t(key)` reads the active dict.
//     Lang persists in localStorage (same key as the desktop client
//     uses, so flipping in one place affects nothing else but feels
//     consistent if the user later runs both).

(() => {
  // ---------- i18n ----------
  const DICT = {
    uz: {
      "title.login":            "Portal — Kirish",
      "title.dashboard":        "Portal — Dashboard",
      "tab.signin":             "Kirish",
      "tab.signup":             "Ro'yxatdan o'tish",
      "auth.signup.subtitle":   "Yangi akkaunt yarating, web orqali portallaringizni boshqaring.",
      "auth.signin.subtitle":   "Akkauntingizga kiring.",
      "field.username":         "Foydalanuvchi nomi",
      "field.password":         "Parol",
      "field.confirm":          "Parolni takrorlang",
      "ph.username":            "alice",
      "ph.password.signup":     "8+ belgi, kuchli parol",
      "ph.password.signin":     "••••••••",
      "ph.password.confirm":    "Parolni qayta yozing",
      "btn.signup":             "Akkaunt yaratish",
      "btn.signin":             "Kirish",
      "btn.signout":            "Chiqish",
      "btn.password.show":      "Parolni ko'rsatish",
      "btn.password.hide":      "Parolni yashirish",
      "btn.busy.signin":        "Kirilmoqda…",
      "btn.busy.signup":        "Yaratilmoqda…",
      "remember":               "Bu qurilmada eslab qol",
      "remember.hint":          "Faqat o'zingizning kompyuteringizda yoqing.",
      "strength.title":         "Parol mezonlari:",
      "strength.length":        "Kamida 8 ta belgi",
      "strength.lower":         "Kichik harf (a-z)",
      "strength.upper":         "Katta harf (A-Z)",
      "strength.digit":         "Raqam (0-9)",
      "strength.special":       "Maxsus belgi (!@#$...)",
      "err.network":            "Server bilan ulanish bo'lmadi.",
      "err.username_taken":     "Bu nom band — boshqa nom tanlang.",
      "err.username_invalid":   "Nom 3-24 belgidan iborat bo'lishi va faqat harf/raqam/_/- dan iborat bo'lishi kerak.",
      "err.password_too_short": "Parol kamida 8 ta belgidan iborat bo'lsin.",
      "err.password_weak":      "Parol kuchsiz. Mezonlar bajarilsin.",
      "err.password_mismatch":  "Parollar mos kelmadi.",
      "err.invalid":            "Foydalanuvchi nomi yoki parol noto'g'ri.",
      "err.locked":             "Juda ko'p urinish. {0} soniyadan so'ng qayta urining.",
      "err.rate_limited":       "Juda ko'p so'rov. Bir necha daqiqadan keyin urining.",
      "err.server":             "Serverda xato.",
      "dash.greeting":          "Xush kelibsiz",
      "dash.subtitle":          "Bu yerdan akkauntingizni va portallaringizni boshqarasiz.",
      "dash.portals.title":     "Sizning portallaringiz",
      "dash.portals.empty":     "Hozircha portallar yo'q.",
      "dash.note":              "Eslatma:",
    },
    en: {
      "title.login":            "Portal — Sign in",
      "title.dashboard":        "Portal — Dashboard",
      "tab.signin":             "Sign in",
      "tab.signup":             "Sign up",
      "auth.signup.subtitle":   "Create an account to manage your portals from the web.",
      "auth.signin.subtitle":   "Sign in to your account.",
      "field.username":         "Username",
      "field.password":         "Password",
      "field.confirm":          "Confirm password",
      "ph.username":            "alice",
      "ph.password.signup":     "8+ chars, strong password",
      "ph.password.signin":     "••••••••",
      "ph.password.confirm":    "Re-enter the password",
      "btn.signup":             "Create account",
      "btn.signin":             "Sign in",
      "btn.signout":            "Sign out",
      "btn.password.show":      "Show password",
      "btn.password.hide":      "Hide password",
      "btn.busy.signin":        "Signing in…",
      "btn.busy.signup":        "Creating account…",
      "remember":               "Remember me on this device",
      "remember.hint":          "Only enable on your own computer.",
      "strength.title":         "Password rules:",
      "strength.length":        "At least 8 characters",
      "strength.lower":         "Lowercase letter (a-z)",
      "strength.upper":         "Uppercase letter (A-Z)",
      "strength.digit":         "Digit (0-9)",
      "strength.special":       "Special character (!@#$...)",
      "err.network":            "Couldn't reach the server.",
      "err.username_taken":     "Username is taken — try another.",
      "err.username_invalid":   "Username must be 3-24 chars, letters/digits/_/- only.",
      "err.password_too_short": "Password must be at least 8 characters.",
      "err.password_weak":      "Weak password. Meet every rule.",
      "err.password_mismatch":  "Passwords don't match.",
      "err.invalid":            "Wrong username or password.",
      "err.locked":             "Too many attempts. Try again in {0} seconds.",
      "err.rate_limited":       "Too many requests. Try again in a few minutes.",
      "err.server":             "Server error.",
      "dash.greeting":          "Welcome back",
      "dash.subtitle":          "Manage your account and portals from here.",
      "dash.portals.title":     "Your portals",
      "dash.portals.empty":     "No portals yet.",
      "dash.note":              "Note:",
    },
  };

  // Lang storage key shared with the desktop client. Falls back to
  // browser language hint, defaulting to UZ to match the desktop's
  // bias for an Uzbek-first audience.
  const LANG_KEY = "portal:lang";
  function readLang() {
    try {
      const v = localStorage.getItem(LANG_KEY);
      if (v === "uz" || v === "en") return v;
    } catch (_) {}
    const nav = (navigator.language || "").toLowerCase();
    return nav.startsWith("en") ? "en" : "uz";
  }
  function writeLang(l) {
    try { localStorage.setItem(LANG_KEY, l); } catch (_) {}
    state.lang = l;
    paint();
  }

  const state = { lang: readLang() };

  function t(key) {
    const dict = DICT[state.lang] || DICT.uz;
    return dict[key] || DICT.en[key] || key;
  }

  // ---------- API ----------
  // Same-origin fetch with credentials. Returns:
  //   - on 2xx: parsed JSON
  //   - on non-2xx: throws an Error with .code (= server-supplied code or "network")
  //                 and .lockoutSeconds (number, 0 if not applicable)
  async function api(path, opts = {}) {
    let res;
    try {
      res = await fetch(path, {
        method: opts.method || "GET",
        headers: { "Content-Type": "application/json" },
        credentials: "same-origin",
        body: opts.body ? JSON.stringify(opts.body) : undefined,
      });
    } catch (_) {
      const err = new Error("network");
      err.code = "network";
      throw err;
    }
    let data = null;
    if (res.headers.get("content-type")?.includes("application/json")) {
      data = await res.json().catch(() => null);
    }
    if (!res.ok) {
      const err = new Error(data?.error || `http_${res.status}`);
      err.code = data?.error || `http_${res.status}`;
      err.lockoutSeconds = data?.lockoutSeconds || 0;
      throw err;
    }
    return data;
  }

  // ---------- repaint ----------
  // Each page registers a paint() callback to refresh i18n strings
  // when the language changes.
  let painters = [];
  function paint() { painters.forEach((fn) => fn()); }
  function onPaint(fn) { painters.push(fn); fn(); }

  // ---------- common header (lang toggle) ----------
  function buildLangToggle(host) {
    host.innerHTML = `
      <div class="lang-toggle" role="group" aria-label="Language">
        <button type="button" data-lang="uz">UZ</button>
        <button type="button" data-lang="en">EN</button>
      </div>
    `;
    host.querySelectorAll("button[data-lang]").forEach((b) => {
      b.addEventListener("click", () => writeLang(b.dataset.lang));
    });
    onPaint(() => {
      host.querySelectorAll("button[data-lang]").forEach((b) => {
        b.classList.toggle("active", b.dataset.lang === state.lang);
      });
    });
  }

  // ---------- exposed ----------
  window.PortalAdmin = {
    t,
    api,
    state,
    onPaint,
    paint,
    buildLangToggle,
    setLang: writeLang,
    // Map server error codes to localised messages. The lockout case
    // takes a count param; everything else is a straight i18n lookup.
    formatError(err) {
      if (!err) return "";
      if (err.code === "locked_out") {
        return t("err.locked").replace("{0}", String(err.lockoutSeconds || 30));
      }
      const map = {
        "network":             "err.network",
        "username_taken":      "err.username_taken",
        "username_invalid":    "err.username_invalid",
        "password_too_short":  "err.password_too_short",
        "password_weak":       "err.password_weak",
        "invalid_credentials": "err.invalid",
        "no_session":          "err.invalid",
        "rate_limited":        "err.rate_limited",
        "server_error":        "err.server",
      };
      return t(map[err.code] || "err.server");
    },
    // Live password-strength evaluation; mirrors the server rules.
    strength(pwd) {
      return {
        length:  pwd.length >= 8,
        lower:   /\p{Ll}/u.test(pwd),
        upper:   /\p{Lu}/u.test(pwd),
        digit:   /[0-9]/.test(pwd),
        special: /[^\p{L}\p{N}\s]/u.test(pwd),
      };
    },
    isStrong(pwd) {
      const s = this.strength(pwd);
      return s.length && s.lower && s.upper && s.digit && s.special;
    },
  };
})();
