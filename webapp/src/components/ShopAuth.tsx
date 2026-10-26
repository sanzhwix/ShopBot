import { createContext, useCallback, useContext, useRef, useState, type FormEvent, type ReactNode } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Eye, EyeOff, KeyRound, Send } from "lucide-react";
import { api, ApiError } from "../lib/api";
import { useT } from "../lib/i18n";
import type { TextKey } from "../lib/i18n/en";
import { haptic } from "../lib/telegram";
import type { ShopSession } from "../lib/types";
import { BottomSheet } from "./BottomSheet";
import { useToast } from "./Toast";
import { Button, Field, Segmented } from "./ui";

/*
  Each shop has its own customer accounts, like its own website.
  ensureSignedIn() is called when the API answers SHOP_AUTH_REQUIRED:
    1. a saved account exists → sign in with it silently (the account manager);
    2. otherwise → open the login sheet and wait for the user.
*/

interface ShopAuthApi {
  ensureSignedIn: (session: ShopSession) => Promise<boolean>;
  openLogin: (session: ShopSession) => Promise<boolean>;
}

const ShopAuthContext = createContext<ShopAuthApi | null>(null);

export function useShopAuth() {
  const ctx = useContext(ShopAuthContext);
  if (!ctx) {
    throw new Error("useShopAuth outside ShopAuthProvider");
  }
  return ctx;
}

export function ShopAuthProvider({ children }: { children: ReactNode }) {
  const t = useT();
  const toast = useToast();
  const queryClient = useQueryClient();
  const [session, setSession] = useState<ShopSession | null>(null);
  const resolver = useRef<((ok: boolean) => void) | null>(null);

  const refresh = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: ["shop"] });
    queryClient.invalidateQueries({ queryKey: ["cart"] });
    queryClient.invalidateQueries({ queryKey: ["vault"] });
  }, [queryClient]);

  const openLogin = useCallback((s: ShopSession) => {
    setSession(s);
    return new Promise<boolean>((resolve) => {
      resolver.current = resolve;
    });
  }, []);

  const ensureSignedIn = useCallback(
    async (s: ShopSession) => {
      if (s.signed_in) {
        return true;
      }
      if (s.has_saved_account) {
        try {
          const res = await api.post<ShopSession>(`/shops/${s.shop_id}/login/vault`);
          toast(t("auth.autoSigned", { shop: s.shop_name, login: res.login }), "info");
          refresh();
          return true;
        } catch (e) {
          if (e instanceof ApiError && e.code === "VAULT_STALE") {
            toast(t("auth.err.VAULT_STALE"), "error");
          }
        }
      }
      return openLogin(s);
    },
    [openLogin, refresh, t, toast],
  );

  const finish = (ok: boolean) => {
    resolver.current?.(ok);
    resolver.current = null;
    setSession(null);
    if (ok) {
      refresh();
    }
  };

  return (
    <ShopAuthContext.Provider value={{ ensureSignedIn, openLogin }}>
      {children}
      {session && <ShopLoginSheet session={session} onDone={finish} />}
    </ShopAuthContext.Provider>
  );
}

function ShopLoginSheet({ session, onDone }: { session: ShopSession; onDone: (ok: boolean) => void }) {
  const t = useT();
  const toast = useToast();
  const config = useQuery({ queryKey: ["config"], queryFn: () => api.get<{ demo_login: string; demo_password: string }>("/config") });
  const [mode, setMode] = useState<"signin" | "register">("signin");
  const [login, setLogin] = useState("");
  const [password, setPassword] = useState("");
  const [name, setName] = useState("");
  const [save, setSave] = useState(true);
  const [showPassword, setShowPassword] = useState(false);
  const [busy, setBusy] = useState<"form" | "telegram" | null>(null);
  const [error, setError] = useState("");
  const shop = session.shop_name;

  const done = () => {
    haptic.success();
    toast(t("auth.signedIn", { shop }));
    onDone(true);
  };

  const fail = (e: unknown) => {
    haptic.error();
    const code = e instanceof ApiError ? e.code : "";
    const key = `auth.err.${code}` as TextKey;
    setError(code && t(key) !== key ? t(key) : t("common.error"));
  };

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setBusy("form");
    setError("");
    try {
      await api.post(`/shops/${session.shop_id}/${mode === "signin" ? "login" : "register"}`, { login, password, name, save });
      done();
    } catch (err) {
      fail(err);
    } finally {
      setBusy(null);
    }
  };

  const withTelegram = async () => {
    setBusy("telegram");
    setError("");
    try {
      await api.post(`/shops/${session.shop_id}/login/telegram`);
      done();
    } catch (err) {
      fail(err);
    } finally {
      setBusy(null);
    }
  };

  const demoLogin = config.data?.demo_login ?? "demo@shopbot.kz";
  const demoPassword = config.data?.demo_password ?? "demo123";

  return (
    <BottomSheet
      open
      onClose={() => onDone(false)}
      title={
        <div className="flex items-center gap-3">
          <div className="flex size-11 items-center justify-center rounded-2xl bg-soft text-accent">
            <KeyRound className="size-5" aria-hidden />
          </div>
          <h2 className="text-lg leading-tight font-bold">{t("auth.title", { shop })}</h2>
        </div>
      }
    >
      <p className="mb-4 text-sm text-hint">{t("auth.subtitle", { shop })}</p>

      <Button block onClick={withTelegram} loading={busy === "telegram"} disabled={busy !== null} className="bg-[#2aabee]! text-white!">
        <Send className="size-4" aria-hidden />
        {t("auth.telegram")}
      </Button>

      <div className="my-4 flex items-center gap-3 text-xs text-hint">
        <span className="h-px flex-1 bg-line" />
        {t("auth.or", { shop })}
        <span className="h-px flex-1 bg-line" />
      </div>

      <div className="mb-4 flex justify-center">
        <Segmented
          value={mode}
          onChange={(m) => {
            setMode(m);
            setError("");
          }}
          options={[
            { value: "signin", label: t("auth.tabSignIn") },
            { value: "register", label: t("auth.tabRegister") },
          ]}
        />
      </div>

      <form onSubmit={submit} className="space-y-3">
        {mode === "register" && <Field label={t("auth.name")} value={name} onChange={(e) => setName(e.target.value)} autoComplete="name" />}
        <Field label={t("auth.login")} value={login} onChange={(e) => setLogin(e.target.value)} autoComplete="username" inputMode="email" required />
        <div className="relative">
          <Field
            label={t("auth.password")}
            type={showPassword ? "text" : "password"}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete={mode === "signin" ? "current-password" : "new-password"}
            required
          />
          <button
            type="button"
            onClick={() => setShowPassword((v) => !v)}
            className="absolute right-2 bottom-2 rounded-lg p-1.5 text-hint hover:bg-line"
            aria-label={t("auth.password")}
          >
            {showPassword ? <EyeOff className="size-5" /> : <Eye className="size-5" />}
          </button>
        </div>

        <label className="flex cursor-pointer items-start gap-3 rounded-xl bg-soft p-3">
          <input type="checkbox" checked={save} onChange={(e) => setSave(e.target.checked)} className="mt-0.5 size-5 accent-[var(--accent)]" />
          <span>
            <span className="block text-sm font-medium">{t("auth.save")}</span>
            <span className="block text-xs text-hint">{t("auth.saveHint")}</span>
          </span>
        </label>

        {error && (
          <p className="text-sm font-medium text-danger" role="alert">
            {error}
          </p>
        )}

        <Button type="submit" block loading={busy === "form"} disabled={busy !== null}>
          {mode === "signin" ? t("auth.tabSignIn") : t("auth.tabRegister")}
        </Button>

        {mode === "signin" && (
          <p className="text-center text-xs text-hint">
            {t("auth.demo", { login: demoLogin, password: demoPassword })} ·{" "}
            <button
              type="button"
              className="font-semibold text-link"
              onClick={() => {
                setLogin(demoLogin);
                setPassword(demoPassword);
              }}
            >
              {t("auth.fill")}
            </button>
          </p>
        )}
      </form>
    </BottomSheet>
  );
}

export interface CartInput {
  product_id: number;
  size: string;
  color: string;
  qty: number;
  source: string;
}

/** Adds to cart; signs in to the shop first if the shop asks for it. */
export function useAddToCart() {
  const { ensureSignedIn } = useShopAuth();
  const queryClient = useQueryClient();
  const toast = useToast();
  const t = useT();

  return useCallback(
    async (input: CartInput): Promise<boolean> => {
      const attempt = () => api.post<{ cart_count: number }>("/cart", input);
      try {
        try {
          await attempt();
        } catch (e) {
          if (!(e instanceof ApiError) || e.code !== "SHOP_AUTH_REQUIRED") {
            throw e;
          }
          if (!(await ensureSignedIn(e.body.session as ShopSession))) {
            return false;
          }
          await attempt();
        }
      } catch (e) {
        haptic.error();
        toast(e instanceof Error ? e.message : t("common.error"), "error");
        return false;
      }
      haptic.success();
      toast(t("product.added"));
      queryClient.invalidateQueries({ queryKey: ["me"] });
      queryClient.invalidateQueries({ queryKey: ["cart"] });
      return true;
    },
    [ensureSignedIn, queryClient, t, toast],
  );
}
