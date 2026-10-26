import { Link, useNavigate, useParams } from "react-router";
import { useQuery } from "@tanstack/react-query";
import { Check, ChevronRight, KeyRound, MapPin, Star, Store, Truck } from "lucide-react";
import { api } from "../lib/api";
import { dateTime, money } from "../lib/format";
import { useLang } from "../lib/i18n";
import type { TextKey } from "../lib/i18n/en";
import type { Order, OrderStatus } from "../lib/types";
import { Button, Card, EmptyState, ErrorState, PageHeader, ProductImage, Spinner } from "../components/ui";

// Status identity is the colored dot; the text stays in the normal ink color, so it reads in both themes.
const STATUS_COLOR: Record<OrderStatus, string> = {
  pending_payment: "#fab219",
  paid: "#2a78d6",
  accepted: "#6250d6",
  ready: "#eb6834",
  delivered: "#0ca30c",
  cancelled: "#d03b3b",
};

export function StatusPill({ status }: { status: OrderStatus }) {
  const { t } = useLang();
  return (
    <span className="inline-flex shrink-0 items-center gap-1.5 rounded-full bg-soft px-2.5 py-1 text-xs font-semibold">
      <span className="size-2 rounded-full" style={{ background: STATUS_COLOR[status] }} aria-hidden />
      {t(`status.${status}` as TextKey)}
    </span>
  );
}

export function OrdersPage() {
  const { t, lang } = useLang();
  const navigate = useNavigate();
  const q = useQuery({
    queryKey: ["orders", lang],
    queryFn: () => api.get<{ orders: Order[] }>("/orders"),
    refetchInterval: 15_000,
  });

  if (q.isPending) {
    return <Spinner />;
  }
  if (q.isError) {
    return <ErrorState onRetry={() => q.refetch()} />;
  }
  return (
    <div className="mx-auto max-w-xl">
      <PageHeader title={t("orders.title")} />
      {q.data.orders.length === 0 && (
        <EmptyState
          icon="📦"
          title={t("orders.empty")}
          action={
            <Button variant="secondary" onClick={() => navigate("/map")}>
              {t("cart.openMap")}
            </Button>
          }
        />
      )}
      <ul className="space-y-2 px-3">
        {q.data.orders.map((o) => (
          <li key={o.id}>
            <Link to={`/orders/${o.id}`} className="flex items-center gap-3 rounded-2xl bg-card p-3 hover:bg-soft">
              <span className="flex size-12 items-center justify-center rounded-xl bg-soft text-2xl" aria-hidden>
                {o.shop_logo}
              </span>
              <span className="min-w-0 flex-1">
                <span className="flex items-center justify-between gap-2">
                  <span className="truncate font-semibold">{o.shop_name}</span>
                  <StatusPill status={o.status} />
                </span>
                <span className="mt-0.5 flex justify-between text-sm text-hint">
                  <span>
                    #{o.id} · {dateTime(o.created_at, lang)}
                  </span>
                  <span className="font-semibold text-ink">{money(o.total)}</span>
                </span>
              </span>
              <ChevronRight className="size-5 text-hint" aria-hidden />
            </Link>
          </li>
        ))}
      </ul>
    </div>
  );
}

const STEPS: OrderStatus[] = ["paid", "accepted", "ready", "delivered"];

export function OrderPage() {
  const { id } = useParams();
  const { t, lang } = useLang();
  const navigate = useNavigate();
  const q = useQuery({
    queryKey: ["orders", id, lang],
    queryFn: () => api.get<{ order: Order }>(`/orders/${id}`),
    refetchInterval: 8_000,
  });

  if (q.isPending) {
    return <Spinner />;
  }
  if (q.isError) {
    return <ErrorState onRetry={() => q.refetch()} />;
  }
  const o = q.data.order;
  const current = STEPS.indexOf(o.status);

  return (
    <div className="mx-auto max-w-xl pb-6">
      <PageHeader back title={t("order.title", { id: o.id })} subtitle={`${o.shop_logo} ${o.shop_name} · ${dateTime(o.created_at, lang)}`} right={<StatusPill status={o.status} />} />
      <div className="space-y-3 px-3">
        {o.status === "pending_payment" && (
          <Button block onClick={() => navigate(`/pay/${o.id}`)} className="bg-[#f14635]! text-white!">
            {t("order.payNow")} · {money(o.total)}
          </Button>
        )}

        {o.status !== "cancelled" && o.status !== "pending_payment" && (
          <Card className="p-4">
            <ol className="flex items-start justify-between">
              {STEPS.map((s, i) => {
                const done = i <= current;
                return (
                  <li key={s} className="relative flex flex-1 flex-col items-center gap-1.5 text-center">
                    {i > 0 && (
                      <span className={`absolute top-3.5 right-1/2 h-0.5 w-full -translate-y-1/2 ${i <= current ? "bg-[#0ca30c]" : "bg-line"}`} aria-hidden />
                    )}
                    <span
                      className={`relative z-10 flex size-7 items-center justify-center rounded-full text-xs font-bold ${done ? "bg-[#0ca30c] text-white" : "bg-soft text-hint"}`}
                    >
                      {done ? <Check className="size-4" aria-hidden /> : i + 1}
                    </span>
                    <span className={`text-xs ${i === current ? "font-semibold" : "text-hint"}`}>{t(`status.${s}` as TextKey)}</span>
                  </li>
                );
              })}
            </ol>
          </Card>
        )}

        <Card className="p-3">
          <h2 className="mb-1 font-semibold">{t("order.items")}</h2>
          <ul className="divide-y divide-line">
            {o.items.map((it, i) => (
              <li key={i} className="flex items-center gap-3 py-2">
                <ProductImage photo={it.photo} emoji={it.image} alt={it.name} size="sm" />
                <span className="min-w-0 flex-1">
                  <span className="block truncate">{it.name}</span>
                  <span className="block text-xs text-hint">
                    {it.qty} × {money(it.price)}
                    {[it.size, it.color && t(`color.${it.color}` as TextKey)].filter(Boolean).map((v) => ` · ${v}`)}
                  </span>
                </span>
                <span className="font-semibold">{money(it.qty * it.price)}</span>
              </li>
            ))}
          </ul>
          <div className="mt-2 flex justify-between border-t border-line pt-2 text-lg font-bold">
            <span>{t("order.total")}</span>
            <span>{money(o.total)}</span>
          </div>
        </Card>

        <Card className="space-y-2 p-3 text-sm">
          <p className="flex items-center gap-2">
            {o.delivery_type === "delivery" ? <Truck className="size-4 text-hint" aria-hidden /> : <Store className="size-4 text-hint" aria-hidden />}
            <span className="font-medium">{o.delivery_type === "delivery" ? t("order.delivery") : t("order.pickup")}</span>
          </p>
          {o.address && (
            <p className="flex items-center gap-2 text-hint">
              <MapPin className="size-4" aria-hidden />
              {o.address}
            </p>
          )}
          {o.account_login && (
            <p className="flex items-center gap-2 text-hint">
              <KeyRound className="size-4" aria-hidden />
              {t("order.account")}: {o.account_login}
            </p>
          )}
        </Card>

        {o.status === "delivered" && (
          <Button block variant="secondary" onClick={() => navigate("/feedback")}>
            <Star className="size-4" aria-hidden />
            {t("profile.feedback")}
          </Button>
        )}
      </div>
    </div>
  );
}
