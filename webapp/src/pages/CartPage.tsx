import { Link, useNavigate } from "react-router";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { KeyRound, Trash2 } from "lucide-react";
import { api } from "../lib/api";
import { money } from "../lib/format";
import { useLang } from "../lib/i18n";
import type { TextKey } from "../lib/i18n/en";
import { haptic } from "../lib/telegram";
import type { CartGroup } from "../lib/types";
import { QtyStepper } from "../components/Product";
import { Button, Card, EmptyState, ErrorState, PageHeader, ProductImage, Spinner } from "../components/ui";

export function CartPage() {
  const { t, lang } = useLang();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const q = useQuery({ queryKey: ["cart", lang], queryFn: () => api.get<{ groups: CartGroup[]; count: number }>("/cart") });

  const setQty = async (itemId: number, qty: number) => {
    haptic.select();
    const data = await api.patch<{ groups: CartGroup[]; count: number }>(`/cart/${itemId}`, { qty });
    queryClient.setQueryData(["cart", lang], data);
    queryClient.invalidateQueries({ queryKey: ["me"] });
  };

  if (q.isPending) {
    return <Spinner />;
  }
  if (q.isError) {
    return <ErrorState onRetry={() => q.refetch()} />;
  }
  const groups = q.data.groups;

  return (
    <div className="mx-auto max-w-xl">
      <PageHeader title={t("cart.title")} subtitle={groups.length > 1 ? t("cart.perShop") : undefined} />
      {groups.length === 0 && (
        <EmptyState
          icon="🧺"
          title={t("cart.empty")}
          hint={t("cart.emptyHint")}
          action={
            <Button onClick={() => navigate("/map")} variant="secondary">
              {t("cart.openMap")}
            </Button>
          }
        />
      )}
      <div className="space-y-3 px-3">
        {groups.map((g) => (
          <Card key={g.shop.id} className="p-3">
            <Link to={`/shop/${g.shop.id}`} className="flex items-center gap-3 pb-2">
              <span className="flex size-10 items-center justify-center rounded-xl bg-soft text-2xl" aria-hidden>
                {g.shop.logo}
              </span>
              <span className="min-w-0 flex-1">
                <span className="block truncate font-semibold">{g.shop.name}</span>
                <span className="flex items-center gap-1 truncate text-xs text-hint">
                  <KeyRound className="size-3.5" aria-hidden />
                  {g.session.signed_in ? g.session.login : t("shop.notSignedIn")}
                </span>
              </span>
              <span className="text-xs text-hint">{t(`catOne.${g.shop.category}` as TextKey)}</span>
            </Link>
            <ul className="divide-y divide-line">
              {g.items.map((it) => (
                <li key={it.id} className={`flex items-center gap-3 py-2.5 ${it.in_stock ? "" : "opacity-50"}`}>
                  <ProductImage photo={it.photo} emoji={it.image} alt={it.name} size="sm" />
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-[15px] font-medium">{it.name}</p>
                    <p className="text-xs text-hint">
                      {[it.size, it.color && t(`color.${it.color}` as TextKey)].filter(Boolean).join(" · ")}
                      {!it.in_stock && t("shop.outOfStock")}
                    </p>
                    <p className="text-sm font-semibold">{money(it.price * it.qty)}</p>
                  </div>
                  {it.qty === 1 ? (
                    <div className="flex items-center gap-1">
                      <button onClick={() => setQty(it.id, 0)} className="rounded-full p-2 text-hint hover:bg-soft" aria-label={t("accounts.delete")}>
                        <Trash2 className="size-4" />
                      </button>
                      <QtyStepper value={1} onChange={(v) => setQty(it.id, v)} />
                    </div>
                  ) : (
                    <QtyStepper value={it.qty} min={0} onChange={(v) => setQty(it.id, v)} />
                  )}
                </li>
              ))}
            </ul>
            <Button block className="mt-2" onClick={() => navigate(`/checkout/${g.shop.id}`)} disabled={g.total === 0}>
              {t("cart.checkout", { price: money(g.total) })}
            </Button>
          </Card>
        ))}
      </div>
    </div>
  );
}
