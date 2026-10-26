import { useQuery } from "@tanstack/react-query";
import { api } from "./api";
import type { User } from "./types";

export interface Me {
  user: User;
  cart_count: number;
  owner_shop_id: number | null;
}

export function useMe() {
  return useQuery({ queryKey: ["me"], queryFn: () => api.get<Me>("/me"), staleTime: 30_000 });
}
