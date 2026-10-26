import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { CircleCheck, CreditCard, LoaderCircle, ShieldCheck } from "lucide-react";
import { api } from "../lib/api";
import { money } from "../lib/format";
import { useLang } from "../lib/i18n";
import { haptic, openExternal } from "../lib/telegram";
import type { Order, PaymentRedirect, Shop } from "../lib/types";
import { useToast } from "../components/Toast";
import { Button, ErrorState } from "../components/ui";

const KASPI_RED = "#f14635";

interface PaymentInfo {
  order: Order;
  shop: Shop;
  payment: PaymentRedirect;
  can_confirm: boolean;
}

/**
 * A stand-in for the Kaspi payment page. Real Kaspi e-commerce needs a merchant
 * contract, so this page is clearly marked DEMO and charges nothing.
 */
export function PayPage() {
  const { orderId } = useParams();
  const { t, lang } = useLang();
  const navigate = useNavigate();
  const toast = useToast();
  const queryClient = useQueryClient();
  const [stage, setStage] = useState<"redirect" | "form" | "processing" | "done">("redirect");
  const q = useQuery({ queryKey: ["payment", orderId, lang], queryFn: () => api.get<PaymentInfo>(`/payments/${orderId}`) });

  useEffect(() => {
    const id = setTimeout(() => setStage((s) => (s === "redirect" ? "form" : s)), 1100);
    return () => clearTimeout(id);
  }, []);

  useEffect(() => {
    if (q.data && q.data.order.status !== "pending_payment") {
      setStage("done");
    }
  }, [q.data]);

  if (q.isError) {
    return <ErrorState onRetry={() => q.refetch()} />;
  }

  const pay = async () => {
    setStage("processing");
    try {
      await new Promise((r) => setTimeout(r, 900));
      await api.post(`/payments/${orderId}/confirm`);
      haptic.success();
      setStage("done");
      queryClient.invalidateQueries({ queryKey: ["orders"] });
    } catch (e) {
      haptic.error();
      toast(e instanceof Error ? e.message : t("common.error"), "error");
      setStage("form");
    }
  };

  if (stage === "redirect" || !q.data) {
    return (
      <div className="flex min-h-dvh flex-col items-center justify-center gap-4 text-hint">
        <LoaderCircle className="size-10 animate-spin" style={{ color: KASPI_RED }} aria-hidden />
        <p className="font-medium">{t("pay.redirect")}</p>
      </div>
    );
  }

  const { order, shop } = q.data;

  if (stage === "done") {
    return (
      <div className="mx-auto flex min-h-dvh max-w-md flex-col items-center justify-center gap-3 px-6 text-center">
        <CircleCheck className="size-20 text-[#0ca30c]" aria-hidden />
        <h1 className="text-2xl font-bold">{t("pay.success")}</h1>
        <p className="text-3xl font-bold">{money(order.total)}</p>
        <p className="text-hint">{t("pay.successHint", { shop: shop.name })}</p>
        <Button block className="mt-4" onClick={() => navigate(`/orders/${order.id}`, { replace: true })}>
          {t("pay.track")}
        </Button>
      </div>
    );
  }

  return (
    <div className="min-h-dvh bg-page">
      <header className="px-5 pt-6 pb-10 text-white" style={{ background: KASPI_RED }}>
        <div className="mx-auto flex max-w-md items-center justify-between">
          <span className="text-xl font-extrabold tracking-tight">Kaspi Pay</span>
          <span className="rounded-full bg-white/20 px-2.5 py-1 text-xs font-bold tracking-wide">{t("common.demo").toUpperCase()}</span>
        </div>
      </header>
      <main className="mx-auto -mt-6 max-w-md space-y-3 px-4">
        <div className="rounded-2xl bg-card p-5 shadow-sm">
          <Row label={t("pay.merchant")} value={`${shop.logo} ${shop.name}`} />
          <Row label={t("pay.order")} value={`#${order.id}`} />
          <div className="mt-3 border-t border-line pt-3">
            <p className="text-sm text-hint">{t("pay.amount")}</p>
            <p className="text-4xl font-bold">{money(order.total)}</p>
          </div>
        </div>
        <div className="rounded-2xl bg-card p-4">
          <p className="mb-2 text-sm text-hint">{t("pay.payFrom")}</p>
          <div className="flex items-center gap-3 rounded-xl p-3 shadow-[0_0_0_2px_#f14635]">
            <CreditCard className="size-6" style={{ color: KASPI_RED }} aria-hidden />
            <span className="font-semibold">{t("pay.card")}</span>
          </div>
        </div>
        <p className="flex items-center gap-2 rounded-xl bg-soft p-3 text-xs text-hint">
          <ShieldCheck className="size-4 shrink-0" aria-hidden />
          {t("pay.demoBanner")}
        </p>
        {q.data.can_confirm ? (
          <Button block onClick={pay} loading={stage === "processing"} className="py-4 text-base text-white!" style={{ background: KASPI_RED }}>
            {stage === "processing" ? t("pay.processing") : t("pay.payBtn", { price: money(order.total) })}
          </Button>
        ) : (
          <>
            <p className="text-sm text-hint">{t("pay.manual")}</p>
            <Button block onClick={() => openExternal(q.data.payment.url)} className="text-white!" style={{ background: KASPI_RED }}>
              {t("pay.openKaspi")}
            </Button>
          </>
        )}
      </main>
    </div>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between gap-3 py-1 text-[15px]">
      <span className="text-hint">{label}</span>
      <span className="truncate font-semibold">{value}</span>
    </div>
  );
}
