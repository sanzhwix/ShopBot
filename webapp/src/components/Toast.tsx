import { createContext, useCallback, useContext, useRef, useState, type ReactNode } from "react";
import { CircleAlert, CircleCheck, Info } from "lucide-react";

type Kind = "success" | "error" | "info";
interface Toast {
  id: number;
  text: string;
  kind: Kind;
}

const ToastContext = createContext<(text: string, kind?: Kind) => void>(() => {});

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);
  const nextId = useRef(1);

  const show = useCallback((text: string, kind: Kind = "success") => {
    const id = nextId.current++;
    setToasts((list) => [...list.slice(-2), { id, text, kind }]);
    setTimeout(() => setToasts((list) => list.filter((t) => t.id !== id)), 3200);
  }, []);

  const icons = { success: CircleCheck, error: CircleAlert, info: Info };
  return (
    <ToastContext.Provider value={show}>
      {children}
      <div className="pointer-events-none fixed inset-x-0 top-3 z-[2000] flex flex-col items-center gap-2" aria-live="polite">
        {toasts.map((t) => {
          const Icon = icons[t.kind];
          return (
            <div
              key={t.id}
              className="animate-toast pointer-events-auto flex max-w-[92vw] items-center gap-2 rounded-2xl bg-[#1c1c1e]/95 px-4 py-3 text-sm font-medium text-white shadow-lg"
            >
              <Icon className={`size-5 shrink-0 ${t.kind === "error" ? "text-[#ff6b6b]" : t.kind === "success" ? "text-[#4cd964]" : "text-[#8ab4f8]"}`} aria-hidden />
              <span>{t.text}</span>
            </div>
          );
        })}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast() {
  return useContext(ToastContext);
}
