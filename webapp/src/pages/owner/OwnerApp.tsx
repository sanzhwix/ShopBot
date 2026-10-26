import { useEffect, useState, type FormEvent } from "react";
import { Navigate, NavLink, Outlet, Route, Routes, useNavigate } from "react-router";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { BarChart3, ClipboardList, Boxes, LogOut, Settings, Store } from "lucide-react";
import { api, ApiError, streamURL } from "../../lib/api";
import { money } from "../../lib/format";
import { useLang } from "../../lib/i18n";
import { haptic, inTelegram } from "../../lib/telegram";
import type { Order, Owner, Shop } from "../../lib/types";
import { useToast } from "../../components/Toast";
import { Button, Field, Segmented, Spinner } from "../../components/ui";
import { Dashboard } from "./Dashboard";
import { OwnerOrders } from "./OwnerOrders";
import { OwnerCatalogue } from "./OwnerCatalogue";
import { OwnerSettings } from "./OwnerSettings";

export interface OwnerMe {
  owner: Owner;
  shop: Shop;
  telegram_linked: boolean;
}

export function useOwnerMe() {
  const { lang } = useLang();
  return useQuery({ queryKey: ["owner", "me", lang], queryFn: () => api.get<OwnerMe>("/owner/me"), retry: false });
}

export default function OwnerApp() {
  return (
    <Routes>
      <Route path="login" element={<OwnerLogin />} />
      <Route element={<OwnerLayout />}>
        <Route index element={<Dashboard />} />
        <Route path="orders" element={<OwnerOrders />} />
        <Route path="catalogue" element={<OwnerCatalogue />} />
        <Route path="settings" element={<OwnerSettings />} />
      </Route>
    </Routes>
  );
}

function OwnerLayout() {
  const { t, lang, setLang } = useLang();
  const me = useOwnerMe();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const live = useLiveOrders(Boolean(me.data));

  if (me.isPending) {
    return <Spinner />;
  }
  if (me.isError) {
    return me.error instanceof ApiError && me.error.status === 401 ? <Navigate to="/owner/login" replace /> : <Spinner />;
  }

  const logout = async () => {
    await api.post("/owner/logout");
    queryClient.removeQueries({ queryKey: ["owner"] });
    navigate("/owner/login");
  };

  const tabs = [
    { to: "/owner", label: t("owner.dashboard"), icon: BarChart3, end: true },
    { to: "/owner/orders", label: t("owner.orders"), icon: ClipboardList },
    { to: "/owner/catalogue", label: t("owner.catalogue"), icon: Boxes },
    { to: "/owner/settings", label: t("owner.settings"), icon: Settings },
  ];
  const { shop } = me.data;

  return (
    <div className="min-h-full">
      <header className="sticky top-0 z-[800] border-b border-line bg-card/95 backdrop-blur">
        <div className="mx-auto flex max-w-6xl items-center gap-3 px-4 py-3">
          <span className="flex size-10 items-center justify-center rounded-xl bg-soft text-2xl" aria-hidden>
            {shop.logo}
          </span>
          <div className="min-w-0 flex-1">
            <p className="truncate font-bold">{shop.name}</p>
            <p className="flex items-center gap-1.5 text-xs text-hint">
              <span className={`size-2 rounded-full ${live ? "bg-[#0ca30c]" : "bg-[#fab219]"}`} aria-hidden />
              {live ? t("owner.live") : t("owner.offline")} · {me.data.owner.name}
            </p>
          </div>
          <Segmented
            value={lang}
            onChange={setLang}
            options={[
              { value: "en", label: "EN" },
              { value: "ru", label: "RU" },
            ]}
          />
          {!inTelegram && (
            <button onClick={logout} className="rounded-full p-2 text-hint hover:bg-soft" aria-label={t("owner.logout")} title={t("owner.logout")}>
              <LogOut className="size-5" />
            </button>
          )}
        </div>
        <nav className="no-scrollbar mx-auto flex max-w-6xl gap-1 overflow-x-auto px-3" aria-label={t("owner.dashboard")}>
          {tabs.map(({ to, label, icon: Icon, end }) => (
            <NavLink
              key={to}
              to={to}
              end={end}
              className={({ isActive }) =>
                `flex shrink-0 items-center gap-1.5 border-b-2 px-3 py-2.5 text-sm font-semibold whitespace-nowrap ${
                  isActive ? "border-accent text-accent" : "border-transparent text-hint hover:text-ink"
                }`
              }
            >
              <Icon className="size-4" aria-hidden />
              {label}
            </NavLink>
          ))}
        </nav>
      </header>
      <main className="mx-auto max-w-6xl p-3 sm:p-4">
        <Outlet />
      </main>
    </div>
  );
}

/** Listens for new and changed orders and refreshes the dashboard. Returns true while connected. */
function useLiveOrders(enabled: boolean) {
  const { t } = useLang();
  const toast = useToast();
  const queryClient = useQueryClient();
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    if (!enabled) {
      return;
    }
    const source = new EventSource(streamURL("/owner/stream"));
    source.onopen = () => setConnected(true);
    source.onerror = () => setConnected(false);
    source.onmessage = (e) => {
      const event = JSON.parse(e.data) as { type: string; order: Order };
      if (event.type === "order_paid") {
        haptic.success();
        toast(t("owner.newOrder", { id: event.order.id, price: money(event.order.total) }), "success");
      }
      queryClient.invalidateQueries({ queryKey: ["owner"] });
    };
    // A slow refresh as a safety net if the stream is blocked by a proxy.
    const timer = setInterval(() => queryClient.invalidateQueries({ queryKey: ["owner", "orders"] }), 30_000);
    return () => {
      source.close();
      clearInterval(timer);
    };
  }, [enabled, queryClient, t, toast]);

  return connected;
}

const DEMO_SHOPS = [
  { email: "owner@nan-and-co.kz", label: "🥐 Nan & Co" },
  { email: "owner@steppe-wear.kz", label: "🧥 Steppe Wear" },
  { email: "owner@dala-market.kz", label: "🛒 Dala Market" },
  { email: "owner@lale-flowers.kz", label: "💐 Lale Flowers" },
  { email: "owner@volt-electronics.kz", label: "📱 Volt Electronics" },
  { email: "owner@densaulyk-pharmacy.kz", label: "💊 Densaulyk Pharmacy" },
];

function OwnerLogin() {
  const { t } = useLang();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const me = useOwnerMe();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  if (me.data) {
    return <Navigate to="/owner" replace />;
  }

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await api.post("/owner/login", { email, password });
      await queryClient.invalidateQueries({ queryKey: ["owner"] });
      navigate("/owner", { replace: true });
    } catch {
      setError(t("owner.login.error"));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="mx-auto flex min-h-dvh max-w-md flex-col justify-center p-4">
      <div className="rounded-3xl bg-card p-6">
        <div className="mb-5 flex items-center gap-3">
          <span className="flex size-12 items-center justify-center rounded-2xl bg-accent text-accent-ink" aria-hidden>
            <Store className="size-6" />
          </span>
          <div>
            <h1 className="text-xl font-bold">{t("owner.login.title")}</h1>
            <p className="text-sm text-hint">{t("owner.login.subtitle")}</p>
          </div>
        </div>
        <form onSubmit={submit} className="space-y-3">
          <Field label={t("owner.login.email")} type="email" value={email} onChange={(e) => setEmail(e.target.value)} autoComplete="username" required />
          <Field label={t("owner.login.password")} type="password" value={password} onChange={(e) => setPassword(e.target.value)} autoComplete="current-password" required />
          {error && (
            <p className="text-sm font-medium text-danger" role="alert">
              {error}
            </p>
          )}
          <Button type="submit" block loading={busy}>
            {t("owner.login.submit")}
          </Button>
        </form>
        <p className="mt-5 mb-2 text-xs font-semibold text-hint">{t("owner.login.demo")}</p>
        <div className="flex flex-wrap gap-1.5">
          {DEMO_SHOPS.map((s) => (
            <button
              key={s.email}
              onClick={() => {
                setEmail(s.email);
                setPassword("demo123");
              }}
              className="rounded-full bg-soft px-3 py-1.5 text-xs font-medium hover:bg-line"
            >
              {s.label}
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}
