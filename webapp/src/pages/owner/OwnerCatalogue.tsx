import { useRef, useState, type FormEvent } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Camera, LoaderCircle, Plus } from "lucide-react";
import { api } from "../../lib/api";
import { shrinkPhoto } from "../../lib/photo";
import { useLang } from "../../lib/i18n";
import type { Category, Product } from "../../lib/types";
import { BottomSheet } from "../../components/BottomSheet";
import { useToast } from "../../components/Toast";
import { Button, Field, ProductImage, Spinner } from "../../components/ui";

interface Catalogue {
  categories: Category[];
  products: Product[];
}

export function OwnerCatalogue() {
  const { t, lang } = useLang();
  const q = useQuery({ queryKey: ["owner", "catalogue", lang], queryFn: () => api.get<Catalogue>("/owner/catalogue") });
  const [adding, setAdding] = useState(false);

  if (q.isPending) {
    return <Spinner />;
  }
  const { categories, products } = q.data!;

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap gap-2">
        <Button onClick={() => setAdding(true)}>
          <Plus className="size-4" aria-hidden />
          {t("owner.cat.add")}
        </Button>
        <AddCategory />
      </div>
      {categories.map((c) => (
        <section key={c.id} className="rounded-2xl bg-card p-3">
          <h2 className="px-1 pb-2 font-semibold">
            {c.name} <span className="text-sm font-normal text-hint">· {products.filter((p) => p.category_id === c.id).length}</span>
          </h2>
          <ul className="divide-y divide-line">
            {products
              .filter((p) => p.category_id === c.id)
              .map((p) => (
                <ProductRow key={p.id} product={p} />
              ))}
          </ul>
        </section>
      ))}
      {adding && <AddProductSheet categories={categories} onClose={() => setAdding(false)} />}
    </div>
  );
}

function ProductRow({ product }: { product: Product }) {
  const { t } = useLang();
  const toast = useToast();
  const queryClient = useQueryClient();
  const [price, setPrice] = useState(String(product.price));

  const save = async (next: { price?: number; in_stock?: boolean }) => {
    const body = { price: next.price ?? product.price, in_stock: next.in_stock ?? product.in_stock };
    if (body.price <= 0) {
      setPrice(String(product.price));
      return;
    }
    await api.patch(`/owner/products/${product.id}`, body);
    toast(t("owner.cat.saved"));
    queryClient.invalidateQueries({ queryKey: ["owner", "catalogue"] });
    queryClient.invalidateQueries({ queryKey: ["shop"] });
  };

  return (
    <li className={`flex items-center gap-3 px-1 py-2 ${product.in_stock ? "" : "opacity-60"}`}>
      <PhotoButton product={product} />
      <span className="min-w-0 flex-1 truncate text-sm font-medium">{product.name}</span>
      <label className="flex items-center gap-1 rounded-lg bg-soft px-2 py-1">
        <input
          inputMode="numeric"
          value={price}
          onChange={(e) => setPrice(e.target.value.replace(/\D/g, ""))}
          onBlur={() => Number(price) !== product.price && save({ price: Number(price) })}
          onKeyDown={(e) => e.key === "Enter" && (e.target as HTMLInputElement).blur()}
          className="tabular w-20 bg-transparent text-right text-sm outline-none"
          aria-label={`${product.name}: ${t("owner.cat.price")}`}
        />
        <span className="text-sm text-hint">₸</span>
      </label>
      <label className="flex cursor-pointer items-center gap-2 text-xs text-hint">
        <input type="checkbox" checked={product.in_stock} onChange={(e) => save({ in_stock: e.target.checked })} className="peer sr-only" />
        <span className="relative h-6 w-10 rounded-full bg-line transition peer-checked:bg-[#0ca30c] after:absolute after:top-0.5 after:left-0.5 after:size-5 after:rounded-full after:bg-white after:shadow after:transition peer-checked:after:translate-x-4" />
        <span className="sr-only">{t("owner.cat.inStock")}</span>
      </label>
    </li>
  );
}

/** The product photo; tapping it uploads a new one (shrunk in the browser first). */
function PhotoButton({ product }: { product: Product }) {
  const { t } = useLang();
  const toast = useToast();
  const queryClient = useQueryClient();
  const input = useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState(false);

  const upload = async (file: File | undefined) => {
    if (!file) {
      return;
    }
    setBusy(true);
    try {
      await api.upload(`/owner/products/${product.id}/photo`, "photo", await shrinkPhoto(file), "photo.jpg");
      toast(t("owner.cat.photoSaved"));
      queryClient.invalidateQueries({ queryKey: ["owner", "catalogue"] });
      queryClient.invalidateQueries({ queryKey: ["shop"] });
    } catch (e) {
      toast(e instanceof Error ? e.message : t("common.error"), "error");
    } finally {
      setBusy(false);
      if (input.current) {
        input.current.value = "";
      }
    }
  };

  return (
    <button
      type="button"
      onClick={() => input.current?.click()}
      className="relative shrink-0 rounded-xl"
      aria-label={`${product.name}: ${t("owner.cat.photoHint")}`}
      title={t("owner.cat.photoHint")}
    >
      <ProductImage photo={product.photo} emoji={product.image} alt="" size="sm" />
      <span className="absolute -right-1 -bottom-1 flex size-5 items-center justify-center rounded-full bg-accent text-accent-ink shadow" aria-hidden>
        {busy ? <LoaderCircle className="size-3 animate-spin" /> : <Camera className="size-3" />}
      </span>
      <input ref={input} type="file" accept="image/*" hidden onChange={(e) => upload(e.target.files?.[0])} />
    </button>
  );
}

function AddCategory() {
  const { t } = useLang();
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [nameEn, setNameEn] = useState("");
  const [nameRu, setNameRu] = useState("");

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    await api.post("/owner/categories", { name_en: nameEn, name_ru: nameRu });
    setOpen(false);
    setNameEn("");
    setNameRu("");
    queryClient.invalidateQueries({ queryKey: ["owner", "catalogue"] });
  };

  return (
    <>
      <Button variant="secondary" onClick={() => setOpen(true)}>
        <Plus className="size-4" aria-hidden />
        {t("owner.cat.addCategory")}
      </Button>
      <BottomSheet open={open} onClose={() => setOpen(false)} title={<h2 className="text-lg font-bold">{t("owner.cat.addCategory")}</h2>}>
        <form onSubmit={submit} className="space-y-3">
          <Field label={t("owner.cat.nameEn")} value={nameEn} onChange={(e) => setNameEn(e.target.value)} required />
          <Field label={t("owner.cat.nameRu")} value={nameRu} onChange={(e) => setNameRu(e.target.value)} />
          <Button type="submit" block>
            {t("common.save")}
          </Button>
        </form>
      </BottomSheet>
    </>
  );
}

function AddProductSheet({ categories, onClose }: { categories: Category[]; onClose: () => void }) {
  const { t } = useLang();
  const toast = useToast();
  const queryClient = useQueryClient();
  const [form, setForm] = useState({ category_id: categories[0]?.id ?? 0, name_en: "", name_ru: "", price: "", image: "📦", sizes: "", material: "" });
  const [photo, setPhoto] = useState<File | null>(null);
  const set = (k: keyof typeof form) => (e: { target: { value: string } }) => setForm({ ...form, [k]: e.target.value });

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    const created = await api.post<{ id: number }>("/owner/products", {
      ...form,
      category_id: Number(form.category_id),
      price: Number(form.price),
      sizes: form.sizes
        .split(",")
        .map((s) => s.trim())
        .filter(Boolean),
      colors: [],
    });
    if (photo) {
      await api.upload(`/owner/products/${created.id}/photo`, "photo", await shrinkPhoto(photo), "photo.jpg");
    }
    toast(t("owner.cat.added"));
    queryClient.invalidateQueries({ queryKey: ["owner", "catalogue"] });
    queryClient.invalidateQueries({ queryKey: ["shop"] });
    onClose();
  };

  return (
    <BottomSheet open onClose={onClose} title={<h2 className="text-lg font-bold">{t("owner.cat.add")}</h2>}>
      <form onSubmit={submit} className="space-y-3">
        <label className="block">
          <span className="mb-1.5 block text-sm font-medium text-hint">{t("owner.cat.category")}</span>
          <select value={form.category_id} onChange={set("category_id")} className="w-full rounded-xl bg-soft px-3 py-3 outline-none">
            {categories.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>
        </label>
        <Field label={t("owner.cat.nameEn")} value={form.name_en} onChange={set("name_en")} required />
        <Field label={t("owner.cat.nameRu")} value={form.name_ru} onChange={set("name_ru")} />
        <div className="grid grid-cols-2 gap-3">
          <Field label={t("owner.cat.price")} inputMode="numeric" value={form.price} onChange={(e) => setForm({ ...form, price: e.target.value.replace(/\D/g, "") })} required />
          <Field label={t("owner.cat.emoji")} value={form.image} onChange={set("image")} maxLength={4} />
        </div>
        <Field label={t("owner.cat.sizes")} placeholder="S, M, L" value={form.sizes} onChange={set("sizes")} />
        <label className="block">
          <span className="mb-1.5 block text-sm font-medium text-hint">{t("owner.cat.photo")}</span>
          <input type="file" accept="image/*" onChange={(e) => setPhoto(e.target.files?.[0] ?? null)} className="block w-full text-sm file:mr-3 file:rounded-lg file:border-0 file:bg-soft file:px-3 file:py-2 file:font-medium" />
        </label>
        <Button type="submit" block>
          {t("common.save")}
        </Button>
      </form>
    </BottomSheet>
  );
}
