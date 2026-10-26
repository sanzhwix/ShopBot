import { useState } from "react";
import { Link } from "react-router";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { api } from "../../lib/api";
import { ago, money, number, shortDate } from "../../lib/format";
import { useLang } from "../../lib/i18n";
import type { TextKey } from "../../lib/i18n/en";
import type { Order, OwnerStats } from "../../lib/types";
import { ErrorState, Segmented, Spinner } from "../../components/ui";
import { StatusPill } from "../OrdersPage";
import { BarList, ChartCard, Donut, Funnel, Heatmap, KpiTile, RevenueChart, ShareBar } from "./charts";

type Range = "7" | "30" | "90";

export function Dashboard() {
  const { t, lang } = useLang();
  const [range, setRange] = useState<Range>("30");
  const stats = useQuery({
    queryKey: ["owner", "analytics", range, lang],
    queryFn: () => api.get<OwnerStats>(`/owner/analytics?range=${range}`),
    placeholderData: keepPreviousData,
  });
  const recent = useQuery({
    queryKey: ["owner", "orders", "recent", lang],
    queryFn: () => api.get<{ orders: Order[] }>("/owner/orders"),
  });

  if (stats.isPending) {
    return <Spinner />;
  }
  if (stats.isError) {
    return <ErrorState onRetry={() => stats.refetch()} />;
  }
  const s = stats.data;
  const days = s.range_days;
  const label = (prefix: string) => (key: string) => t(`${prefix}.${key}` as TextKey);
  const customers = [
    { name: "new", value: s.customers.new ?? 0 },
    { name: "returning", value: s.customers.returning ?? 0 },
  ];

  return (
    <div className={`space-y-3 transition-opacity ${stats.isFetching ? "opacity-60" : ""}`}>
      <div className="flex flex-wrap items-center gap-3">
        <Segmented
          value={range}
          onChange={setRange}
          options={(["7", "30", "90"] as Range[]).map((r) => ({ value: r, label: t(`range.${r}` as TextKey) }))}
        />
      </div>

      <div className="grid grid-cols-2 gap-3 lg:grid-cols-5">
        <div className="col-span-2 lg:col-span-1">
          <KpiTile hero label={t("kpi.revenue")} kpi={s.kpis.revenue} format={money} days={days} />
        </div>
        <KpiTile label={t("kpi.orders")} kpi={s.kpis.orders} format={number} days={days} />
        <KpiTile label={t("kpi.avg")} kpi={s.kpis.avg_order} format={money} days={days} />
        <KpiTile label={t("kpi.new")} kpi={s.kpis.new_customers} format={number} days={days} />
        <KpiTile label={t("kpi.conversion")} kpi={s.kpis.conversion} format={(n) => `${n.toFixed(1)}%`} days={days} />
      </div>

      <div className="grid gap-3 lg:grid-cols-3">
        <ChartCard
          className="lg:col-span-2"
          title={t("chart.revenue")}
          subtitle={t("chart.revenueSub")}
          table={{
            head: ["", t("kpi.revenue"), t("kpi.orders")],
            rows: s.series.map((p) => [shortDate(p.day, lang), money(p.revenue), p.orders]),
          }}
        >
          <RevenueChart series={s.series} />
        </ChartCard>
        <ChartCard
          title={t("chart.top")}
          subtitle={t("chart.topSub")}
          table={{ head: ["", t("common.pcs", { n: "" }).trim(), t("kpi.revenue")], rows: s.top_products.map((p) => [`${p.image} ${p.name}`, p.qty, money(p.revenue)]) }}
        >
          <BarList
            format={money}
            items={s.top_products.slice(0, 6).map((p) => ({ label: p.name, icon: p.image, photo: p.photo, value: p.revenue, sub: t("common.pcs", { n: p.qty }) }))}
          />
        </ChartCard>
      </div>

      <div className="grid gap-3 lg:grid-cols-3">
        <ChartCard
          className="lg:col-span-2"
          title={t("chart.heatmap")}
          subtitle={t("chart.heatmapSub")}
          table={{
            head: ["", ...Array.from({ length: 24 }, (_, h) => String(h))],
            rows: s.heatmap.map((row, d) => [t(`wd.${d}` as TextKey), ...row]),
          }}
        >
          <Heatmap grid={s.heatmap} />
        </ChartCard>
        <ChartCard
          title={t("chart.categories")}
          table={{ head: ["", t("kpi.revenue")], rows: s.categories.map((c) => [c.name, money(c.value)]) }}
        >
          <Donut items={s.categories} format={money} />
        </ChartCard>
      </div>

      <div className="grid gap-3 lg:grid-cols-3">
        <ChartCard
          className="lg:col-span-2"
          title={t("chart.funnel")}
          subtitle={t("chart.funnelSub")}
          table={{ head: ["", ""], rows: s.funnel.map((f) => [t(`funnel.${f.name}` as TextKey), number(f.value)]) }}
        >
          <Funnel steps={s.funnel} label={label("funnel")} />
        </ChartCard>
        <div className="space-y-3">
          <ChartCard title={t("chart.sources")} table={{ head: ["", t("kpi.orders")], rows: s.sources.map((x) => [t(`source.${x.name}` as TextKey), x.value]) }}>
            <ShareBar items={s.sources} label={label("source")} />
          </ChartCard>
          <ChartCard title={t("chart.customers")} table={{ head: ["", ""], rows: customers.map((x) => [t(`customers.${x.name}` as TextKey), x.value]) }}>
            <ShareBar items={customers} label={label("customers")} />
          </ChartCard>
        </div>
      </div>

      <section className="rounded-2xl bg-card p-4">
        <header className="mb-2 flex items-center justify-between">
          <h2 className="font-semibold">{t("chart.recent")}</h2>
          <Link to="/owner/orders" className="text-sm font-semibold text-link">
            {t("owner.orders")} →
          </Link>
        </header>
        <div className="overflow-x-auto">
          <table className="tabular w-full min-w-[560px] text-sm">
            <tbody className="divide-y divide-line">
              {(recent.data?.orders ?? []).slice(0, 8).map((o) => (
                <tr key={o.id}>
                  <td className="py-2 pr-2 font-semibold">#{o.id}</td>
                  <td className="py-2 pr-2 text-hint">{ago(o.created_at, lang)}</td>
                  <td className="max-w-40 truncate py-2 pr-2">
                    {o.customer.first_name} {o.customer.last_name}
                  </td>
                  <td className="max-w-56 truncate py-2 pr-2 text-hint">{o.items.map((i) => `${i.qty}× ${i.name}`).join(", ")}</td>
                  <td className="py-2 pr-2 text-right font-semibold">{money(o.total)}</td>
                  <td className="py-2 text-right">
                    <StatusPill status={o.status} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}
