// Small typed wrapper around window.Telegram.WebApp.
// Outside Telegram (a normal browser) every call is a safe no-op.

type HapticStyle = "light" | "medium" | "heavy";

interface TelegramWebApp {
  initData: string;
  initDataUnsafe: { user?: { id: number; first_name: string; language_code?: string } };
  colorScheme: "light" | "dark";
  version: string;
  ready(): void;
  expand(): void;
  disableVerticalSwipes?(): void;
  isVersionAtLeast(v: string): boolean;
  openLink(url: string): void;
  openTelegramLink(url: string): void;
  close(): void;
  onEvent(event: string, cb: () => void): void;
  setHeaderColor?(color: string): void;
  setBackgroundColor?(color: string): void;
  requestContact?(cb: (ok: boolean, res?: { responseUnsafe?: { contact?: { phone_number?: string } } }) => void): void;
  BackButton: { show(): void; hide(): void; onClick(cb: () => void): void; offClick(cb: () => void): void };
  HapticFeedback: {
    impactOccurred(style: HapticStyle): void;
    notificationOccurred(type: "success" | "error" | "warning"): void;
    selectionChanged(): void;
  };
  LocationManager?: {
    isInited: boolean;
    isLocationAvailable: boolean;
    init(cb?: () => void): void;
    getLocation(cb: (data: { latitude: number; longitude: number } | null) => void): void;
  };
}

declare global {
  interface Window {
    Telegram?: { WebApp: TelegramWebApp };
  }
}

const webApp = window.Telegram?.WebApp;

/** True when the page runs inside Telegram with a signed user. */
export const inTelegram = Boolean(webApp && webApp.initData);

export const tg = inTelegram ? webApp! : undefined;

export function initData(): string {
  return tg?.initData ?? "";
}

export function telegramLanguage(): string | undefined {
  return tg?.initDataUnsafe.user?.language_code;
}

/** Call once on start: full height, no swipe-to-close while dragging the map, theme sync. */
export function setupTelegram() {
  if (!tg) {
    return;
  }
  tg.ready();
  tg.expand();
  if (tg.isVersionAtLeast("7.7")) {
    tg.disableVerticalSwipes?.();
  }
  const applyScheme = () => document.documentElement.setAttribute("data-theme", tg.colorScheme);
  applyScheme();
  tg.onEvent("themeChanged", applyScheme);
}

export const haptic = {
  tap: () => tg?.HapticFeedback.impactOccurred("light"),
  success: () => tg?.HapticFeedback.notificationOccurred("success"),
  error: () => tg?.HapticFeedback.notificationOccurred("error"),
  select: () => tg?.HapticFeedback.selectionChanged(),
};

export function openExternal(url: string) {
  if (tg) {
    tg.openLink(url);
  } else {
    window.open(url, "_blank", "noopener");
  }
}

/** Opens a t.me or tg:// link: inside Telegram it stays in the app, in a browser it opens a new tab. */
export function openTelegram(url: string) {
  if (tg && url.startsWith("https://t.me/")) {
    tg.openTelegramLink(url);
  } else {
    window.open(url, "_blank", "noopener");
  }
}

/** Asks Telegram for the user's phone number. Resolves "" if refused or unsupported. */
export function requestPhone(): Promise<string> {
  return new Promise((resolve) => {
    if (!tg?.requestContact || !tg.isVersionAtLeast("6.9")) {
      resolve("");
      return;
    }
    tg.requestContact((ok, res) => resolve(ok ? (res?.responseUnsafe?.contact?.phone_number ?? "") : ""));
  });
}

/**
 * The user's location: Telegram LocationManager first (Bot API 8.0),
 * then the browser Geolocation API. Resolves null if both fail.
 */
export function getLocation(timeoutMs = 6000): Promise<{ lat: number; lng: number } | null> {
  return new Promise((resolve) => {
    let done = false;
    const finish = (v: { lat: number; lng: number } | null) => {
      if (!done) {
        done = true;
        resolve(v);
      }
    };
    setTimeout(() => finish(null), timeoutMs);

    const browser = () => {
      if (!navigator.geolocation) {
        finish(null);
        return;
      }
      navigator.geolocation.getCurrentPosition(
        (p) => finish({ lat: p.coords.latitude, lng: p.coords.longitude }),
        () => finish(null),
        { enableHighAccuracy: true, timeout: timeoutMs - 500, maximumAge: 60_000 },
      );
    };

    const lm = tg?.isVersionAtLeast("8.0") ? tg.LocationManager : undefined;
    if (!lm) {
      browser();
      return;
    }
    const ask = () => {
      if (!lm.isLocationAvailable) {
        browser();
        return;
      }
      lm.getLocation((d) => (d ? finish({ lat: d.latitude, lng: d.longitude }) : browser()));
    };
    if (lm.isInited) {
      ask();
    } else {
      lm.init(ask);
    }
  });
}
