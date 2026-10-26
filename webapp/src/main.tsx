import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { App } from "./App";
import { ShopAuthProvider } from "./components/ShopAuth";
import { ToastProvider } from "./components/Toast";
import { LangProvider } from "./lib/i18n";
import { setupTelegram } from "./lib/telegram";
import { track } from "./lib/track";
import "./index.css";

setupTelegram();
if (!/^\/(owner|admin)/.test(location.pathname)) {
  track("app_open");
}

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, refetchOnWindowFocus: false, staleTime: 15_000 },
  },
});

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <LangProvider>
        <ToastProvider>
          <BrowserRouter>
            <ShopAuthProvider>
              <App />
            </ShopAuthProvider>
          </BrowserRouter>
        </ToastProvider>
      </LangProvider>
    </QueryClientProvider>
  </StrictMode>,
);
