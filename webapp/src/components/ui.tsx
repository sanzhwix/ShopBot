import { useState, type ButtonHTMLAttributes, type InputHTMLAttributes, type ReactNode } from "react";
import { useNavigate } from "react-router";
import { ChevronLeft, LoaderCircle } from "lucide-react";
import { inTelegram } from "../lib/telegram";
import { useT } from "../lib/i18n";

type Variant = "primary" | "secondary" | "ghost" | "danger";

const variants: Record<Variant, string> = {
  primary: "bg-accent text-accent-ink hover:brightness-105 active:brightness-95",
  secondary: "bg-soft text-ink hover:bg-line",
  ghost: "text-link hover:bg-soft",
  danger: "bg-soft text-danger hover:bg-line",
};

export function Button({
  variant = "primary",
  loading,
  block,
  className = "",
  children,
  disabled,
  ...rest
}: ButtonHTMLAttributes<HTMLButtonElement> & { variant?: Variant; loading?: boolean; block?: boolean }) {
  return (
    <button
      {...rest}
      disabled={disabled || loading}
      className={`inline-flex items-center justify-center gap-2 rounded-xl px-4 py-3 text-[15px] font-semibold transition disabled:opacity-50 ${variants[variant]} ${block ? "w-full" : ""} ${className}`}
    >
      {loading && <LoaderCircle className="size-4 animate-spin" aria-hidden />}
      {children}
    </button>
  );
}

export function Card({ children, className = "" }: { children: ReactNode; className?: string }) {
  return <div className={`rounded-2xl bg-card ${className}`}>{children}</div>;
}

export function Spinner({ className = "" }: { className?: string }) {
  const t = useT();
  return (
    <div className={`flex items-center justify-center py-16 text-hint ${className}`} role="status">
      <LoaderCircle className="size-6 animate-spin" aria-hidden />
      <span className="sr-only">{t("common.loading")}</span>
    </div>
  );
}

export function ErrorState({ onRetry }: { onRetry?: () => void }) {
  const t = useT();
  return (
    <div className="flex flex-col items-center gap-3 py-16 text-center text-hint">
      <p>{t("common.error")}</p>
      {onRetry && (
        <Button variant="secondary" onClick={onRetry}>
          {t("common.retry")}
        </Button>
      )}
    </div>
  );
}

export function EmptyState({ icon, title, hint, action }: { icon: ReactNode; title: string; hint?: string; action?: ReactNode }) {
  return (
    <div className="flex flex-col items-center gap-2 px-8 py-16 text-center">
      <div className="mb-2 text-5xl" aria-hidden>
        {icon}
      </div>
      <p className="text-lg font-semibold">{title}</p>
      {hint && <p className="text-sm text-hint">{hint}</p>}
      {action && <div className="mt-4">{action}</div>}
    </div>
  );
}

/** Page title. In a browser it has its own back arrow; in Telegram the native BackButton is used. */
export function PageHeader({ title, subtitle, back, right }: { title: string; subtitle?: string; back?: boolean; right?: ReactNode }) {
  const navigate = useNavigate();
  const t = useT();
  return (
    <header className="flex items-center gap-2 px-4 pt-4 pb-3">
      {back && !inTelegram && (
        <button onClick={() => navigate(-1)} className="-ml-2 rounded-full p-1.5 text-link hover:bg-soft" aria-label={t("common.back")}>
          <ChevronLeft className="size-6" />
        </button>
      )}
      <div className="min-w-0 flex-1">
        <h1 className="truncate text-[22px] font-bold leading-tight">{title}</h1>
        {subtitle && <p className="truncate text-sm text-hint">{subtitle}</p>}
      </div>
      {right}
    </header>
  );
}

export function Chip({
  active,
  children,
  onClick,
  className = "",
}: {
  active?: boolean;
  children: ReactNode;
  onClick?: () => void;
  className?: string;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      className={`inline-flex shrink-0 items-center gap-1.5 rounded-full px-3.5 py-2 text-sm font-medium whitespace-nowrap transition ${
        active ? "bg-accent text-accent-ink" : "bg-card text-ink shadow-[0_0_0_1px_var(--border)] hover:bg-soft"
      } ${className}`}
    >
      {children}
    </button>
  );
}

export function Segmented<V extends string>({
  value,
  options,
  onChange,
}: {
  value: V;
  options: { value: V; label: string }[];
  onChange: (v: V) => void;
}) {
  return (
    <div className="inline-flex rounded-xl bg-soft p-1" role="radiogroup">
      {options.map((o) => (
        <button
          key={o.value}
          role="radio"
          aria-checked={value === o.value}
          onClick={() => onChange(o.value)}
          className={`rounded-lg px-3.5 py-1.5 text-sm font-medium transition ${value === o.value ? "bg-card shadow-sm" : "text-hint hover:text-ink"}`}
        >
          {o.label}
        </button>
      ))}
    </div>
  );
}

export function Field({ label, hint, ...rest }: InputHTMLAttributes<HTMLInputElement> & { label: string; hint?: string }) {
  return (
    <label className="block">
      <span className="mb-1.5 block text-sm font-medium text-hint">{label}</span>
      <input
        {...rest}
        className="w-full rounded-xl bg-soft px-3.5 py-3 text-[15px] text-ink outline-none placeholder:text-hint focus:shadow-[0_0_0_2px_var(--accent)]"
      />
      {hint && <span className="mt-1 block text-xs text-hint">{hint}</span>}
    </label>
  );
}

type TileSize = "sm" | "thumb" | "md" | "lg";
const TILE: Record<TileSize, string> = {
  sm: "size-12 shrink-0 rounded-xl",
  thumb: "size-16 shrink-0 rounded-xl",
  md: "aspect-[4/3] w-full rounded-2xl",
  lg: "aspect-[4/3] w-full rounded-3xl",
};
const EMOJI_TEXT: Record<TileSize, string> = { sm: "text-2xl", thumb: "text-3xl", md: "text-5xl", lg: "text-8xl" };

/** The product photo; the emoji tile is shown when there is no photo or it fails to load. */
export function ProductImage({ photo, emoji, alt, size = "md", tint }: { photo?: string; emoji: string; alt: string; size?: TileSize; tint?: string }) {
  const [failed, setFailed] = useState("");
  if (!photo || failed === photo) {
    return <EmojiTile emoji={emoji} size={size} tint={tint} />;
  }
  return <img src={photo} alt={alt} loading="lazy" decoding="async" onError={() => setFailed(photo)} className={`bg-soft object-cover ${TILE[size]}`} />;
}

/** A big emoji on a soft tile, used as the product picture when there is no photo. */
export function EmojiTile({ emoji, size = "md", tint }: { emoji: string; size?: TileSize; tint?: string }) {
  return (
    <div
      className={`flex items-center justify-center ${TILE[size]} ${EMOJI_TEXT[size]}`}
      style={{ background: tint ? `color-mix(in srgb, ${tint} 12%, var(--card))` : "var(--soft)" }}
      aria-hidden
    >
      {emoji}
    </div>
  );
}
