import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { en, type TextKey } from "./i18n/en";
import { ru } from "./i18n/ru";
import { api, setApiLang } from "./api";
import { telegramLanguage } from "./telegram";

export type Lang = "en" | "ru";
export type T = (key: TextKey, vars?: Record<string, string | number>) => string;

const dictionaries: Record<Lang, Record<TextKey, string>> = { en, ru };
const STORAGE_KEY = "shopbot.lang";

function initialLang(): Lang {
  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved === "en" || saved === "ru") {
      return saved;
    }
  } catch {
    // storage can be blocked; fall through to the defaults
  }
  const code = telegramLanguage() ?? navigator.language.slice(0, 2);
  return code === "ru" || code === "kk" ? "ru" : "en";
}

interface LangContextValue {
  lang: Lang;
  setLang: (lang: Lang) => void;
  t: T;
}

const LangContext = createContext<LangContextValue | null>(null);

export function LangProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const [lang, setLangState] = useState<Lang>(() => {
    const l = initialLang();
    setApiLang(l);
    return l;
  });

  const setLang = useCallback(
    (next: Lang) => {
      setApiLang(next);
      setLangState(next);
      try {
        localStorage.setItem(STORAGE_KEY, next);
      } catch {
        // not important: the server also stores the choice
      }
      document.documentElement.lang = next;
      api.patch("/me", { lang: next }).catch(() => {});
      // Product and shop names come localized from the API.
      queryClient.invalidateQueries();
    },
    [queryClient],
  );

  const t = useCallback<T>(
    (key, vars) => {
      let s = dictionaries[lang][key] ?? en[key] ?? key;
      if (vars) {
        for (const [k, v] of Object.entries(vars)) {
          s = s.replaceAll(`{${k}}`, String(v));
        }
      }
      return s;
    },
    [lang],
  );

  const value = useMemo(() => ({ lang, setLang, t }), [lang, setLang, t]);
  return <LangContext.Provider value={value}>{children}</LangContext.Provider>;
}

export function useLang() {
  const ctx = useContext(LangContext);
  if (!ctx) {
    throw new Error("useLang outside LangProvider");
  }
  return ctx;
}

export function useT(): T {
  return useLang().t;
}
