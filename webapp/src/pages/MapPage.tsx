import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useSearchParams } from "react-router";
import { MapContainer, Marker, TileLayer, useMap, useMapEvents } from "react-leaflet";
import { useQuery } from "@tanstack/react-query";
import type { Map as LeafletMap } from "leaflet";
import { ChevronRight, List, LocateFixed, Star, UserPlus } from "lucide-react";
import { api } from "../lib/api";
import { CATEGORIES, categoryMeta, isCategory, OSM_COLOR, OsmIcon } from "../lib/categories";
import { distance } from "../lib/format";
import { distanceM, useUserLocation, type LatLng } from "../lib/geo";
import { useLang } from "../lib/i18n";
import type { TextKey } from "../lib/i18n/en";
import { haptic } from "../lib/telegram";
import { track } from "../lib/track";
import type { Mall, OsmPlace, Shop } from "../lib/types";
import { BottomSheet } from "../components/BottomSheet";
import { pinIcon, userIcon } from "../components/pins";
import { useToast } from "../components/Toast";
import { Button } from "../components/ui";
import { useIsDark } from "../lib/theme";

type Selected = { kind: "shop"; shop: Shop } | { kind: "mall"; mall: Mall } | { kind: "osm"; place: OsmPlace };

// OpenStreetMap's own tiles: free with attribution. Dark mode inverts the tile layer only (see index.css).
const TILES = "https://tile.openstreetmap.org/{z}/{x}/{y}.png";
const ATTRIBUTION = '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors';

const round2 = (n: number) => Math.round(n * 100) / 100;

export function MapPage() {
  const { t, lang } = useLang();
  const toast = useToast();
  const navigate = useNavigate();
  const isDark = useIsDark();
  const [params, setParams] = useSearchParams();
  const rawCat = params.get("cat") ?? "";
  const cat = isCategory(rawCat) ? rawCat : "";
  const loc = useUserLocation();
  const [center, setCenter] = useState<LatLng>(loc.point);
  const [selected, setSelected] = useState<Selected | null>(null);
  const [listOpen, setListOpen] = useState(false);

  const data = useQuery({
    queryKey: ["map", cat, lang],
    queryFn: () => api.get<{ malls: Mall[]; shops: Shop[] }>(`/map${cat ? `?cat=${cat}` : ""}`),
  });
  const osm = useQuery({
    queryKey: ["osm", round2(center.lat), round2(center.lng)],
    queryFn: () => api.get<{ places: OsmPlace[]; error?: string }>(`/osm/nearby?lat=${round2(center.lat)}&lng=${round2(center.lng)}`),
    enabled: loc.ready,
    staleTime: 10 * 60_000,
    // OpenStreetMap's free API is sometimes overloaded; try again a bit later.
    refetchInterval: (query) => (query.state.data?.error ? 20_000 : false),
  });

  useEffect(() => {
    track("map_view", { meta: cat || "all" });
  }, [cat]);

  const located = useRef(false);
  useEffect(() => {
    if (!loc.ready || located.current) {
      return;
    }
    located.current = true;
    setCenter(loc.point);
    if (loc.demoArea) {
      toast(t("map.demoArea"), "info");
    }
  }, [loc.ready, loc.point, loc.demoArea, toast, t]);

  // All malls with partner shops, and only standalone shops on the "All" view
  // (shops inside a mall are reached through the mall pin).
  const malls = data.data?.malls ?? [];
  const shops = useMemo(() => {
    const list = data.data?.shops ?? [];
    return cat === "" ? list.filter((s) => s.mall_id === null) : list;
  }, [data.data, cat]);
  const osmPlaces = useMemo(
    () => (osm.data?.places ?? []).filter((p) => cat === "" || p.category === cat),
    [osm.data, cat],
  );

  const pick = (s: Selected) => {
    haptic.tap();
    setSelected(s);
    if (s.kind === "shop") {
      track("pin_tap", { shop_id: s.shop.id, source: "map" });
    }
  };

  const setCategory = (c: string) => {
    haptic.select();
    setSelected(null);
    setParams(c ? { cat: c } : {}, { replace: true });
  };

  const listItems = useMemo(() => {
    const items: { key: string; name: string; sub: string; dist: number; logo: string; open: () => void }[] = [
      ...malls.map((m) => ({
        key: `m${m.id}`,
        name: m.name,
        sub: t("catOne.mall"),
        dist: distanceM(loc.point, m),
        logo: "🏬",
        open: () => navigate(`/mall/${m.id}`),
      })),
      ...(data.data?.shops ?? []).map((s) => ({
        key: `s${s.id}`,
        name: s.name,
        sub: s.mall_name ? `${t(`catOne.${s.category}` as TextKey)} · ${s.mall_name}` : t(`catOne.${s.category}` as TextKey),
        dist: distanceM(loc.point, s),
        logo: s.logo,
        open: () => navigate(`/shop/${s.id}?from=map`),
      })),
    ];
    return items.sort((a, b) => a.dist - b.dist);
  }, [malls, data.data, loc.point, navigate, t]);

  return (
    <div className="fixed inset-x-0 top-0 bottom-[calc(var(--tab-h)+var(--safe-bottom))]">
      <MapContainer
        center={[loc.point.lat, loc.point.lng]}
        zoom={14}
        zoomControl={false}
        className={`size-full ${isDark ? "dark-tiles" : ""}`}
        attributionControl
      >
        <TileLayer url={TILES} attribution={ATTRIBUTION} maxZoom={19} />
        <MapEvents onMove={setCenter} onTap={() => setSelected(null)} />
        <FlyTo point={loc.ready ? loc.point : null} />
        {loc.real && !loc.demoArea && <Marker position={[loc.real.lat, loc.real.lng]} icon={userIcon} interactive={false} />}

        {osmPlaces.map((p) => (
          <Marker
            key={p.id}
            position={[p.lat, p.lng]}
            icon={pinIcon(OSM_COLOR, OsmIcon, "sm", selected?.kind === "osm" && selected.place.id === p.id)}
            eventHandlers={{ click: () => pick({ kind: "osm", place: p }) }}
            zIndexOffset={-100}
            opacity={selected?.kind === "osm" && selected.place.id === p.id ? 1 : 0.75}
          />
        ))}
        {malls.map((m) => (
          <Marker
            key={`m${m.id}`}
            position={[m.lat, m.lng]}
            icon={pinIcon(categoryMeta("mall").color, categoryMeta("mall").icon, "lg", selected?.kind === "mall" && selected.mall.id === m.id)}
            eventHandlers={{ click: () => pick({ kind: "mall", mall: m }) }}
            zIndexOffset={selected?.kind === "mall" && selected.mall.id === m.id ? 1000 : 200}
          />
        ))}
        {shops.map((s) => {
          const meta = categoryMeta(s.category);
          const active = selected?.kind === "shop" && selected.shop.id === s.id;
          return (
            <Marker
              key={`s${s.id}`}
              position={[s.lat, s.lng]}
              icon={pinIcon(meta.color, meta.icon, "md", active)}
              eventHandlers={{ click: () => pick({ kind: "shop", shop: s }) }}
              zIndexOffset={active ? 1000 : 300}
            />
          );
        })}
      </MapContainer>

      {/* Category filters */}
      <div className="no-scrollbar absolute inset-x-0 top-0 z-[500] flex gap-2 overflow-x-auto px-3 pt-3 pb-2">
        <FilterChip active={cat === ""} onClick={() => setCategory("")} label={t("cat.all")} />
        {CATEGORIES.map((c) => (
          <FilterChip key={c.key} active={cat === c.key} onClick={() => setCategory(c.key)} label={t(`cat.${c.key}` as TextKey)} color={c.color} Icon={c.icon} />
        ))}
      </div>

      {/* OpenStreetMap note */}
      {osmPlaces.length > 0 && !selected && (
        <div className="pointer-events-none absolute top-[60px] left-3 z-[500] flex items-center gap-2 rounded-full bg-card/95 px-3 py-1.5 text-xs text-hint shadow">
          <span className="size-2.5 rounded-full" style={{ background: OSM_COLOR }} aria-hidden />
          {t("map.osmCount", { n: osmPlaces.length })}
        </div>
      )}

      {/* Map buttons */}
      <div className="absolute right-3 bottom-4 z-[500] flex flex-col gap-2" style={{ bottom: selected ? 200 : 16 }}>
        <RoundButton label={t("map.list")} onClick={() => setListOpen(true)}>
          <List className="size-5" />
        </RoundButton>
        <LocateButton point={loc.point} />
      </div>

      {selected && (
        <div className="absolute inset-x-3 bottom-3 z-[600] mx-auto max-w-xl">
          <Preview selected={selected} from={loc.point} allShops={data.data?.shops ?? []} onClose={() => setSelected(null)} />
        </div>
      )}

      <BottomSheet open={listOpen} onClose={() => setListOpen(false)} title={<h2 className="text-lg font-bold">{t("map.nearest")}</h2>}>
        <ul className="-mx-2">
          {listItems.map((it) => (
            <li key={it.key}>
              <button onClick={it.open} className="flex w-full items-center gap-3 rounded-xl p-2 text-left hover:bg-soft">
                <span className="flex size-11 items-center justify-center rounded-xl bg-soft text-2xl" aria-hidden>
                  {it.logo}
                </span>
                <span className="min-w-0 flex-1">
                  <span className="block truncate font-semibold">{it.name}</span>
                  <span className="block truncate text-sm text-hint">{it.sub}</span>
                </span>
                <span className="tabular text-sm text-hint">{distance(it.dist, lang)}</span>
              </button>
            </li>
          ))}
        </ul>
      </BottomSheet>
    </div>
  );
}

function FilterChip({ active, onClick, label, color, Icon }: { active: boolean; onClick: () => void; label: string; color?: string; Icon?: typeof OsmIcon }) {
  return (
    <button
      onClick={onClick}
      aria-pressed={active}
      className={`flex shrink-0 items-center gap-1.5 rounded-full px-3.5 py-2 text-sm font-semibold whitespace-nowrap shadow-md transition ${
        active ? "bg-accent text-accent-ink" : "bg-card text-ink"
      }`}
    >
      {Icon && <Icon className="size-4" style={{ color: active ? undefined : color }} aria-hidden />}
      {label}
    </button>
  );
}

function RoundButton({ label, onClick, children }: { label: string; onClick: () => void; children: React.ReactNode }) {
  return (
    <button onClick={onClick} aria-label={label} title={label} className="flex size-11 items-center justify-center rounded-full bg-card text-ink shadow-md hover:bg-soft">
      {children}
    </button>
  );
}

// The buttons sit outside <MapContainer>, so they reach the map through this handle.
const mapHandle: { current: LeafletMap | null } = { current: null };

function LocateButton({ point }: { point: LatLng }) {
  const { t } = useLang();
  return (
    <RoundButton label={t("map.nearest")} onClick={() => mapHandle.current?.flyTo([point.lat, point.lng], 15)}>
      <LocateFixed className="size-5" />
    </RoundButton>
  );
}

function MapEvents({ onMove, onTap }: { onMove: (c: LatLng) => void; onTap: () => void }) {
  const map = useMapEvents({
    moveend: () => {
      const c = map.getCenter();
      onMove({ lat: c.lat, lng: c.lng });
    },
    click: onTap,
  });
  useEffect(() => {
    mapHandle.current = map;
    return () => {
      mapHandle.current = null;
    };
  }, [map]);
  return null;
}

function FlyTo({ point }: { point: LatLng | null }) {
  const map = useMap();
  const done = useRef(false);
  useEffect(() => {
    if (point && !done.current) {
      done.current = true;
      map.setView([point.lat, point.lng], 14);
    }
  }, [point, map]);
  return null;
}

function Preview({ selected, from, allShops, onClose }: { selected: Selected; from: LatLng; allShops: Shop[]; onClose: () => void }) {
  const { t, lang } = useLang();
  const navigate = useNavigate();

  if (selected.kind === "shop") {
    const s = selected.shop;
    const meta = categoryMeta(s.category);
    return (
      <PreviewShell onClose={onClose}>
        <div className="flex gap-3">
          <span className="flex size-14 shrink-0 items-center justify-center rounded-2xl text-3xl" style={{ background: `color-mix(in srgb, ${meta.color} 14%, var(--card))` }} aria-hidden>
            {s.logo}
          </span>
          <div className="min-w-0 flex-1">
            <h2 className="truncate text-lg font-bold">{s.name}</h2>
            <p className="truncate text-sm text-hint">
              {t(`catOne.${s.category}` as TextKey)}
              {s.mall_name && ` · ${s.mall_name}`}
            </p>
            <p className="mt-1 flex items-center gap-3 text-sm">
              <span className="flex items-center gap-1 font-semibold">
                <Star className="size-4 fill-[#eda100] text-[#eda100]" aria-hidden />
                {s.rating.toFixed(1)}
              </span>
              <span className="text-hint">{s.hours}</span>
              <span className="text-hint">{distance(distanceM(from, s), lang)}</span>
            </p>
          </div>
        </div>
        <p className="mt-2 line-clamp-2 text-sm text-hint">{s.description}</p>
        <Button block className="mt-3" onClick={() => navigate(`/shop/${s.id}?from=map`)}>
          {t("map.openCatalogue")}
          <ChevronRight className="size-4" aria-hidden />
        </Button>
      </PreviewShell>
    );
  }

  if (selected.kind === "mall") {
    const m = selected.mall;
    const inside = allShops.filter((s) => s.mall_id === m.id);
    return (
      <PreviewShell onClose={onClose}>
        <div className="flex gap-3">
          <span className="flex size-14 shrink-0 items-center justify-center rounded-2xl bg-[#2a78d6]/15 text-3xl" aria-hidden>
            🏬
          </span>
          <div className="min-w-0 flex-1">
            <h2 className="truncate text-lg font-bold">{m.name}</h2>
            <p className="truncate text-sm text-hint">
              {m.address} · {distance(distanceM(from, m), lang)}
            </p>
            <p className="mt-1 text-sm font-medium">{t("map.shopsInside", { n: m.shop_count })}</p>
          </div>
        </div>
        {inside.length > 0 && (
          <div className="mt-2 flex flex-wrap gap-1.5">
            {inside.map((s) => (
              <span key={s.id} className="rounded-full bg-soft px-2.5 py-1 text-xs font-medium">
                {s.logo} {s.name}
              </span>
            ))}
          </div>
        )}
        <Button block className="mt-3" onClick={() => navigate(`/mall/${m.id}`)}>
          {t("map.openMall")}
          <ChevronRight className="size-4" aria-hidden />
        </Button>
      </PreviewShell>
    );
  }

  return <OsmPreview place={selected.place} from={from} onClose={onClose} />;
}

function OsmPreview({ place, from, onClose }: { place: OsmPlace; from: LatLng; onClose: () => void }) {
  const { t, lang } = useLang();
  const toast = useToast();
  const [invites, setInvites] = useState<number | null>(null);
  const [busy, setBusy] = useState(false);

  const invite = async () => {
    setBusy(true);
    try {
      const res = await api.post<{ invites: number }>("/invites", {
        osm_id: place.id,
        name: place.name,
        category: place.category,
        lat: place.lat,
        lng: place.lng,
      });
      setInvites(res.invites);
      haptic.success();
      toast(t("map.inviteHint"), "info");
    } finally {
      setBusy(false);
    }
  };

  return (
    <PreviewShell onClose={onClose}>
      <div className="flex gap-3">
        <span className="flex size-14 shrink-0 items-center justify-center rounded-2xl bg-soft text-hint" aria-hidden>
          <OsmIcon className="size-7" />
        </span>
        <div className="min-w-0 flex-1">
          <h2 className="truncate text-lg font-bold">{place.name}</h2>
          <p className="truncate text-sm text-hint">
            {t(`catOne.${place.category}` as TextKey)}
            {place.address && ` · ${place.address}`}
          </p>
          <p className="mt-1 text-sm text-hint">
            {t("map.notPartner")} · {distance(distanceM(from, place), lang)}
          </p>
        </div>
      </div>
      {invites === null ? (
        <Button block variant="secondary" className="mt-3" onClick={invite} loading={busy}>
          <UserPlus className="size-4" aria-hidden />
          {t("map.invite")}
        </Button>
      ) : (
        <p className="mt-3 rounded-xl bg-soft p-3 text-center text-sm font-medium">{t("map.invited", { n: invites })}</p>
      )}
      <p className="mt-2 text-center text-[11px] text-hint">© OpenStreetMap contributors</p>
    </PreviewShell>
  );
}

function PreviewShell({ children, onClose }: { children: React.ReactNode; onClose: () => void }) {
  const { t } = useLang();
  return (
    <div className="animate-sheet relative rounded-3xl bg-card p-4 shadow-[0_8px_30px_rgba(0,0,0,0.22)]">
      <button onClick={onClose} className="absolute top-2 right-2 rounded-full px-2 py-1 text-xs text-hint hover:bg-soft" aria-label={t("common.close")}>
        ✕
      </button>
      {children}
    </div>
  );
}
