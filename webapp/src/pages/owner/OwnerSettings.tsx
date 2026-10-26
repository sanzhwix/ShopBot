import { useEffect, useState, type FormEvent } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { BellRing, CircleCheck, CircleAlert } from "lucide-react";
import { api } from "../../lib/api";
import { useLang } from "../../lib/i18n";
import type { TextKey } from "../../lib/i18n/en";
import { useToast } from "../../components/Toast";
import { Button, Field, Spinner } from "../../components/ui";
import { useOwnerMe } from "./OwnerApp";

export function OwnerSettings() {
  const { t } = useLang();
  const me = useOwnerMe();
  const toast = useToast();
  const queryClient = useQueryClient();
  const [hours, setHours] = useState("");
  const [kaspi, setKaspi] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (me.data) {
      setHours(me.data.shop.hours);
      setKaspi(me.data.shop.kaspi_pay_url);
    }
  }, [me.data]);

  if (!me.data) {
    return <Spinner />;
  }
  const { shop, telegram_linked } = me.data;

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setBusy(true);
    try {
      await api.patch("/owner/shop", { hours, kaspi_pay_url: kaspi });
      toast(t("owner.set.saved"));
      queryClient.invalidateQueries({ queryKey: ["owner", "me"] });
    } catch (err) {
      toast(err instanceof Error ? err.message : t("common.error"), "error");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="grid gap-3 md:grid-cols-2">
      <section className="rounded-2xl bg-card p-4">
        <h2 className="mb-3 font-semibold">{t("owner.set.shop")}</h2>
        <div className="mb-4 flex items-center gap-3">
          <span className="text-4xl" aria-hidden>
            {shop.logo}
          </span>
          <div>
            <p className="font-semibold">{shop.name}</p>
            <p className="text-sm text-hint">
              {t(`catOne.${shop.category}` as TextKey)} · {shop.address}
            </p>
          </div>
        </div>
        <form onSubmit={submit} className="space-y-3">
          <Field label={t("owner.set.hours")} value={hours} onChange={(e) => setHours(e.target.value)} />
          <Field label={t("owner.set.kaspi")} placeholder="https://pay.kaspi.kz/pay/…" value={kaspi} onChange={(e) => setKaspi(e.target.value)} hint={t("owner.set.kaspiHint")} />
          <Button type="submit" loading={busy}>
            {t("common.save")}
          </Button>
        </form>
      </section>
      <section className="rounded-2xl bg-card p-4">
        <h2 className="mb-3 flex items-center gap-2 font-semibold">
          <BellRing className="size-5" aria-hidden />
          {t("owner.set.telegram")}
        </h2>
        <p className="flex items-start gap-2 text-sm">
          {telegram_linked ? (
            <CircleCheck className="size-5 shrink-0 text-[#0ca30c]" aria-hidden />
          ) : (
            <CircleAlert className="size-5 shrink-0 text-[#fab219]" aria-hidden />
          )}
          {telegram_linked ? t("owner.set.linked") : t("owner.set.notLinked")}
        </p>
      </section>
    </div>
  );
}
