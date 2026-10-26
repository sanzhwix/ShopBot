import { useEffect, useMemo, useRef, useState } from "react";
import { Link, useParams, useSearchParams } from "react-router";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Clock, KeyRound, MapPin, ShoppingCart, Star } from "lucide-react";
import { api } from "../lib/api";
import { categoryMeta } from "../lib/categories";
import { money } from "../lib/format";
import { useLang } from "../lib/i18n";
import type { TextKey } from "../lib/i18n/en";
import { sourceParam, track } from "../lib/track";
import type { CartGroup, Category, Product, Shop, ShopSession } from "../lib/types";
import { ProductCard, ProductSheet } from "../components/Product";
import { useShopAuth } from "../components/ShopAuth";
import { useToast } from "../components/Toast";
import { ErrorState, PageHeader, Spinner } from "../components/ui";

interface ShopData {
  shop: Shop;
  categories: Category[];
  products: Product[];
  session: ShopSession;
}

export function ShopPage() {
  const { id } = useParams();
  const { t, lang } = useLang();
  const [params, setParams] = useSearchParams();
  const source = sourceParam(params.get("from"));
  const q = useQuery({ queryKey: ["shop", id, lang], queryFn: () => api.get<ShopData>(`/shops/${id}`) });
  const cart = useQuery({ queryKey: ["cart", lang], queryFn: () => api.get<{ groups: CartGroup[] }>("/cart") });
  const [activeCat, setActiveCat] = useState<number | null>(null);
  const sections = useRef(new Map<number, HTMLElement>());

  useEffect(() => {
    track("shop_view", { shop_id: Number(id), source });
  }, [id, source]);

  const productId = Number(params.get("product")) || null;
  const openProduct = q.data?.products.find((p) => p.id === productId) ?? null;
  const setOpenProduct = (p: Product | null) => {
    const next = new URLSearchParams(params);
    if (p) {
      next.set("product", String(p.id));
    } else {
      next.delete("product");
    }
    setParams(next, { replace: true });
  };

  const byCategory = useMemo(() => {
    const groups = new Map<number, Product[]>();
    for (const p of q.data?.products ?? []) {
      groups.set(p.category_id, [...(groups.get(p.category_id) ?? []), p]);
    }
    return groups;
  }, [q.data]);

  // Highlight the tab of the section at the top of the screen.
  useEffect(() => {
    if (!q.data) {
      return;
    }
    const observer = new IntersectionObserver(
      (entries) => {
        const visible = entries.filter((e) => e.isIntersecting).sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top);
        if (visible[0]) {
          setActiveCat(Number(visible[0].target.getAttribute("data-cat")));
        }
      },
      { rootMargin: "-120px 0px -60% 0px" },
    );
    sections.current.forEach((el) => observer.observe(el));
    return () => observer.disconnect();
  }, [q.data]);

  if (q.isPending) {
    return <Spinner />;
  }
  if (q.isError) {
    return <ErrorState onRetry={() => q.refetch()} />;
  }
  const { shop, categories, session } = q.data;
  const meta = categoryMeta(shop.category);
  const group = cart.data?.groups.find((g) => g.shop.id === shop.id);

  const scrollTo = (catId: number) => {
    setActiveCat(catId);
    const el = sections.current.get(catId);
    if (el) {
      window.scrollTo({ top: el.getBoundingClientRect().top + window.scrollY - 64, behavior: "smooth" });
    }
  };

  return (
    <div className="mx-auto max-w-xl">
      <PageHeader back title={shop.name} />
      <section className="mx-3 rounded-3xl p-4" style={{ background: `color-mix(in srgb, ${meta.color} 13%, var(--card))` }}>
        <div className="flex gap-3">
          <span className="flex size-16 shrink-0 items-center justify-center rounded-2xl bg-card text-4xl shadow-sm" aria-hidden>
            {shop.logo}
          </span>
          <div className="min-w-0 flex-1">
            <p className="text-sm font-medium" style={{ color: meta.color }}>
              {t(`catOne.${shop.category}` as TextKey)}
              {shop.mall_name && ` · ${shop.mall_name}`}
            </p>
            <p className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-sm">
              <span className="flex items-center gap-1 font-semibold">
                <Star className="size-4 fill-[#eda100] text-[#eda100]" aria-hidden />
                {shop.rating.toFixed(1)}
              </span>
              <span className="flex items-center gap-1 text-hint">
                <Clock className="size-4" aria-hidden />
                {shop.hours}
              </span>
            </p>
            <p className="mt-1 flex items-center gap-1 text-sm text-hint">
              <MapPin className="size-4 shrink-0" aria-hidden />
              <span className="truncate">{shop.address}</span>
            </p>
          </div>
        </div>
        <p className="mt-3 text-sm">{shop.description}</p>
        <SessionChip session={session} />
      </section>

      <nav className="no-scrollbar sticky top-0 z-[400] mt-3 flex gap-2 overflow-x-auto bg-page px-3 py-2" aria-label="Categories">
        {categories.map((c) => (
          <button
            key={c.id}
            onClick={() => scrollTo(c.id)}
            className={`shrink-0 rounded-full px-3.5 py-1.5 text-sm font-semibold whitespace-nowrap transition ${
              activeCat === c.id ? "bg-ink text-page" : "bg-card text-ink"
            }`}
          >
            {c.name}
          </button>
        ))}
      </nav>

      {categories.map((c) => {
        const products = byCategory.get(c.id) ?? [];
        if (products.length === 0) {
          return null;
        }
        return (
          <section
            key={c.id}
            data-cat={c.id}
            ref={(el) => {
              if (el) {
                sections.current.set(c.id, el);
              } else {
                sections.current.delete(c.id);
              }
            }}
            className="px-3 pt-3"
          >
            <h2 className="px-1 pb-2 text-lg font-bold">{c.name}</h2>
            <div className="grid grid-cols-2 gap-2.5 sm:grid-cols-3">
              {products.map((p) => (
                <ProductCard key={p.id} product={p} tint={meta.color} onOpen={() => setOpenProduct(p)} />
              ))}
            </div>
          </section>
        );
      })}

      <div className="h-24" />

      {group && group.items.length > 0 && !openProduct && (
        <div className="fixed inset-x-0 bottom-[calc(var(--tab-h)+var(--safe-bottom)+10px)] z-[900] px-3">
          <Link
            to="/cart"
            className="mx-auto flex max-w-xl items-center justify-between rounded-2xl bg-accent px-4 py-3.5 font-semibold text-accent-ink shadow-lg"
          >
            <span className="flex items-center gap-2">
              <ShoppingCart className="size-5" aria-hidden />
              {t("common.pcs", { n: group.items.reduce((n, i) => n + i.qty, 0) })}
            </span>
            <span>{t("shop.goToCart", { price: money(group.total) })}</span>
          </Link>
        </div>
      )}

      <ProductSheet product={openProduct} tint={meta.color} source={source} onClose={() => setOpenProduct(null)} />
    </div>
  );
}

function SessionChip({ session }: { session: ShopSession }) {
  const { t } = useLang();
  const { openLogin } = useShopAuth();
  const queryClient = useQueryClient();
  const toast = useToast();

  if (session.signed_in) {
    const signOut = async () => {
      await api.post(`/shops/${session.shop_id}/logout`);
      queryClient.invalidateQueries({ queryKey: ["shop"] });
      toast(t("shop.signOut"), "info");
    };
    return (
      <div className="mt-3 flex items-center justify-between gap-2 rounded-xl bg-card/80 px-3 py-2 text-sm">
        <span className="flex min-w-0 items-center gap-2">
          <KeyRound className="size-4 shrink-0 text-[var(--good-text)]" aria-hidden />
          <span className="truncate">{t("shop.signedInAs", { login: session.login })}</span>
        </span>
        <button onClick={signOut} className="shrink-0 font-semibold text-link">
          {t("shop.signOut")}
        </button>
      </div>
    );
  }
  return (
    <div className="mt-3 flex items-center justify-between gap-2 rounded-xl bg-card/80 px-3 py-2 text-sm">
      <span className="flex items-center gap-2 text-hint">
        <KeyRound className="size-4" aria-hidden />
        {session.has_saved_account ? `${t("accounts.saved")}: ${session.saved_login}` : t("shop.notSignedIn")}
      </span>
      <button onClick={() => openLogin(session)} className="font-semibold text-link">
        {t("shop.signIn")}
      </button>
    </div>
  );
}
