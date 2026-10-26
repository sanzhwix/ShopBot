import { initData } from "./telegram";

/** An API error keeps the HTTP status, the machine code and the whole body. */
export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    public body: Record<string, unknown>,
  ) {
    super(String(body.error ?? `HTTP ${status}`));
  }
}

let currentLang = "en";

export function setApiLang(lang: string) {
  currentLang = lang;
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = { "X-Lang": currentLang };
  const data = initData();
  if (data) {
    headers["X-Telegram-Init-Data"] = data;
  }
  const isForm = body instanceof FormData;
  if (body !== undefined && !isForm) {
    headers["Content-Type"] = "application/json";
  }
  const res = await fetch(`/api${path}`, {
    method,
    headers,
    body: body === undefined ? undefined : isForm ? body : JSON.stringify(body),
    credentials: "same-origin",
  });
  if (res.status === 204) {
    return undefined as T;
  }
  const json = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new ApiError(res.status, String(json.code ?? ""), json);
  }
  return json as T;
}

export const api = {
  get: <T>(path: string) => request<T>("GET", path),
  post: <T>(path: string, body?: unknown) => request<T>("POST", path, body ?? {}),
  patch: <T>(path: string, body: unknown) => request<T>("PATCH", path, body),
  del: <T>(path: string) => request<T>("DELETE", path),
  /** Sends a file as multipart form data. */
  upload: <T>(path: string, field: string, file: Blob, filename: string) => {
    const form = new FormData();
    form.append(field, file, filename);
    return request<T>("POST", path, form);
  },
};

/** URL for EventSource, which cannot send headers: initData goes in the query. */
export function streamURL(path: string): string {
  const data = initData();
  return data ? `/api${path}?init=${encodeURIComponent(data)}` : `/api${path}`;
}
