import { api } from "./api";

type EventType = "app_open" | "map_view" | "pin_tap" | "shop_view" | "product_view";

/** Sends a usage event for the analytics. Failures are ignored: tracking must never break the app. */
export function track(type: EventType, data: { shop_id?: number; product_id?: number; source?: string; meta?: string } = {}) {
  api.post("/events", { type, ...data }).catch(() => {});
}

export type Source = "map" | "search" | "direct";

export function sourceParam(value: string | null): Source {
  return value === "map" || value === "search" ? value : "direct";
}
