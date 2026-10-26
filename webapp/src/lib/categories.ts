import { Building2, Croissant, Flower2, Pill, Shirt, ShoppingBasket, Smartphone, Store, type LucideIcon } from "lucide-react";
import type { CategoryKey } from "./types";

export interface CategoryMeta {
  key: CategoryKey;
  color: string; // pin color, from the validated chart palette
  icon: LucideIcon;
}

// One hue per category, taken from the dataviz palette so map and charts agree.
export const CATEGORIES: CategoryMeta[] = [
  { key: "mall", color: "#2a78d6", icon: Building2 },
  { key: "clothes", color: "#6250d6", icon: Shirt },
  { key: "bakery", color: "#eb6834", icon: Croissant },
  { key: "grocery", color: "#1baf7a", icon: ShoppingBasket },
  { key: "pharmacy", color: "#e34948", icon: Pill },
  { key: "electronics", color: "#eda100", icon: Smartphone },
  { key: "flowers", color: "#e87ba4", icon: Flower2 },
];

export const OSM_COLOR = "#8e8e93";
export const OsmIcon = Store;

export function categoryMeta(key: string): CategoryMeta {
  return CATEGORIES.find((c) => c.key === key) ?? CATEGORIES[0];
}

export function isCategory(key: string | null): key is CategoryKey {
  return CATEGORIES.some((c) => c.key === key);
}
