const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:3001"

function getToken(): string | null {
  if (typeof window === "undefined") return null
  return localStorage.getItem("nidus_token")
}

export function setToken(token: string) {
  localStorage.setItem("nidus_token", token)
}

export function clearToken() {
  localStorage.removeItem("nidus_token")
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
    create: (data: { name: string; slug: string; repoUrl?: string; framework?: string }) =>
      request("/api/projects", { method: "POST", body: JSON.stringify(data) }),
  },
  deployments: {
    list: (projectId: string) => request(`/api/projects/${projectId}/deployments`),
    listPreviews: (projectId: string) => request(`/api/projects/${projectId}/previews`),
    deploy: (projectId: string, branch?: string) =>
      request(`/api/projects/${projectId}/deploy${branch ? `?branch=${branch}` : ""}`, { method: "POST" }),
    metrics: (projectId: string, branch?: string) =>
      request(`/api/projects/${projectId}/metrics${branch ? `?branch=${branch}` : ""}`),
    rollback: (projectId: string, deploymentId: string) =>
      request(`/api/projects/${projectId}/deployments/${deploymentId}/rollback`, { method: "POST" }),
  },
  domains: {
    list: (projectId: string) => request(`/api/projects/${projectId}/domains`),
    add: (projectId: string, domain: string) =>
      request(`/api/projects/${projectId}/domains`, { method: "POST", body: JSON.stringify({ domain }) }),
    delete: (projectId: string, domainId: string) =>
      request(`/api/projects/${projectId}/domains/${domainId}`, { method: "DELETE" }),
    verify: (projectId: string, domainId: string) =>
      request(`/api/projects/${projectId}/domains/${domainId}/verify`, { method: "POST" }),
  },
}
