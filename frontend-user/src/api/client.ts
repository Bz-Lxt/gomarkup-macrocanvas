const TOKEN_KEY = "mc_token";

export function getToken(): string {
  return localStorage.getItem(TOKEN_KEY) || "";
}
export function setToken(t: string) {
  localStorage.setItem(TOKEN_KEY, t);
}
export function clearToken() {
  localStorage.removeItem(TOKEN_KEY);
}

async function req(path: string, init: RequestInit = {}) {
  const headers = new Headers(init.headers);
  headers.set("Content-Type", "application/json");
  const tok = getToken();
  if (tok) headers.set("Authorization", `Bearer ${tok}`);
  const res = await fetch(path, { ...init, headers });
  const body = await res.json().catch(() => ({ ok: false, error: { code: "MC_INTERNAL", message: "bad json" } }));
  if (!res.ok || !body.ok) {
    const msg = body?.error?.message || `HTTP ${res.status}`;
    const err = new Error(msg) as Error & { code?: string };
    err.code = body?.error?.code;
    throw err;
  }
  return body.data;
}

export const api = {
  login: (username: string, password: string) => req("/api/v1/auth/login", { method: "POST", body: JSON.stringify({ username, password }) }),
  status: () => req("/api/v1/status"),
  macros: () => req("/api/v1/macros"),
  getMacro: (id: string) => req(`/api/v1/macros/${id}`),
  saveMacro: (id: string, m: unknown) => req(`/api/v1/macros/${id}`, { method: "PUT", body: JSON.stringify(m) }),
  createMacro: (m: unknown) => req("/api/v1/macros", { method: "POST", body: JSON.stringify(m) }),
  deleteMacro: (id: string) => req(`/api/v1/macros/${id}`, { method: "DELETE" }),
  validate: (id: string) => req(`/api/v1/macros/${id}/validate`, { method: "POST" }),
  deploy: (id: string) => req(`/api/v1/macros/${id}/deploy`, { method: "POST" }),
  run: (id: string) => req(`/api/v1/macros/${id}/run`, { method: "POST" }),
  emergency: () => req("/api/v1/emergency-stop", { method: "POST" }),
  events: () => req("/api/v1/events"),
  clearEvents: () => req("/api/v1/events/clear", { method: "POST" }),
  settings: () => req("/api/v1/settings"),
  putSettings: (b: unknown) => req("/api/v1/settings", { method: "PUT", body: JSON.stringify(b) }),
  authorize: () => req("/api/v1/capture/authorize", { method: "POST" }),
  benchmark: (strategy: string) => req("/api/v1/benchmark", { method: "POST", body: JSON.stringify({ strategy, target_us: 1000, samples: 60 }) }),
  runs: () => req("/api/v1/runs"),
};

export function eventsWS(): WebSocket {
  const proto = location.protocol === "https:" ? "wss" : "ws";
  return new WebSocket(`${proto}://${location.host}/ws/events?token=${encodeURIComponent(getToken())}`);
}
