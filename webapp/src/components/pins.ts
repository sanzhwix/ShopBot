import L from "leaflet";
import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import type { LucideIcon } from "lucide-react";

type PinSize = "sm" | "md" | "lg";

const SIZES: Record<PinSize, { w: number; h: number; icon: number }> = {
  sm: { w: 26, h: 34, icon: 12 },
  md: { w: 36, h: 46, icon: 17 },
  lg: { w: 44, h: 56, icon: 21 },
};

const cache = new Map<string, L.DivIcon>();
const iconIds = new WeakMap<LucideIcon, number>();
let nextIconId = 1;

function iconId(icon: LucideIcon): number {
  let id = iconIds.get(icon);
  if (!id) {
    id = nextIconId++;
    iconIds.set(icon, id);
  }
  return id;
}

/**
 * A water-drop pin: colored teardrop, white circle, category icon inside.
 * iconAnchor is the tip of the drop, so the tip points at the exact place.
 */
export function pinIcon(color: string, icon: LucideIcon, size: PinSize, active = false): L.DivIcon {
  const key = `${color}|${iconId(icon)}|${size}|${active}`;
  const hit = cache.get(key);
  if (hit) {
    return hit;
  }
  const { w, h, icon: iconSize } = SIZES[size];
  const glyph = renderToStaticMarkup(createElement(icon, { size: iconSize, color, strokeWidth: 2.4, "aria-hidden": true }));
  // The drop is drawn in a 36×46 box and scaled; the circle center is at (18, 17.5).
  const cy = (17.5 / 46) * h;
  const html = `
    <div class="pin${active ? " pin--active" : ""}" style="position:relative;width:${w}px;height:${h}px">
      <svg width="${w}" height="${h}" viewBox="0 0 36 46" aria-hidden="true">
        <path d="M18 1C8.6 1 1 8.4 1 17.6 1 30 18 45 18 45s17-15 17-27.4C35 8.4 27.4 1 18 1z" fill="${color}" stroke="#fff" stroke-width="1.5"/>
        <circle cx="18" cy="17.5" r="11.5" fill="#fff"/>
      </svg>
      <div style="position:absolute;left:0;width:${w}px;top:${cy - iconSize / 2}px;height:${iconSize}px;display:flex;justify-content:center">${glyph}</div>
    </div>`;
  const divIcon = L.divIcon({
    className: "pin-host",
    html,
    iconSize: [w, h],
    iconAnchor: [w / 2, h],
  });
  cache.set(key, divIcon);
  return divIcon;
}

export const userIcon = L.divIcon({
  className: "pin-host",
  html: '<div class="user-dot"></div>',
  iconSize: [18, 18],
  iconAnchor: [9, 9],
});
