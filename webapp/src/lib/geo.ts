import { useEffect, useState } from "react";
import { getLocation } from "./telegram";

export interface LatLng {
  lat: number;
  lng: number;
}

/** Baiterek, the center of the Astana demo area. */
export const ASTANA: LatLng = { lat: 51.1283, lng: 71.4305 };

export function distanceM(a: LatLng, b: LatLng): number {
  const R = 6371000;
  const rad = Math.PI / 180;
  const dLat = (b.lat - a.lat) * rad;
  const dLng = (b.lng - a.lng) * rad;
  const h = Math.sin(dLat / 2) ** 2 + Math.cos(a.lat * rad) * Math.cos(b.lat * rad) * Math.sin(dLng / 2) ** 2;
  return 2 * R * Math.asin(Math.sqrt(h));
}

export interface UserLocation {
  /** Where distances are measured from: the real position, or Astana if the user is far away. */
  point: LatLng;
  /** The real position, if we got one. */
  real: LatLng | null;
  /** True when the user is outside Astana and we show the demo area instead. */
  demoArea: boolean;
  ready: boolean;
}

let cached: UserLocation | null = null;

/** Asks for the location once per app session and shares the answer. */
export function useUserLocation(): UserLocation {
  const [loc, setLoc] = useState<UserLocation>(cached ?? { point: ASTANA, real: null, demoArea: false, ready: false });

  useEffect(() => {
    if (cached) {
      return;
    }
    let alive = true;
    getLocation().then((real) => {
      const far = real ? distanceM(real, ASTANA) > 30_000 : false;
      cached = { point: real && !far ? real : ASTANA, real, demoArea: far, ready: true };
      if (alive) {
        setLoc(cached);
      }
    });
    return () => {
      alive = false;
    };
  }, []);

  return loc;
}
