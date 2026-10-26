import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { MapPin, MessageCircle, Phone, Store, Truck } from "lucide-react";
import { api } from "../../lib/api";
import { ago, money } from "../../lib/format";
import { useLang } from "../../lib/i18n";
import type { TextKey } from "../../lib/i18n/en";
import { haptic, openTelegram } from "../../lib/telegram";
import type { Order, OrderStatus } from "../../lib/types";
import { useToast } from "../../components/Toast";
import { Button, Chip, EmptyState, Spinner } from "../../components/ui";
import { StatusPill } from "../OrdersPage";

const FILTERS = ["active", "", "delivered", "cancelled"] as const;
const NEXT: Partial<Record<OrderStatus, OrderStatus>> = { pending_payment: "paid", paid: "accepted", accepted: "ready", ready: "delivered" };

export function OwnerOrders() {
  const { t, lang } = useLang();
  const [filter, setFilter] = useState<(typeof FILTERS)[number]>("active");
  const q = useQuery({
    queryKey: ["owner", "orders", filter, lang],
    queryFn: () => api.get<{ orders: Order[] }>(`/owner/orders?status=${filter}`),
  });
  const filterLabel = (f: string) => t(`owner.orders.${f || "all"}` as TextKey);

  return (
    <div className="space-y-3">
      <div className="no-scrollbar flex gap-2 overflow-x-auto">
        {FILTERS.map((f) => (
          <Chip key={f} active={filter === f} onClick={() => setFilter(f)}>
            {filterLabel(f)}
          </Chip>
        ))}
      </div>
      {q.isPending ? (
        <Spinner />
      ) : q.data?.orders.length === 0 ? (
        <EmptyState icon="📭" title={t("owner.orders.empty")} />
      ) : (
        <div className="grid gap-3 md:grid-cols-2">
          {q.data?.orders.map((o) => <OrderCard key={o.id} order={o} />)}
        </div>
      )}
    </div>
  );
}

function OrderCard({ order: o }: { order: Order }) {
  const { t, lang } = useLang();
  const toast = useToast();
  const queryClient = useQueryClient();
  const [busy, setBusy] = useState<string | null>(null);
  const next = NEXT[o.status];
  const c = o.customer;

  const setStatus = async (status: OrderStatus) => {
    setBusy(status);
    try {
      await api.patch(`/owner/orders/${o.id}`, { status });
      haptic.success();
      toast(`#${o.id}: ${t(`status.${status}` as TextKey)}`);
      queryClient.invalidateQueries({ queryKey: ["owner"] });
    } catch (e) {
      toast(e instanceof Error ? e.message : t("common.error"), "error");
    } finally {
      setBusy(null);
    }
  };

  // Real customers have Telegram ids; demo customers (negative ids) cannot be messaged.
  const chatURL = c.id <= 0 ? "" : c.username ? `https://t.me/${c.username}` : `tg://user?id=${c.id}`;

  return (
    <article className="rounded-2xl bg-card p-4">
      <header className="flex items-start justify-between gap-2">
        <div>
          <p className="font-bold">#{o.id}</p>
          <p className="text-xs text-hint">
            {ago(o.created_at, lang)} · {t(`source.${o.source}` as TextKey)}
          </p>
        </div>
        <StatusPill status={o.status} />
      </header>

      <div className="mt-3 flex items-center gap-3">
        <span className="flex size-10 items-center justify-center rounded-full bg-soft font-semibold" aria-hidden>
          {c.first_name.slice(0, 1)}
        </span>
        <div className="min-w-0 flex-1">
          <p className="truncate font-semibold">
            {c.first_name} {c.last_name}
          </p>
          <p className="truncate text-xs text-hint">
            {c.username && `@${c.username}`}
            {o.phone && (
              <a href={`tel:${o.phone.replace(/\s/g, "")}`} className="ml-2 inline-flex items-center gap-1 text-link">
                <Phone className="size-3" aria-hidden />
                {o.phone}
              </a>
            )}
          </p>
        </div>
        {chatURL && (
          <Button variant="secondary" className="px-3 py-2 text-sm" onClick={() => openTelegram(chatURL)}>
            <MessageCircle className="size-4" aria-hidden />
            {t("owner.message")}
          </Button>
        )}
      </div>

      <ul className="mt-3 space-y-1 text-sm">
        {o.items.map((it, i) => (
          <li key={i} className="flex justify-between gap-2">
            <span className="min-w-0 truncate">
              {it.image} {it.qty} × {it.name}
              {(it.size || it.color) && (
                <span className="text-hint"> · {[it.size, it.color && t(`color.${it.color}` as TextKey)].filter(Boolean).join(", ")}</span>
              )}
            </span>
            <span className="shrink-0">{money(it.qty * it.price)}</span>
          </li>
        ))}
      </ul>
      <p className="mt-2 flex justify-between border-t border-line pt-2 font-bold">
        <span>{t("order.total")}</span>
        <span>{money(o.total)}</span>
      </p>

      <p className="mt-2 flex items-center gap-1.5 text-sm text-hint">
        {o.delivery_type === "delivery" ? <Truck className="size-4" aria-hidden /> : <Store className="size-4" aria-hidden />}
        {o.delivery_type === "delivery" ? t("order.delivery") : t("order.pickup")}
        {o.address && (
          <>
            <MapPin className="ml-1 size-4" aria-hidden />
            <span className="truncate">{o.address}</span>
          </>
        )}
      </p>
      {o.comment && <p className="mt-1 rounded-lg bg-soft px-2.5 py-1.5 text-sm">💬 {o.comment}</p>}

      {(next || o.status !== "delivered") && o.status !== "cancelled" && (
        <div className="mt-3 flex gap-2">
          {next && (
            <Button className="flex-1 py-2.5 text-sm" loading={busy === next} disabled={busy !== null} onClick={() => setStatus(next)}>
              {next === "paid" ? t("status.paid") : t(`owner.next.${next}` as TextKey)}
            </Button>
          )}
          {o.status !== "delivered" && (
            <Button variant="danger" className="py-2.5 text-sm" loading={busy === "cancelled"} disabled={busy !== null} onClick={() => setStatus("cancelled")}>
              {t("owner.cancel")}
            </Button>
          )}
        </div>
      )}
    </article>
  );
}
