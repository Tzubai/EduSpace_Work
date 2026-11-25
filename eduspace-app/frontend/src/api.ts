import { useEffect, useState } from "react";

// The token lives in sessionStorage: it is cleared when the tab closes.
const KEY = "eduspace_token";
export const getToken = () => sessionStorage.getItem(KEY);
export const setToken = (t: string) => sessionStorage.setItem(KEY, t);
export const clearToken = () => sessionStorage.removeItem(KEY);

export async function api<T = any>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch("/api/v1" + path, {
    method,
    headers: { "Content-Type": "application/json", ...(getToken() ? { Authorization: "Bearer " + getToken() } : {}) },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const data = await res.json().catch(() => null);
  if (!res.ok) {
    if (res.status === 401 && getToken()) {
      clearToken();
      location.reload();
    }
    throw new Error(data?.error || "Something went wrong");
  }
  return data as T;
}

// Loads a list or object from the API; pass null to skip loading.
export function useLoad<T = any>(path: string | null) {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState("");
  const [tick, setTick] = useState(0);
  useEffect(() => {
    if (!path) {
      setData(null);
      setError("");
      return;
    }
    let live = true;
    setData(null);
    setError("");
    api<T>("GET", path)
      .then((d) => live && setData(d))
      .catch((e) => live && setError(e.message));
    return () => {
      live = false;
    };
  }, [path, tick]);
  return { data, error, reload: () => setTick((t) => t + 1) };
}

export const naira = (kobo: number) =>
  "₦" + (kobo / 100).toLocaleString("en-NG", { minimumFractionDigits: 0, maximumFractionDigits: 2 });
export const toKobo = (v: string) => {
  const amount = Number(v);
  return Number.isFinite(amount) && amount > 0 ? Math.round(amount * 100) : 0;
};
export const today = () => {
  const now = new Date();
  const year = now.getFullYear();
  const month = String(now.getMonth() + 1).padStart(2, "0");
  const day = String(now.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
};
