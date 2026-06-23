const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:3001"

function getToken(): string | null {
  if (typeof window === "undefined") return null
  return localStorage.getItem("canopy_token")
}

export function setToken(token: string) {
  localStorage.setItem("canopy_token", token)
}

export function clearToken() {
  localStorage.removeItem("canopy_token")
}

export function isAuthenticated(): boolean {
  return !!getToken()
}

async function request(path: string, options: RequestInit = {}) {
  const token = getToken()
  const res = await fetch(`${API}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options.headers,
    },
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ message: res.statusText }))
    throw new Error(err.message ?? "Erro desconhecido")
  }
  return res.json()
}

export const api = {
  request,
  auth: {
    login: (email: string, password: string) =>
      request("/api/auth/login", { method: "POST", body: JSON.stringify({ email, password }) }),
    register: (email: string, name: string, password: string) =>
      request("/api/auth/register", { method: "POST", body: JSON.stringify({ email, name, password }) }),
    me: () => request("/api/auth/me"),
  },
  projects: {
    list: () => request("/api/projects"),
    get: (id: string) => request(`/api/projects/${id}`),
    create: (data: { name: string; slug: string; repoUrl?: string }) =>
      request("/api/projects", { method: "POST", body: JSON.stringify(data) }),
  },
  deployments: {
    list: (projectId: string) => request(`/api/projects/${projectId}/deployments`),
    deploy: (projectId: string) => request(`/api/projects/${projectId}/deploy`, { method: "POST" }),
  },
}
