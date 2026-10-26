const ASTANA_TZ = "Asia/Almaty"; // UTC+5, same as Astana

const numberFmt = new Intl.NumberFormat("ru-RU");

/** 12900 → "12 900 ₸" */
export function money(n: number): string {
  return `${numberFmt.format(Math.round(n))} ₸`;
}

/** Compact numbers for chart axes: 1250 → "1.3K", 3_050_000 → "3.1M". */
export function compact(n: number): string {
  if (Math.abs(n) >= 1_000_000) {
    return `${(n / 1_000_000).toFixed(1).replace(/\.0$/, "")}M`;
  }
  if (Math.abs(n) >= 1_000) {
    return `${(n / 1_000).toFixed(n >= 10_000 ? 0 : 1).replace(/\.0$/, "")}K`;
  }
  return String(Math.round(n));
}

export function number(n: number): string {
  return numberFmt.format(Math.round(n));
}

/** 850 → "850 m", 2300 → "2.3 km" (or "м"/"км" in Russian). */
export function distance(meters: number, lang: string): string {
  const ru = lang === "ru";
  if (meters < 1000) {
    return `${Math.round(meters / 10) * 10} ${ru ? "м" : "m"}`;
  }
  return `${(meters / 1000).toFixed(1).replace(".", ru ? "," : ".")} ${ru ? "км" : "km"}`;
}

function locale(lang: string) {
  return lang === "ru" ? "ru-RU" : "en-GB";
}

/** Unix seconds → "26 Oct, 14:32" in Astana time. */
export function dateTime(unix: number, lang: string): string {
  return new Intl.DateTimeFormat(locale(lang), {
    day: "numeric",
    month: "short",
    hour: "2-digit",
    minute: "2-digit",
    timeZone: ASTANA_TZ,
  }).format(new Date(unix * 1000));
}

/** Unix seconds → "26 Oct" in Astana time. */
export function shortDate(unix: number, lang: string): string {
  return new Intl.DateTimeFormat(locale(lang), { day: "numeric", month: "short", timeZone: ASTANA_TZ }).format(
    new Date(unix * 1000),
  );
}

/** "5 min ago", "2 h ago", or a date. */
export function ago(unix: number, lang: string): string {
  const s = Date.now() / 1000 - unix;
  const ru = lang === "ru";
  if (s < 60) {
    return ru ? "только что" : "just now";
  }
  if (s < 3600) {
    return ru ? `${Math.floor(s / 60)} мин назад` : `${Math.floor(s / 60)} min ago`;
  }
  if (s < 86400) {
    return ru ? `${Math.floor(s / 3600)} ч назад` : `${Math.floor(s / 3600)} h ago`;
  }
  return shortDate(unix, lang);
}

/** Signed percent change, or null when there is nothing to compare with. */
export function change(value: number, prev: number): number | null {
  if (!prev) {
    return null;
  }
  return ((value - prev) / prev) * 100;
}
