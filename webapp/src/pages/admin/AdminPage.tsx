import { useState, type FormEvent } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { LineChart, Star } from "lucide-react";
import { api, ApiError } from "../../lib/api";
import { ago, money, number } from "../../lib/format";
import { useLang } from "../../lib/i18n";
import type { TextKey } from "../../lib/i18n/en";
import type { FounderStats } from "../../lib/types";
import { Button, Field, Segmented, Spinner } from "../../components/ui";
import { BarList, ChartCard, DayLine, Funnel } from "../owner/charts";

/** Founder metrics: real tester data only, for the HADI cycle and beta testing. */
export default function AdminPage() {
  const { t, lang, setLang } = useLang();
  const q = useQuery({
    queryKey: ["admin", "metrics"],
    queryFn: () => api.get<FounderStats>("/admin/metrics"),
    retry: false,
    refetchInterval: 30_000,
  });

  if (q.isPending) {
    return <Spinner />;
  }
  if (q.error instanceof ApiError && q.error.status === 401) {
    return <AdminLogin />;
  }
  if (!q.data) {
    return <Spinner />;
  }
  const s = q.data;
  const tiles: [string, string][] = [
    [t("admin.users"), number(s.users)],
    [t("admin.new7"), number(s.new_users_7d)],
    [t("admin.activeToday"), number(s.active_today)],
    [t("admin.orders"), number(s.orders)],
    [t("admin.gmv"), money(s.gmv)],
    [t("admin.rating"), s.feedback_avg ? `${s.feedback_avg.toFixed(1)} / 5` : "—"],
  ];

  return (
    <div className="mx-auto max-w-6xl space-y-3 p-3 sm:p-4">
      <header className="flex items-center gap-3 py-2">
        <span className="flex size-11 items-center justify-center rounded-2xl bg-accent text-accent-ink" aria-hidden>
          <LineChart className="size-6" />
        </span>
        <div className="flex-1">
          <h1 className="text-xl font-bold">{t("admin.title")}</h1>
          <p className="text-sm text-hint">{t("admin.subtitle")}</p>
        </div>
        <Segmented
          value={lang}
          onChange={setLang}
          options={[
            { value: "en", label: "EN" },
            { value: "ru", label: "RU" },
          ]}
        />
      </header>

      <div className="grid grid-cols-2 gap-3 md:grid-cols-3 lg:grid-cols-6">
        {tiles.map(([label, value]) => (
          <div key={label} className="rounded-2xl bg-card p-4">
            <p className="text-sm text-hint">{label}</p>
            <p className="mt-1 text-2xl font-semibold">{value}</p>
          </div>
        ))}
      </div>

      <div className="grid gap-3 lg:grid-cols-2">
        <ChartCard title={t("admin.dau")} table={{ head: ["", t("admin.users")], rows: s.dau.map((d) => [new Date(d.day * 1000).toLocaleDateString(), d.value]) }}>
          <DayLine data={s.dau} label={t("admin.users")} />
        </ChartCard>
        <ChartCard title={t("admin.funnel")} table={{ head: ["", ""], rows: s.funnel.map((f) => [t(`step.${f.name}` as TextKey), f.value]) }}>
          <Funnel steps={s.funnel} label={(k) => t(`step.${k}` as TextKey)} />
        </ChartCard>
      </div>

      <div className="grid gap-3 lg:grid-cols-3">
        <ChartCard title={t("admin.searches")}>
          {s.top_searches?.length ? (
            <BarList format={number} items={s.top_searches.map((x) => ({ label: x.name, value: x.value }))} />
          ) : (
            <p className="text-sm text-hint">{t("admin.none")}</p>
          )}
        </ChartCard>
        <ChartCard title={t("admin.invites")}>
          {s.invites?.length ? (
            <BarList format={number} items={s.invites.map((x) => ({ label: x.name, value: x.count, sub: t(`catOne.${x.category}` as TextKey) }))} />
          ) : (
            <p className="text-sm text-hint">{t("admin.none")}</p>
          )}
        </ChartCard>
        <ChartCard title={t("admin.feedback")}>
          {s.feedback?.length ? (
            <ul className="max-h-72 space-y-2 overflow-y-auto">
              {s.feedback.map((f, i) => (
                <li key={i} className="rounded-xl bg-soft p-2.5 text-sm">
                  <p className="flex items-center gap-1">
                    {Array.from({ length: 5 }, (_, n) => (
                      <Star key={n} className={`size-3.5 ${n < f.rating ? "fill-[#eda100] text-[#eda100]" : "text-hint"}`} aria-hidden />
                    ))}
                    <span className="ml-1 text-xs text-hint">
                      {f.name} · {ago(f.created_at, lang)}
                    </span>
                  </p>
                  {f.comment && <p className="mt-1">{f.comment}</p>}
                </li>
              ))}
            </ul>
          ) : (
            <p className="text-sm text-hint">{t("admin.none")}</p>
          )}
        </ChartCard>
      </div>
    </div>
  );
}

function AdminLogin() {
  const { t } = useLang();
  const queryClient = useQueryClient();
  const [password, setPassword] = useState("");
  const [error, setError] = useState(false);

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    try {
      await api.post("/admin/login", { password });
      queryClient.invalidateQueries({ queryKey: ["admin"] });
    } catch {
      setError(true);
    }
  };

  return (
    <div className="mx-auto flex min-h-dvh max-w-sm flex-col justify-center p-4">
      <form onSubmit={submit} className="space-y-3 rounded-3xl bg-card p-6">
        <h1 className="text-xl font-bold">{t("admin.title")}</h1>
        <Field label={t("admin.password")} type="password" value={password} onChange={(e) => setPassword(e.target.value)} autoFocus />
        {error && <p className="text-sm text-danger">{t("owner.login.error")}</p>}
        <Button type="submit" block>
          {t("owner.login.submit")}
        </Button>
      </form>
    </div>
  );
}
