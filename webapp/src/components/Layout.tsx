import { useEffect } from "react";
import { NavLink, Outlet, useLocation, useNavigate } from "react-router";
import { Map, Package, Search, ShoppingCart, User } from "lucide-react";
import { useT } from "../lib/i18n";
import { useMe } from "../lib/hooks";
import { haptic, tg } from "../lib/telegram";

const TAB_ROOTS = ["/map", "/search", "/cart", "/orders", "/accounts"];

/** Shows Telegram's native back arrow on inner pages and wires it to the router. */
export function useTelegramBackButton() {
  const location = useLocation();
  const navigate = useNavigate();
  useEffect(() => {
    if (!tg) {
      return;
    }
    const isRoot = TAB_ROOTS.includes(location.pathname) || location.pathname === "/owner";
    if (isRoot) {
      tg.BackButton.hide();
      return;
    }
    const back = () => {
      const idx = (window.history.state as { idx?: number } | null)?.idx ?? 0;
      if (idx > 0) {
        navigate(-1);
      } else {
        navigate(location.pathname.startsWith("/owner") ? "/owner" : "/map");
      }
    };
    tg.BackButton.show();
    tg.BackButton.onClick(back);
    return () => tg?.BackButton.offClick(back);
  }, [location.pathname, navigate]);
}

export function ShopperLayout({ tabs = true }: { tabs?: boolean }) {
  return (
    <div className={`min-h-full ${tabs ? "pb-[calc(var(--tab-h)+var(--safe-bottom))]" : ""}`}>
      <Outlet />
      {tabs && <TabBar />}
    </div>
  );
}

function TabBar() {
  const t = useT();
  const me = useMe();
  const cartCount = me.data?.cart_count ?? 0;
  const items = [
    { to: "/map", label: t("tab.map"), icon: Map },
    { to: "/search", label: t("tab.search"), icon: Search },
    { to: "/cart", label: t("tab.cart"), icon: ShoppingCart, badge: cartCount },
    { to: "/orders", label: t("tab.orders"), icon: Package },
    { to: "/accounts", label: t("tab.profile"), icon: User },
  ];
  return (
    <nav
      className="fixed inset-x-0 bottom-0 z-[1000] border-t border-line bg-card/95 pb-[var(--safe-bottom)] backdrop-blur"
      aria-label="Main"
    >
      <ul className="mx-auto flex h-[var(--tab-h)] max-w-xl">
        {items.map(({ to, label, icon: Icon, badge }) => (
          <li key={to} className="flex-1">
            <NavLink
              to={to}
              onClick={() => haptic.select()}
              className={({ isActive }) =>
                `flex h-full flex-col items-center justify-center gap-0.5 text-[11px] font-medium transition ${isActive ? "text-accent" : "text-hint hover:text-ink"}`
              }
            >
              <span className="relative">
                <Icon className="size-6" aria-hidden />
                {!!badge && (
                  <span className="absolute -top-1.5 -right-2.5 min-w-[18px] rounded-full bg-danger px-1 text-center text-[11px] leading-[18px] font-bold text-white">
                    {badge}
                  </span>
                )}
              </span>
              {label}
            </NavLink>
          </li>
        ))}
      </ul>
    </nav>
  );
}
