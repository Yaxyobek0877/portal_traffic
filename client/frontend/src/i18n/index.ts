// Tiny i18n module. Two languages (uz/en), zustand-backed, persisted
// in localStorage. The hook returns a `t` function plus the current
// language and a setter so any component can switch languages live.
//
// Why not react-i18next? Overkill for two languages and a few dozen
// strings. This file + uz.ts + en.ts is < 200 lines total and gets
// everything we need: type-safe keys, fallback to en for missing
// translations, hot language switch.

import { create } from "zustand";
import { uz } from "./uz";
import { en } from "./en";
import type { Dict, StringKey } from "./strings";

export type Language = "uz" | "en";

const STORAGE_KEY = "portal:lang";

const dictionaries: Record<Language, Dict> = { uz, en };

// detectInitial picks a sensible default the first time the user runs
// the app: persisted choice → browser hint → uz (project's home).
function detectInitial(): Language {
  if (typeof localStorage !== "undefined") {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved === "uz" || saved === "en") return saved;
  }
  if (typeof navigator !== "undefined") {
    const lang = (navigator.language || "").toLowerCase();
    if (lang.startsWith("uz")) return "uz";
    if (lang.startsWith("en")) return "en";
    // Default to en for non-uz speakers since the project's target is
    // international gamers and developers.
    return "en";
  }
  return "uz";
}

type LangStore = {
  lang: Language;
  setLang: (l: Language) => void;
};

export const useLanguage = create<LangStore>((set) => ({
  lang: detectInitial(),
  setLang: (l) => {
    if (typeof localStorage !== "undefined") {
      localStorage.setItem(STORAGE_KEY, l);
    }
    set({ lang: l });
  },
}));

// Translate a key. If the active dictionary is missing the key (which
// type-checking should prevent), fall back to the en dictionary and
// finally to the key itself, so the UI never crashes.
export function translate(lang: Language, key: StringKey): string {
  const dict = dictionaries[lang];
  if (dict && key in dict) return dict[key];
  if (key in en) return en[key];
  return key;
}

// Hook form. Re-renders the calling component whenever the language
// store changes, so a simple `setLang("en")` flips every UI string.
export function useT() {
  const lang = useLanguage((s) => s.lang);
  const setLang = useLanguage((s) => s.setLang);
  const t = (key: StringKey) => translate(lang, key);
  return { t, lang, setLang };
}
