import { useEffect, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { X } from "lucide-react";
import { useT } from "../lib/i18n";

/** A panel that slides up from the bottom. Closes on backdrop tap or Escape. */
export function BottomSheet({
  open,
  onClose,
  title,
  children,
  footer,
  modal = true,
}: {
  open: boolean;
  onClose: () => void;
  title?: ReactNode;
  children: ReactNode;
  footer?: ReactNode;
  /** false: no dark backdrop, the page behind stays usable (map preview). */
  modal?: boolean;
}) {
  const t = useT();
  useEffect(() => {
    if (!open) {
      return;
    }
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  if (!open) {
    return null;
  }
  return createPortal(
    <div className={`fixed inset-0 z-[1500] flex flex-col justify-end ${modal ? "" : "pointer-events-none"}`}>
      {modal && <div className="animate-fade absolute inset-0 bg-black/40" onClick={onClose} aria-hidden />}
      <div
        role="dialog"
        aria-modal={modal}
        className="animate-sheet pointer-events-auto relative mx-auto flex max-h-[88dvh] w-full max-w-xl flex-col rounded-t-3xl bg-card shadow-[0_-8px_30px_rgba(0,0,0,0.18)]"
      >
        <div className="mx-auto mt-2 h-1 w-10 shrink-0 rounded-full bg-line" aria-hidden />
        <div className="flex shrink-0 items-start gap-2 px-5 pt-3">
          <div className="min-w-0 flex-1">{title}</div>
          <button onClick={onClose} className="-mr-2 rounded-full p-1.5 text-hint hover:bg-soft" aria-label={t("common.close")}>
            <X className="size-5" />
          </button>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto px-5 pt-2 pb-4">{children}</div>
        {footer && <div className="shrink-0 px-5 pt-2 pb-[calc(12px+var(--safe-bottom))]">{footer}</div>}
      </div>
    </div>,
    document.body,
  );
}
