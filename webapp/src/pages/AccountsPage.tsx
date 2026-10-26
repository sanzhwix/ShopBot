import { Link } from "react-router";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { BarChart3, ChevronRight, KeyRound, ShieldCheck, Star, Trash2 } from "lucide-react";
import { api } from "../lib/api";
import { ago } from "../lib/format";
import { useMe } from "../lib/hooks";
import { useLang } from "../lib/i18n";
import { haptic } from "../lib/telegram";
import type { VaultEntry } from "../lib/types";
import { useToast } from "../components/Toast";
import { Card, PageHeader, Segmented, Spinner } from "../components/ui";

export function AccountsPage() {
  const { t, lang, setLang } = useLang();
  const me = useMe();
  const toast = useToast();
  const queryClient = useQueryClient();
  const vault = useQuery({ queryKey: ["vault"], queryFn: () => api.get<{ accounts: VaultEntry[] }>("/vault") });

  const remove = async (entry: VaultEntry) => {
    await api.del(`/vault/${entry.id}`);
    haptic.success();
    toast(t("accounts.deleted"), "info");
    queryClient.invalidateQueries({ queryKey: ["vault"] });
    queryClient.invalidateQueries({ queryKey: ["shop"] });
  };

  const user = me.data?.user;
  return (
    <div className="mx-auto max-w-xl space-y-3 pb-6">
      <PageHeader title={t("profile.title")} />

      {user && (
        <Card className="mx-3 flex items-center gap-3 p-3">
          <span className="flex size-12 items-center justify-center rounded-full bg-accent text-lg font-bold text-accent-ink" aria-hidden>
            {user.first_name.slice(0, 1)}
          </span>
          <span className="min-w-0 flex-1">
            <span className="block truncate font-semibold">
              {user.first_name} {user.last_name}
            </span>
            {user.username && <span className="block truncate text-sm text-hint">@{user.username}</span>}
          </span>
          <Segmented
            value={lang}
            onChange={setLang}
            options={[
              { value: "en", label: "EN" },
              { value: "ru", label: "RU" },
            ]}
          />
        </Card>
      )}

      <section className="px-3">
        <h2 className="flex items-center gap-2 px-1 pt-2 pb-1 text-lg font-bold">
          <KeyRound className="size-5" aria-hidden />
          {t("accounts.title")}
        </h2>
        <p className="flex gap-2 px-1 pb-3 text-sm text-hint">
          <ShieldCheck className="mt-0.5 size-4 shrink-0" aria-hidden />
          {t("accounts.hint")}
        </p>
        {vault.isPending ? (
          <Spinner />
        ) : (vault.data?.accounts.length ?? 0) === 0 ? (
          <Card className="p-4 text-sm text-hint">{t("accounts.empty")}</Card>
        ) : (
          <Card className="divide-y divide-line">
            {vault.data!.accounts.map((a) => (
              <div key={a.id} className="flex items-center gap-3 p-3">
                <Link to={`/shop/${a.shop_id}`} className="flex min-w-0 flex-1 items-center gap-3">
                  <span className="flex size-11 items-center justify-center rounded-xl bg-soft text-2xl" aria-hidden>
                    {a.shop_logo}
                  </span>
                  <span className="min-w-0">
                    <span className="block truncate font-semibold">{a.shop_name}</span>
                    <span className="block truncate text-sm">{a.login}</span>
                    <span className="block text-xs text-hint">
                      {a.signed_in ? `● ${t("accounts.signedIn")}` : t("accounts.saved")} ·{" "}
                      {a.last_used_at ? t("accounts.lastUsed", { time: ago(a.last_used_at, lang) }) : t("accounts.never")}
                    </span>
                  </span>
                </Link>
                <button onClick={() => remove(a)} className="rounded-full p-2 text-danger hover:bg-soft" aria-label={`${t("accounts.delete")} ${a.shop_name}`}>
                  <Trash2 className="size-5" />
                </button>
              </div>
            ))}
          </Card>
        )}
      </section>

      <Card className="mx-3 divide-y divide-line">
        <Link to="/feedback" className="flex items-center gap-3 p-3 hover:bg-soft">
          <Star className="size-5 text-[#eda100]" aria-hidden />
          <span className="flex-1 font-medium">{t("profile.feedback")}</span>
          <ChevronRight className="size-5 text-hint" aria-hidden />
        </Link>
        <Link to="/owner" className="flex items-center gap-3 p-3 hover:bg-soft">
          <BarChart3 className="size-5 text-accent" aria-hidden />
          <span className="flex-1">
            <span className="block font-medium">{t("profile.owner")}</span>
            <span className="block text-sm text-hint">{t("profile.ownerHint")}</span>
          </span>
          <ChevronRight className="size-5 text-hint" aria-hidden />
        </Link>
      </Card>
    </div>
  );
}
