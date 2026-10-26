// Types that mirror the Go API JSON.

export type CategoryKey = "clothes" | "bakery" | "mall" | "grocery" | "pharmacy" | "electronics" | "flowers";

export interface User {
  id: number;
  first_name: string;
  last_name: string;
  username: string;
  lang: "en" | "ru";
  phone: string;
}

export interface Mall {
  id: number;
  slug: string;
  name: string;
  address: string;
  lat: number;
  lng: number;
  shop_count: number;
}

export interface Shop {
  id: number;
  slug: string;
  name: string;
  category: CategoryKey;
  mall_id: number | null;
  mall_name: string;
  lat: number;
  lng: number;
  address: string;
  description: string;
  hours: string;
  rating: number;
  logo: string;
  kaspi_pay_url: string;
}

export interface Category {
  id: number;
  name: string;
  name_en: string;
  name_ru: string;
  sort: number;
}

export interface Product {
  id: number;
  shop_id: number;
  category_id: number;
  name: string;
  name_en: string;
  name_ru: string;
  description: string;
  price: number;
  image: string;
  photo: string;
  photo_credit: string;
  sizes: string[];
  colors: string[];
  material: string;
  in_stock: boolean;
}

export interface ShopSession {
  signed_in: boolean;
  login: string;
  has_saved_account: boolean;
  saved_login: string;
  shop_id: number;
  shop_name: string;
}

export interface OsmPlace {
  id: string;
  name: string;
  category: CategoryKey;
  lat: number;
  lng: number;
  address: string;
  invites: number;
}

export interface CartItem {
  id: number;
  product_id: number;
  shop_id: number;
  name: string;
  image: string;
  photo: string;
  size: string;
  color: string;
  qty: number;
  price: number;
  in_stock: boolean;
}

export interface CartGroup {
  shop: Shop;
  items: CartItem[];
  total: number;
  session: ShopSession;
}

export type OrderStatus = "pending_payment" | "paid" | "accepted" | "ready" | "delivered" | "cancelled";

export interface OrderItem {
  product_id: number;
  name: string;
  image: string;
  photo: string;
  size: string;
  color: string;
  qty: number;
  price: number;
}

export interface Order {
  id: number;
  shop_id: number;
  user_id: number;
  status: OrderStatus;
  total: number;
  delivery_type: "pickup" | "delivery";
  address: string;
  phone: string;
  comment: string;
  source: "map" | "search" | "direct";
  payment_method: string;
  created_at: number;
  paid_at: number | null;
  updated_at: number;
  shop_name: string;
  shop_logo: string;
  shop_category: CategoryKey;
  account_login: string;
  customer: User;
  items: OrderItem[];
}

export interface PaymentRedirect {
  url: string;
  external: boolean;
  mode: "demo" | "link";
}

export interface VaultEntry {
  id: number;
  shop_id: number;
  shop_name: string;
  shop_logo: string;
  category: CategoryKey;
  login: string;
  created_at: number;
  last_used_at: number | null;
  signed_in: boolean;
}

export interface SearchHit extends Product {
  shop: Shop;
}

export interface Owner {
  id: number;
  shop_id: number;
  email: string;
  name: string;
}

export interface KPI {
  value: number;
  prev: number;
}

export interface NamedValue {
  name: string;
  value: number;
}

export interface OwnerStats {
  range_days: number;
  kpis: {
    revenue: KPI;
    orders: KPI;
    avg_order: KPI;
    new_customers: KPI;
    conversion: KPI;
  };
  series: { day: number; revenue: number; orders: number }[];
  statuses: NamedValue[];
  top_products: { name: string; image: string; photo: string; qty: number; revenue: number }[];
  categories: NamedValue[];
  heatmap: number[][];
  funnel: NamedValue[];
  sources: NamedValue[];
  customers: { new?: number; returning?: number };
}

export interface FounderStats {
  users: number;
  new_users_7d: number;
  active_today: number;
  orders: number;
  gmv: number;
  dau: { day: number; value: number }[];
  funnel: NamedValue[];
  top_searches: NamedValue[] | null;
  invites: { name: string; category: string; count: number }[] | null;
  feedback_avg: number;
  feedback: { rating: number; comment: string; name: string; created_at: number }[] | null;
}
