import { useEffect, useState } from "react";

function computeDark(): boolean {
  const forced = document.documentElement.getAttribute("data-theme");
  if (forced) {
    return forced === "dark";
  }
  return window.matchMedia("(prefers-color-scheme: dark)").matches;
}

/** True when the app is in dark mode (Telegram theme or the system setting). */
export function useIsDark(): boolean {
  const [dark, setDark] = useState(computeDark);
  useEffect(() => {
    const update = () => setDark(computeDark());
    const media = window.matchMedia("(prefers-color-scheme: dark)");
    media.addEventListener("change", update);
    const observer = new MutationObserver(update);
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ["data-theme"] });
    return () => {
      media.removeEventListener("change", update);
      observer.disconnect();
    };
  }, []);
  return dark;
}
