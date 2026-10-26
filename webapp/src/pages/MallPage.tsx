import { Link, useParams } from "react-router";
import { useQuery } from "@tanstack/react-query";
import { ChevronRight, MapPin, Star } from "lucide-react";
import { api } from "../lib/api";
import { categoryMeta } from "../lib/categories";
import { useLang } from "../lib/i18n";
import type { TextKey } from "../lib/i18n/en";
import type { Mall, Shop } from "../lib/types";
import { ErrorState, PageHeader, Spinner } from "../components/ui";

export function MallPage() {
  const { id } = useParams();
  const { t, lang } = useLang();
  const q = useQuery({
    queryKey: ["mall", id, lang],
    queryFn: () => api.get<{ mall: Mall; shops: Shop[] }>(`/malls/${id}`),
  });

  if (q.isPending) {
    return <Spinner />;
  }
  if (q.isError) {
    return <ErrorState onRetry={() => q.refetch()} />;
  }
  const { mall, shops } = q.data;

  return (
    <div className="mx-auto max-w-xl">
      <PageHeader back title={`🏬 ${mall.name}`} subtitle={mall.address} />
      <p className="flex items-center gap-1.5 px-4 pb-3 text-sm text-hint">
        <MapPin className="size-4" aria-hidden />
        {t("map.shopsInside", { n: shops.length })}
      </p>
      <ul className="space-y-2 px-3">
        {shops.map((s) => {
          const meta = categoryMeta(s.category);
          return (
            <li key={s.id}>
              <Link to={`/shop/${s.id}?from=map`} className="flex items-center gap-3 rounded-2xl bg-card p-3 hover:bg-soft">
                <span
                  className="flex size-14 shrink-0 items-center justify-center rounded-2xl text-3xl"
                  style={{ background: `color-mix(in srgb, ${meta.color} 14%, var(--card))` }}
                  aria-hidden
                >
                  {s.logo}
                </span>
                <span className="min-w-0 flex-1">
                  <span className="block truncate font-semibold">{s.name}</span>
                  <span className="block truncate text-sm text-hint">
                    {t(`catOne.${s.category}` as TextKey)} · {s.address}
                  </span>
                  <span className="mt-0.5 flex items-center gap-1 text-sm">
                    <Star className="size-3.5 fill-[#eda100] text-[#eda100]" aria-hidden />
                    {s.rating.toFixed(1)} <span className="text-hint">· {s.hours}</span>
                  </span>
                </span>
                <ChevronRight className="size-5 text-hint" aria-hidden />
              </Link>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
