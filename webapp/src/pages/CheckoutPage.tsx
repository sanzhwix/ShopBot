import { useEffect, useState } from "react";
import { Navigate, useNavigate, useParams } from "react-router";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Phone, Store, Truck } from "lucide-react";
import { api, ApiError } from "../lib/api";
import { money } from "../lib/format";
import { useMe } from "../lib/hooks";
import { useLang } from "../lib/i18n";
import { haptic, openExternal, requestPhone, inTelegram } from "../lib/telegram";
import type { CartGroup, Order, PaymentRedirect, ShopSession } from "../lib/types";
import { useShopAuth } from "../components/ShopAuth";
import { useToast } from "../components/Toast";
import { Button, Card, Field, PageHeader, Segmented, Spinner } from "../components/ui";

export function CheckoutPage() {
  const { shopId } = useParams();
  const { t, lang } = useLang();
  const navigate = useNavigate();
  const toast = useToast();
  const queryClient = useQueryClient();
  const { ensureSignedIn } = useShopAuth();
  const me = useMe();
  const cart = useQuery({ queryKey: ["cart", lang], queryFn: () => api.get<{ groups: CartGroup[] }>("/cart") });
  const [delivery, setDelivery] = useState<"pickup" | "delivery">("pickup");
  const [address, setAddress] = useState("");
  const [phone, setPhone] = useState("");
  const [comment, setComment] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (me.data?.user.phone && !phone) {
      setPhone(me.data.user.phone);
    }
  }, [me.data, phone]);

  if (cart.isPending) {
    return <Spinner />;
  }
  const group = cart.data?.groups.find((g) => String(g.shop.id) === shopId);
  if (!group) {
    return <Navigate to="/cart" replace />;
  }

  const pay = async () => {
    if (delivery === "delivery" && !address.trim()) {
      toast(t("checkout.addressRequired"), "error");
      return;
    }
    setBusy(true);
    const body = { shop_id: group.shop.id, delivery_type: delivery, address, phone, comment };
    const place = () => api.post<{ order: Order; payment: PaymentRedirect }>("/checkout", body);
    try {
      let res;
      try {
        res = await place();
      } catch (e) {
        if (!(e instanceof ApiError) || e.code !== "SHOP_AUTH_REQUIRED") {
          throw e;
        }
        if (!(await ensureSignedIn(e.body.session as ShopSession))) {
          return;
        }
        res = await place();
      }
      haptic.success();
      queryClient.invalidateQueries({ queryKey: ["cart"] });
      queryClient.invalidateQueries({ queryKey: ["me"] });
      queryClient.invalidateQueries({ queryKey: ["orders"] });
      if (res.payment.external) {
        openExternal(res.payment.url);
        navigate(`/orders/${res.order.id}`, { replace: true });
      } else {
        navigate(res.payment.url, { replace: true });
      }
    } catch (e) {
      haptic.error();
      toast(e instanceof Error ? e.message : t("common.error"), "error");
    } finally {
      setBusy(false);
    }
  };

  const sharePhone = async () => {
    const p = await requestPhone();
    if (p) {
      setPhone(p.startsWith("+") ? p : `+${p}`);
    }
  };

  return (
    <div className="mx-auto max-w-xl pb-28">
      <PageHeader back title={t("checkout.title")} subtitle={`${group.shop.logo} ${group.shop.name}`} />
      <div className="space-y-3 px-3">
        <Card className="p-3">
          <ul className="space-y-1.5 text-[15px]">
            {group.items
              .filter((i) => i.in_stock)
              .map((i) => (
                <li key={i.id} className="flex justify-between gap-3">
                  <span className="min-w-0 truncate">
                    {i.qty} × {i.name}
                    {i.size && <span className="text-hint"> · {i.size}</span>}
                  </span>
                  <span className="shrink-0 font-medium">{money(i.qty * i.price)}</span>
                </li>
              ))}
          </ul>
          <div className="mt-3 flex justify-between border-t border-line pt-3 text-lg font-bold">
            <span>{t("checkout.total")}</span>
            <span>{money(group.total)}</span>
          </div>
          {group.session.signed_in && <p className="mt-1 text-xs text-hint">{t("checkout.account", { login: group.session.login })}</p>}
        </Card>

        <Card className="space-y-3 p-3">
          <div className="flex justify-center">
            <Segmented
              value={delivery}
              onChange={setDelivery}
              options={[
                { value: "pickup", label: t("checkout.pickup") },
                { value: "delivery", label: t("checkout.delivery") },
              ]}
            />
          </div>
          {delivery === "pickup" ? (
            <p className="flex items-center gap-2 rounded-xl bg-soft p-3 text-sm">
              <Store className="size-4 shrink-0 text-hint" aria-hidden />
              {t("checkout.pickupAt", { address: group.shop.address })}
            </p>
          ) : (
            <div className="flex items-end gap-2">
              <Truck className="mb-3.5 size-5 shrink-0 text-hint" aria-hidden />
              <div className="flex-1">
                <Field label={t("checkout.address")} placeholder={t("checkout.addressPh")} value={address} onChange={(e) => setAddress(e.target.value)} />
              </div>
            </div>
          )}
          <Field label={t("checkout.phone")} type="tel" placeholder="+7 7__ ___ __ __" value={phone} onChange={(e) => setPhone(e.target.value)} />
          {inTelegram && (
            <button onClick={sharePhone} className="flex items-center gap-1.5 text-sm font-semibold text-link">
              <Phone className="size-4" aria-hidden />
              {t("checkout.sharePhone")}
            </button>
          )}
          <Field label={t("checkout.comment")} placeholder={t("checkout.commentPh")} value={comment} onChange={(e) => setComment(e.target.value)} />
        </Card>
      </div>

      <div className="fixed inset-x-0 bottom-0 z-[900] bg-page/95 px-3 pt-2 pb-[calc(12px+var(--safe-bottom))] backdrop-blur">
        <Button block onClick={pay} loading={busy} className="mx-auto max-w-xl bg-[#f14635]! text-white!">
          {t("checkout.pay", { price: money(group.total) })}
        </Button>
      </div>
    </div>
  );
}
