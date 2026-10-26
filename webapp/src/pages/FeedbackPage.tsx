import { useState } from "react";
import { useNavigate } from "react-router";
import { Star } from "lucide-react";
import { api } from "../lib/api";
import { useLang } from "../lib/i18n";
import { haptic } from "../lib/telegram";
import { Button, Card, EmptyState, PageHeader } from "../components/ui";

export function FeedbackPage() {
  const { t } = useLang();
  const navigate = useNavigate();
  const [rating, setRating] = useState(0);
  const [comment, setComment] = useState("");
  const [busy, setBusy] = useState(false);
  const [sent, setSent] = useState(false);

  const send = async () => {
    setBusy(true);
    try {
      await api.post("/feedback", { rating, comment });
      haptic.success();
      setSent(true);
    } finally {
      setBusy(false);
    }
  };

  if (sent) {
    return <EmptyState icon="💛" title={t("feedback.thanks")} action={<Button onClick={() => navigate("/map")}>{t("cart.openMap")}</Button>} />;
  }

  return (
    <div className="mx-auto max-w-xl">
      <PageHeader back title={t("feedback.title")} />
      <Card className="mx-3 space-y-4 p-4">
        <div className="flex justify-center gap-2" role="radiogroup" aria-label={t("feedback.title")}>
          {[1, 2, 3, 4, 5].map((n) => (
            <button
              key={n}
              role="radio"
              aria-checked={rating === n}
              aria-label={String(n)}
              onClick={() => {
                haptic.select();
                setRating(n);
              }}
              className="rounded-full p-1.5 hover:bg-soft"
            >
              <Star className={`size-10 ${n <= rating ? "fill-[#eda100] text-[#eda100]" : "text-hint"}`} />
            </button>
          ))}
        </div>
        <label className="block">
          <span className="mb-1.5 block text-sm font-medium text-hint">{t("feedback.comment")}</span>
          <textarea
            value={comment}
            onChange={(e) => setComment(e.target.value)}
            rows={4}
            className="w-full resize-none rounded-xl bg-soft p-3 text-[15px] outline-none focus:shadow-[0_0_0_2px_var(--accent)]"
          />
        </label>
        <Button block disabled={rating === 0} loading={busy} onClick={send}>
          {t("feedback.send")}
        </Button>
      </Card>
    </div>
  );
}
