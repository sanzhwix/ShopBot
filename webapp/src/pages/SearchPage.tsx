import { useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { Search as SearchIcon, SlidersHorizontal, X } from "lucide-react";
import { api } from "../lib/api";
import { CATEGORIES, categoryMeta } from "../lib/categories";
import { distance, money } from "../lib/format";
import { distanceM, useUserLocation } from "../lib/geo";
import { useLang } from "../lib/i18n";
import type { TextKey } from "../lib/i18n/en";
import type { SearchHit } from "../lib/types";
import { BottomSheet } from "../components/BottomSheet";
import { Button, Chip, EmptyState, ProductImage, Spinner } from "../components/ui";

const SIZES = ["S", "M", "L", "XL"];
const MATERIALS = ["cotton", "fleece", "wool", "denim", "polyester", "leather"];
const POPULAR = { en: ["hoodie", "croissant", "jeans", "roses", "vitamin", "samsa"], ru: ["худи", "круассан", "джинсы", "розы", "витамин", "самса"] };
type Sort = "price_asc" | "price_desc" | "distance";

function useDebounced<T>(value: T, ms: number): T {
  const [v, setV] = useState(value);
  useEffect(() => {
    const id = setTimeout(() => setV(value), ms);
    return () => clearTimeout(id);
  }, [value, ms]);
  return v;
}

export function SearchPage() {
  const { t, lang } = useLang();
  const navigate = useNavigate();
  const loc = useUserLocation();
  const [params, setParams] = useSearchParams();
  const [text, setText] = useState(params.get("q") ?? "");
  const [filtersOpen, setFiltersOpen] = useState(false);
  const query = useDebounced(text.trim(), 350);

  const cat = params.get("cat") ?? "";
  const size = params.get("size") ?? "";
  const material = params.get("material") ?? "";
  const min = params.get("min") ?? "";
  const max = params.get("max") ?? "";
  const sort = (params.get("sort") as Sort) || "price_asc";

  useEffect(() => {
    const next = new URLSearchParams(params);
    if (query) {
      next.set("q", query);
    } else {
      next.delete("q");
    }
    if (next.toString() !== params.toString()) {
      setParams(next, { replace: true });
    }
  }, [query, params, setParams]);

  const update = (key: string, value: string) => {
    const next = new URLSearchParams(params);
    if (value) {
      next.set(key, value);
    } else {
      next.delete(key);
    }
    setParams(next, { replace: true });
  };

  const active = Boolean(query || cat || size || material || min || max);
  const apiQuery = new URLSearchParams({ q: query, cat, size, material, min, max }).toString();
  const results = useQuery({
    queryKey: ["search", apiQuery, lang],
    queryFn: () => api.get<{ results: SearchHit[] }>(`/search?${apiQuery}`),
    enabled: active,
    placeholderData: keepPreviousData,
  });

  const hits = useMemo(() => {
    const list = (results.data?.results ?? []).map((h) => ({ ...h, dist: distanceM(loc.point, h.shop) }));
    const order: Record<Sort, (a: (typeof list)[0], b: (typeof list)[0]) => number> = {
      price_asc: (a, b) => a.price - b.price,
      price_desc: (a, b) => b.price - a.price,
      distance: (a, b) => a.dist - b.dist,
    };
    return list.sort(order[sort]);
  }, [results.data, loc.point, sort]);

  const shopCount = new Set(hits.map((h) => h.shop.id)).size;
  const minPrice = hits.length ? Math.min(...hits.map((h) => h.price)) : 0;
  const filterCount = [size, material, min || max].filter(Boolean).length;

  return (
    <div className="mx-auto max-w-xl">
      <header className="px-4 pt-4 pb-2">
        <h1 className="text-[22px] font-bold">{t("search.title")}</h1>
        <p className="text-sm text-hint">{t("search.subtitle")}</p>
      </header>

      <div className="sticky top-0 z-[400] bg-page px-3 pt-1 pb-2">
        <div className="flex gap-2">
          <label className="flex flex-1 items-center gap-2 rounded-xl bg-card px-3 shadow-[0_0_0_1px_var(--border)]">
            <SearchIcon className="size-5 text-hint" aria-hidden />
            <input
              value={text}
              onChange={(e) => setText(e.target.value)}
              placeholder={t("search.placeholder")}
              className="min-w-0 flex-1 bg-transparent py-3 text-[16px] outline-none placeholder:text-hint"
              enterKeyHint="search"
              aria-label={t("search.title")}
            />
            {text && (
              <button onClick={() => setText("")} className="rounded-full p-1 text-hint hover:bg-soft" aria-label={t("common.close")}>
                <X className="size-4" />
              </button>
            )}
          </label>
          <button
            onClick={() => setFiltersOpen(true)}
            className="relative flex items-center justify-center rounded-xl bg-card px-3.5 shadow-[0_0_0_1px_var(--border)] hover:bg-soft"
            aria-label={t("search.filters")}
          >
            <SlidersHorizontal className="size-5" />
            {filterCount > 0 && (
              <span className="absolute -top-1 -right-1 flex size-5 items-center justify-center rounded-full bg-accent text-[11px] font-bold text-accent-ink">
                {filterCount}
              </span>
            )}
          </button>
        </div>
        <div className="no-scrollbar mt-2 flex gap-2 overflow-x-auto">
          <Chip active={cat === ""} onClick={() => update("cat", "")}>
            {t("cat.all")}
          </Chip>
          {CATEGORIES.filter((c) => c.key !== "mall").map((c) => (
            <Chip key={c.key} active={cat === c.key} onClick={() => update("cat", c.key)}>
              <c.icon className="size-4" style={{ color: cat === c.key ? undefined : c.color }} aria-hidden />
              {t(`cat.${c.key}` as TextKey)}
            </Chip>
          ))}
        </div>
      </div>

      {!active && (
        <div className="px-4 pt-4">
          <p className="mb-2 text-sm font-semibold text-hint">{t("search.popular")}</p>
          <div className="flex flex-wrap gap-2">
            {POPULAR[lang].map((w) => (
              <Chip key={w} onClick={() => setText(w)}>
                {w}
              </Chip>
            ))}
          </div>
        </div>
      )}

      {active && results.isPending && <Spinner />}

      {active && results.data && (
        <div className={`px-3 pt-2 transition-opacity ${results.isFetching ? "opacity-60" : ""}`}>
          {hits.length === 0 ? (
            <EmptyState icon="🔍" title={t("search.empty")} />
          ) : (
            <>
              <p className="px-1 pb-2 text-sm text-hint">{t("search.summary", { n: hits.length, shops: shopCount, price: money(minPrice) })}</p>
              <ul className="space-y-2">
                {hits.map((h) => {
                  const meta = categoryMeta(h.shop.category);
                  return (
                    <li key={h.id}>
                      <button
                        onClick={() => navigate(`/shop/${h.shop.id}?from=search&product=${h.id}`)}
                        className="flex w-full items-center gap-3 rounded-2xl bg-card p-2.5 text-left hover:bg-soft"
                      >
                        <ProductImage photo={h.photo} emoji={h.image} alt={h.name} size="thumb" tint={meta.color} />
                        <div className="min-w-0 flex-1">
                          <p className="truncate font-semibold">{h.name}</p>
                          <p className="truncate text-sm text-hint">
                            {h.shop.logo} {h.shop.name} · {distance(h.dist, lang)}
                          </p>
                          {(h.material || h.sizes.length > 0) && (
                            <p className="truncate text-xs text-hint">
                              {[h.material && t(`material.${h.material}` as TextKey), h.sizes.join(" ")].filter(Boolean).join(" · ")}
                            </p>
                          )}
                        </div>
                        <span className="shrink-0 text-[15px] font-bold">{money(h.price)}</span>
                      </button>
                    </li>
                  );
                })}
              </ul>
            </>
          )}
        </div>
      )}

      <BottomSheet
        open={filtersOpen}
        onClose={() => setFiltersOpen(false)}
        title={<h2 className="text-lg font-bold">{t("search.filters")}</h2>}
        footer={
          <div className="flex gap-2">
            <Button
              variant="secondary"
              onClick={() => {
                const next = new URLSearchParams(params);
                ["size", "material", "min", "max", "sort"].forEach((k) => next.delete(k));
                setParams(next, { replace: true });
              }}
            >
              {t("search.reset")}
            </Button>
            <Button block onClick={() => setFiltersOpen(false)}>
              {t("search.apply")}
            </Button>
          </div>
        }
      >
        <FilterGroup title={t("search.price")}>
          <div className="flex items-center gap-2">
            <input
              inputMode="numeric"
              placeholder={t("search.from")}
              value={min}
              onChange={(e) => update("min", e.target.value.replace(/\D/g, ""))}
              className="w-full rounded-xl bg-soft px-3 py-2.5 outline-none"
              aria-label={`${t("search.price")} ${t("search.from")}`}
            />
            <span className="text-hint">–</span>
            <input
              inputMode="numeric"
              placeholder={t("search.to")}
              value={max}
              onChange={(e) => update("max", e.target.value.replace(/\D/g, ""))}
              className="w-full rounded-xl bg-soft px-3 py-2.5 outline-none"
              aria-label={`${t("search.price")} ${t("search.to")}`}
            />
          </div>
        </FilterGroup>
        <FilterGroup title={t("search.size")}>
          <Chip active={!size} onClick={() => update("size", "")}>
            {t("search.any")}
          </Chip>
          {SIZES.map((s) => (
            <Chip key={s} active={size === s} onClick={() => update("size", s)}>
              {s}
            </Chip>
          ))}
        </FilterGroup>
        <FilterGroup title={t("search.material")}>
          <Chip active={!material} onClick={() => update("material", "")}>
            {t("search.any")}
          </Chip>
          {MATERIALS.map((m) => (
            <Chip key={m} active={material === m} onClick={() => update("material", m)}>
              {t(`material.${m}` as TextKey)}
            </Chip>
          ))}
        </FilterGroup>
        <FilterGroup title={t("search.sort")}>
          {(["price_asc", "price_desc", "distance"] as Sort[]).map((s) => (
            <Chip key={s} active={sort === s} onClick={() => update("sort", s)}>
              {t(`sort.${s}` as TextKey)}
            </Chip>
          ))}
        </FilterGroup>
      </BottomSheet>
    </div>
  );
}

function FilterGroup({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <fieldset className="mb-5">
      <legend className="mb-2 text-sm font-semibold">{title}</legend>
      <div className="flex flex-wrap gap-2">{children}</div>
    </fieldset>
  );
}
