import { lazy, Suspense } from "react";
import { Navigate, Route, Routes } from "react-router";
import { ShopperLayout, useTelegramBackButton } from "./components/Layout";
import { Spinner } from "./components/ui";
import { MapPage } from "./pages/MapPage";
import { MallPage } from "./pages/MallPage";
import { ShopPage } from "./pages/ShopPage";
import { SearchPage } from "./pages/SearchPage";
import { CartPage } from "./pages/CartPage";
import { CheckoutPage } from "./pages/CheckoutPage";
import { PayPage } from "./pages/PayPage";
import { OrderPage, OrdersPage } from "./pages/OrdersPage";
import { AccountsPage } from "./pages/AccountsPage";
import { FeedbackPage } from "./pages/FeedbackPage";

// The owner and founder pages carry the chart library, so they load on demand.
const OwnerApp = lazy(() => import("./pages/owner/OwnerApp"));
const AdminPage = lazy(() => import("./pages/admin/AdminPage"));

export function App() {
  useTelegramBackButton();

  return (
    <Suspense fallback={<Spinner />}>
      <Routes>
        <Route path="/" element={<Navigate to="/map" replace />} />
        <Route element={<ShopperLayout />}>
          <Route path="/map" element={<MapPage />} />
          <Route path="/mall/:id" element={<MallPage />} />
          <Route path="/shop/:id" element={<ShopPage />} />
          <Route path="/search" element={<SearchPage />} />
          <Route path="/cart" element={<CartPage />} />
          <Route path="/orders" element={<OrdersPage />} />
          <Route path="/orders/:id" element={<OrderPage />} />
          <Route path="/accounts" element={<AccountsPage />} />
          <Route path="/feedback" element={<FeedbackPage />} />
        </Route>
        <Route element={<ShopperLayout tabs={false} />}>
          <Route path="/checkout/:shopId" element={<CheckoutPage />} />
          <Route path="/pay/:orderId" element={<PayPage />} />
        </Route>
        <Route path="/owner/*" element={<OwnerApp />} />
        <Route path="/admin" element={<AdminPage />} />
        <Route path="*" element={<Navigate to="/map" replace />} />
      </Routes>
    </Suspense>
  );
}
