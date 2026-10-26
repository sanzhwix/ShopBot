import { useEffect, useState } from "react";
import { Minus, Plus } from "lucide-react";
import { useT } from "../lib/i18n";
import type { TextKey } from "../lib/i18n/en";
import { money } from "../lib/format";
import { track } from "../lib/track";
import type { Product } from "../lib/types";
import { BottomSheet } from "./BottomSheet";
import { useAddToCart } from "./ShopAuth";
import { Button, ProductImage } from "./ui";

const SWATCH: Record<string, string> = {
  black: "#1c1c1e", white: "#ffffff", grey: "#8e8e93", beige: "#d9c7a7", navy: "#1f2f5c", green: "#2f7d4f",
  blue: "#3b6fb6", brown: "#7a4e2d", red: "#c8102e", khaki: "#8a8452", pink: "#f2a7bd",
};

export function ProductCard({ product, tint, onOpen }: { product: Product; tint: string; onOpen: () => void }) {
  const t = useT();
  return (
    <button
      onClick={onOpen}
      className={`flex flex-col gap-2 rounded-2xl bg-card p-2 text-left transition hover:bg-soft ${product.in_stock ? "" : "opacity-50"}`}
    >
      <ProductImage photo={product.photo} emoji={product.image} alt={product.name} tint={tint} />
      <div className="flex min-h-[2.6em] flex-1 px-1 text-[14px] leading-tight font-medium">{product.name}</div>
      <div className="flex items-center justify-between px-1 pb-1">
        <span className="text-[15px] font-bold">{product.in_stock ? money(product.price) : t("shop.outOfStock")}</span>
        {product.in_stock && (
          <span className="flex size-8 items-center justify-center rounded-full bg-accent text-accent-ink" aria-hidden>
            <Plus className="size-4" />
          </span>
        )}
      </div>
    </button>
  );
}

export function ProductSheet({
  product,
  tint,
  source,
  onClose,
}: {
  product: Product | null;
  tint: string;
  source: string;
  onClose: () => void;
}) {
  const t = useT();
  const addToCart = useAddToCart();
  const [size, setSize] = useState("");
  const [color, setColor] = useState("");
  const [qty, setQty] = useState(1);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (product) {
      setSize(product.sizes.length === 1 ? product.sizes[0] : "");
      setColor(product.colors[0] ?? "");
      setQty(1);
      track("product_view", { shop_id: product.shop_id, product_id: product.id, source });
    }
  }, [product, source]);

  if (!product) {
    return null;
  }
  const needSize = product.sizes.length > 0 && !size;

  const add = async () => {
    setBusy(true);
    const ok = await addToCart({ product_id: product.id, size, color, qty, source });
    setBusy(false);
    if (ok) {
      onClose();
    }
  };

  return (
    <BottomSheet
      open
      onClose={onClose}
      footer={
        product.in_stock ? (
          <Button block onClick={add} loading={busy} disabled={needSize}>
            {needSize ? t("product.chooseSize") : t("product.addToCart", { price: money(product.price * qty) })}
          </Button>
        ) : (
          <Button block disabled variant="secondary">
            {t("shop.outOfStock")}
          </Button>
        )
      }
    >
      <ProductImage photo={product.photo} emoji={product.image} alt={product.name} size="lg" tint={tint} />
      {product.photo_credit && <p className="mt-1 truncate text-right text-[10px] text-hint">📷 {product.photo_credit}</p>}
      <h2 className="mt-4 text-xl font-bold">{product.name}</h2>
      <p className="mt-1 text-2xl font-bold">{money(product.price)}</p>
      {product.description && <p className="mt-2 text-[15px] text-hint">{product.description}</p>}
      {product.material && (
        <p className="mt-3 text-sm">
          <span className="text-hint">{t("product.material")}: </span>
          <span className="font-medium">{t(`material.${product.material}` as TextKey)}</span>
        </p>
      )}

      {product.sizes.length > 0 && (
        <fieldset className="mt-4">
          <legend className="mb-2 text-sm font-semibold">{t("product.size")}</legend>
          <div className="flex flex-wrap gap-2">
            {product.sizes.map((s) => (
              <button
                key={s}
                onClick={() => setSize(s)}
                aria-pressed={size === s}
                className={`min-w-12 rounded-xl px-3 py-2 text-sm font-semibold transition ${size === s ? "bg-accent text-accent-ink" : "bg-soft hover:bg-line"}`}
              >
                {s}
              </button>
            ))}
          </div>
        </fieldset>
      )}

      {product.colors.length > 0 && (
        <fieldset className="mt-4">
          <legend className="mb-2 text-sm font-semibold">
            {t("product.color")}: <span className="font-normal text-hint">{color && t(`color.${color}` as TextKey)}</span>
          </legend>
          <div className="flex flex-wrap gap-3">
            {product.colors.map((c) => (
              <button
                key={c}
                onClick={() => setColor(c)}
                aria-pressed={color === c}
                aria-label={t(`color.${c}` as TextKey)}
                className={`size-9 rounded-full shadow-[inset_0_0_0_1px_rgba(0,0,0,0.15)] transition ${color === c ? "ring-2 ring-accent ring-offset-2 ring-offset-[var(--card)]" : ""}`}
                style={{ background: SWATCH[c] ?? c }}
              />
            ))}
          </div>
        </fieldset>
      )}

      {product.in_stock && (
        <div className="mt-5 flex items-center justify-between">
          <span className="text-sm font-semibold">{t("common.pcs", { n: qty })}</span>
          <QtyStepper value={qty} onChange={setQty} />
        </div>
      )}
    </BottomSheet>
  );
}

export function QtyStepper({ value, onChange, min = 1 }: { value: number; onChange: (v: number) => void; min?: number }) {
  return (
    <div className="flex items-center gap-1 rounded-full bg-soft p-1">
      <button onClick={() => onChange(Math.max(min, value - 1))} className="flex size-8 items-center justify-center rounded-full hover:bg-line" aria-label="−">
        <Minus className="size-4" />
      </button>
      <span className="tabular w-7 text-center font-semibold">{value}</span>
      <button onClick={() => onChange(Math.min(20, value + 1))} className="flex size-8 items-center justify-center rounded-full hover:bg-line" aria-label="+">
        <Plus className="size-4" />
      </button>
    </div>
  );
}
