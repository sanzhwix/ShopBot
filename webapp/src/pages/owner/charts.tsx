import { useMemo, useState, type ReactNode } from "react";
import { Area, AreaChart, CartesianGrid, Cell, Pie, PieChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { TrendingDown, TrendingUp } from "lucide-react";
import { change, compact, money, number, shortDate } from "../../lib/format";
import { useLang } from "../../lib/i18n";
import type { KPI, NamedValue } from "../../lib/types";

// Chart rules (dataviz skill): one hue per series in a fixed order, thin marks,
// hairline grid, text in ink colors (never in the series color), a hover readout
// on every chart, and a table view for every chart.

export const SERIES = ["var(--series-1)", "var(--series-2)", "var(--series-3)", "var(--series-4)", "var(--series-5)"];

export function ChartCard({
  title,
  subtitle,
  table,
  children,
  className = "",
}: {
  title: string;
  subtitle?: string;
  table?: { head: string[]; rows: (string | number)[][] };
  children: ReactNode;
  className?: string;
}) {
  const { t } = useLang();
  const [view, setView] = useState<"chart" | "table">("chart");
  return (
    <section className={`rounded-2xl bg-card p-4 ${className}`}>
      <header className="mb-3 flex items-start justify-between gap-3">
        <div>
          <h2 className="font-semibold">{title}</h2>
          {subtitle && <p className="text-xs text-hint">{subtitle}</p>}
        </div>
        {table && (
          <div className="flex shrink-0 rounded-lg bg-soft p-0.5 text-xs" role="radiogroup">
            {(["chart", "table"] as const).map((v) => (
              <button
                key={v}
                role="radio"
                aria-checked={view === v}
                onClick={() => setView(v)}
                className={`rounded-md px-2 py-1 font-medium ${view === v ? "bg-card shadow-sm" : "text-hint"}`}
              >
                {t(v === "chart" ? "chart.chart" : "chart.table")}
              </button>
            ))}
          </div>
        )}
      </header>
      {view === "chart" || !table ? (
        children
      ) : (
        <div className="max-h-80 overflow-auto">
          <table className="tabular w-full text-sm">
            <thead>
              <tr className="text-left text-xs text-hint">
                {table.head.map((h, i) => (
                  <th key={h} className={`pb-2 font-medium ${i > 0 ? "text-right" : ""}`}>
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-line">
              {table.rows.map((row, r) => (
                <tr key={r}>
                  {row.map((cell, i) => (
                    <td key={i} className={`py-1.5 ${i > 0 ? "text-right" : ""}`}>
                      {cell}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

export function KpiTile({ label, kpi, format, days, hero }: { label: string; kpi: KPI; format: (n: number) => string; days: number; hero?: boolean }) {
  const { t } = useLang();
  const delta = change(kpi.value, kpi.prev);
  const up = delta !== null && delta >= 0;
  return (
    <div className="rounded-2xl bg-card p-4">
      <p className="text-sm text-hint">{label}</p>
      <p className={`mt-1 font-semibold tracking-tight ${hero ? "text-[32px] leading-10" : "text-2xl"}`}>{format(kpi.value)}</p>
      {delta !== null && (
        <p className="mt-1 flex items-center gap-1 text-xs">
          <span className={`flex items-center gap-0.5 font-semibold ${up ? "text-[var(--good-text)]" : "text-[var(--bad-text)]"}`}>
            {up ? <TrendingUp className="size-3.5" aria-hidden /> : <TrendingDown className="size-3.5" aria-hidden />}
            {up ? "+" : "−"}
            {Math.abs(delta).toFixed(1)}%
          </span>
          <span className="text-hint">{t("kpi.vsPrev", { n: days })}</span>
        </p>
      )}
    </div>
  );
}

interface Point {
  day: number;
  revenue: number;
  orders: number;
}

export function RevenueChart({ series }: { series: Point[] }) {
  const { lang } = useLang();
  const data = series.map((p) => ({ ...p, label: shortDate(p.day, lang) }));
  return (
    <div className="h-64">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={data} margin={{ top: 8, right: 8, bottom: 0, left: 0 }}>
          <defs>
            <linearGradient id="rev-fill" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="var(--series-1)" stopOpacity={0.14} />
              <stop offset="100%" stopColor="var(--series-1)" stopOpacity={0.02} />
            </linearGradient>
          </defs>
          <CartesianGrid vertical={false} stroke="var(--grid)" strokeWidth={1} />
          <XAxis dataKey="label" tickLine={false} axisLine={{ stroke: "var(--axis)" }} tick={{ fill: "var(--hint)", fontSize: 11 }} minTickGap={28} interval="preserveStartEnd" />
          <YAxis tickFormatter={compact} tickLine={false} axisLine={false} tick={{ fill: "var(--hint)", fontSize: 11 }} width={40} />
          <Tooltip cursor={{ stroke: "var(--axis)", strokeWidth: 1 }} content={<RevenueTip />} />
          <Area
            type="monotone"
            dataKey="revenue"
            stroke="var(--series-1)"
            strokeWidth={2}
            fill="url(#rev-fill)"
            activeDot={{ r: 5, fill: "var(--series-1)", stroke: "var(--card)", strokeWidth: 2 }}
            isAnimationActive={false}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}

function RevenueTip({ active, payload }: { active?: boolean; payload?: { payload: Point & { label: string } }[] }) {
  const { t } = useLang();
  if (!active || !payload?.length) {
    return null;
  }
  const p = payload[0].payload;
  return (
    <div className="rounded-xl bg-card px-3 py-2 text-sm shadow-lg ring-1 ring-line">
      <p className="font-semibold">{money(p.revenue)}</p>
      <p className="text-xs text-hint">
        {p.label} · {t("kpi.orders")}: {p.orders}
      </p>
    </div>
  );
}

/** Line chart for one series with a crosshair, used on the founder page. */
export function DayLine({ data, label }: { data: { day: number; value: number }[]; label: string }) {
  const { lang } = useLang();
  const rows = data.map((d) => ({ ...d, label: shortDate(d.day, lang) }));
  return (
    <div className="h-52">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={rows} margin={{ top: 8, right: 8, bottom: 0, left: 0 }}>
          <CartesianGrid vertical={false} stroke="var(--grid)" />
          <XAxis dataKey="label" tickLine={false} axisLine={{ stroke: "var(--axis)" }} tick={{ fill: "var(--hint)", fontSize: 11 }} minTickGap={24} />
          <YAxis allowDecimals={false} tickLine={false} axisLine={false} tick={{ fill: "var(--hint)", fontSize: 11 }} width={32} />
          <Tooltip
            cursor={{ stroke: "var(--axis)" }}
            content={({ active, payload }) =>
              active && payload?.length ? (
                <div className="rounded-xl bg-card px-3 py-2 text-sm shadow-lg ring-1 ring-line">
                  <p className="font-semibold">{payload[0].value as number}</p>
                  <p className="text-xs text-hint">
                    {(payload[0].payload as { label: string }).label} · {label}
                  </p>
                </div>
              ) : null
            }
          />
          <Area type="monotone" dataKey="value" stroke="var(--series-1)" strokeWidth={2} fill="var(--series-1)" fillOpacity={0.1} isAnimationActive={false} />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}

/** Horizontal bars, one color, value at the tip. */
export function BarList({
  items,
  format,
}: {
  items: { label: string; icon?: string; photo?: string; value: number; sub?: string }[];
  format: (n: number) => string;
}) {
  const [hover, setHover] = useState<number | null>(null);
  const max = Math.max(1, ...items.map((i) => i.value));
  return (
    <ul className="space-y-2.5">
      {items.map((it, i) => (
        <li
          key={it.label}
          tabIndex={0}
          onPointerEnter={() => setHover(i)}
          onPointerLeave={() => setHover(null)}
          onFocus={() => setHover(i)}
          onBlur={() => setHover(null)}
          className={`rounded-lg outline-none ${hover === i ? "bg-soft" : ""}`}
        >
          <div className="flex items-baseline justify-between gap-2 text-sm">
            <span className="flex min-w-0 items-center gap-1.5 truncate">
              {it.photo ? (
                <img src={it.photo} alt="" className="size-5 shrink-0 rounded object-cover" loading="lazy" />
              ) : (
                it.icon && <span aria-hidden>{it.icon}</span>
              )}
              {it.label}
              {it.sub && hover === i && <span className="text-hint"> · {it.sub}</span>}
            </span>
          </div>
          <div className="mt-1 flex items-center gap-2">
            <div className="h-2.5 flex-1">
              <div className="h-full rounded-r-[4px]" style={{ width: `${(it.value / max) * 100}%`, background: "var(--series-1)" }} />
            </div>
            <span className="tabular w-24 shrink-0 text-right text-sm font-semibold">{format(it.value)}</span>
          </div>
        </li>
      ))}
    </ul>
  );
}

/** Part-to-whole donut (≤ 5 segments). Colors follow the category name, not its rank. */
export function Donut({ items, format }: { items: NamedValue[]; format: (n: number) => string }) {
  const { t } = useLang();
  const [hover, setHover] = useState<string | null>(null);
  const total = items.reduce((s, i) => s + i.value, 0) || 1;
  const colorOf = useMemo(() => {
    const names = [...items.map((i) => i.name)].sort();
    return (name: string) => SERIES[names.indexOf(name) % SERIES.length];
  }, [items]);
  const shown = hover ? items.find((i) => i.name === hover) : null;
  return (
    <div className="flex flex-col items-center gap-4 sm:flex-row lg:flex-col">
      <div className="relative size-44 shrink-0">
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <Pie
              data={items}
              dataKey="value"
              nameKey="name"
              innerRadius="62%"
              outerRadius="100%"
              stroke="var(--card)"
              strokeWidth={2}
              isAnimationActive={false}
              onMouseEnter={(d: { name?: string }) => setHover(d.name ?? null)}
              onMouseLeave={() => setHover(null)}
            >
              {items.map((it) => (
                <Cell key={it.name} fill={colorOf(it.name)} opacity={hover && hover !== it.name ? 0.45 : 1} />
              ))}
            </Pie>
          </PieChart>
        </ResponsiveContainer>
        <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center text-center">
          <span className="text-lg font-semibold">{compact(shown ? shown.value : total)}</span>
          <span className="max-w-24 truncate text-xs text-hint">{shown ? shown.name : t("kpi.revenue")}</span>
        </div>
      </div>
      <ul className="w-full space-y-1.5 text-sm">
        {items.map((it) => (
          <li
            key={it.name}
            className={`flex items-center gap-2 rounded-md px-1 ${hover === it.name ? "bg-soft" : ""}`}
            onPointerEnter={() => setHover(it.name)}
            onPointerLeave={() => setHover(null)}
          >
            <span className="size-2.5 shrink-0 rounded-sm" style={{ background: colorOf(it.name) }} aria-hidden />
            <span className="min-w-0 flex-1 truncate">{it.name}</span>
            <span className="tabular text-hint">{Math.round((it.value / total) * 100)}%</span>
            <span className="tabular w-24 text-right font-medium">{format(it.value)}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}

/** Orders by weekday × hour. One hue; stronger = more orders. */
export function Heatmap({ grid }: { grid: number[][] }) {
  const { t } = useLang();
  const [hover, setHover] = useState<{ d: number; h: number } | null>(null);
  const max = Math.max(1, ...grid.flat());
  const hours = Array.from({ length: 24 }, (_, h) => h);
  const total = grid.flat().reduce((a, b) => a + b, 0);
  return (
    <div>
      <p className="mb-2 h-5 text-sm" aria-live="polite">
        {hover ? (
          <>
            <span className="font-semibold">{grid[hover.d][hover.h]}</span>
            <span className="text-hint">
              {" "}
              · {t(`wd.${hover.d}` as "wd.0")} {String(hover.h).padStart(2, "0")}:00–{String(hover.h + 1).padStart(2, "0")}:00
            </span>
          </>
        ) : (
          <span className="text-hint">
            {t("kpi.orders")}: {number(total)}
          </span>
        )}
      </p>
      <div className="overflow-x-auto">
        <div className="grid min-w-[520px] gap-[2px]" style={{ gridTemplateColumns: "32px repeat(24, minmax(0, 1fr))" }}>
          {grid.map((row, d) => (
            <div key={d} className="contents">
              <span className="pr-1 text-[11px] leading-5 text-hint">{t(`wd.${d}` as "wd.0")}</span>
              {row.map((v, h) => (
                <span
                  key={h}
                  tabIndex={0}
                  onPointerEnter={() => setHover({ d, h })}
                  onPointerLeave={() => setHover(null)}
                  onFocus={() => setHover({ d, h })}
                  aria-label={`${t(`wd.${d}` as "wd.0")} ${h}:00 — ${v}`}
                  className={`h-5 rounded-[3px] outline-none ${hover?.d === d && hover.h === h ? "ring-2 ring-ink" : ""}`}
                  style={{ background: v ? `rgba(var(--heat-rgb), ${0.1 + 0.9 * (v / max)})` : "var(--soft)" }}
                />
              ))}
            </div>
          ))}
          <span />
          {hours.map((h) => (
            <span key={h} className="text-center text-[10px] text-hint">
              {h % 3 === 0 ? h : ""}
            </span>
          ))}
        </div>
      </div>
      <div className="mt-2 flex items-center justify-end gap-2 text-[11px] text-hint">
        0
        <span className="h-2 w-24 rounded-full" style={{ background: "linear-gradient(90deg, rgba(var(--heat-rgb),0.1), rgba(var(--heat-rgb),1))" }} aria-hidden />
        {max}
      </div>
    </div>
  );
}

/** Funnel: bar length relative to the first step, and the step-to-step conversion. */
export function Funnel({ steps, label }: { steps: NamedValue[]; label: (key: string) => string }) {
  const { t } = useLang();
  const [hover, setHover] = useState<number | null>(null);
  const first = Math.max(1, steps[0]?.value ?? 1);
  return (
    <ol className="space-y-2">
      {steps.map((s, i) => {
        const prev = i > 0 ? steps[i - 1].value : 0;
        const rate = i > 0 && prev ? (s.value / prev) * 100 : null;
        return (
          <li
            key={s.name}
            tabIndex={0}
            onPointerEnter={() => setHover(i)}
            onPointerLeave={() => setHover(null)}
            onFocus={() => setHover(i)}
            onBlur={() => setHover(null)}
            className={`rounded-lg px-1 py-0.5 outline-none ${hover === i ? "bg-soft" : ""}`}
          >
            <div className="flex items-baseline justify-between gap-2 text-sm">
              <span className="truncate">{label(s.name)}</span>
              <span className="shrink-0 whitespace-nowrap">
                {rate !== null && <span className="tabular mr-2 text-xs text-hint">{`${rate.toFixed(1)}% ${t("chart.step")}`}</span>}
                <span className="tabular font-semibold">{number(s.value)}</span>
              </span>
            </div>
            <div className="mt-1 h-3">
              <div className="h-full rounded-r-[4px]" style={{ width: `${Math.max(1, (s.value / first) * 100)}%`, background: "var(--series-1)" }} />
            </div>
          </li>
        );
      })}
    </ol>
  );
}

/** One 100% bar split into a few parts, with a legend. */
export function ShareBar({ items, label }: { items: NamedValue[]; label: (key: string) => string }) {
  const [hover, setHover] = useState<string | null>(null);
  const total = items.reduce((s, i) => s + i.value, 0) || 1;
  return (
    <div>
      <div className="flex h-4 w-full gap-[2px] overflow-hidden rounded-[4px]">
        {items
          .filter((i) => i.value > 0)
          .map((it) => (
            <span
              key={it.name}
              onPointerEnter={() => setHover(it.name)}
              onPointerLeave={() => setHover(null)}
              className="h-full transition-opacity"
              style={{
                width: `${(it.value / total) * 100}%`,
                background: SERIES[items.indexOf(it)],
                opacity: hover && hover !== it.name ? 0.45 : 1,
              }}
              title={`${label(it.name)}: ${it.value}`}
            />
          ))}
      </div>
      <ul className="mt-3 space-y-1.5 text-sm">
        {items.map((it, i) => (
          <li
            key={it.name}
            className={`flex items-center gap-2 rounded-md px-1 ${hover === it.name ? "bg-soft" : ""}`}
            onPointerEnter={() => setHover(it.name)}
            onPointerLeave={() => setHover(null)}
          >
            <span className="size-2.5 rounded-sm" style={{ background: SERIES[i] }} aria-hidden />
            <span className="flex-1">{label(it.name)}</span>
            <span className="tabular text-hint">{Math.round((it.value / total) * 100)}%</span>
            <span className="tabular w-14 text-right font-medium">{number(it.value)}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}
