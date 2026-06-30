const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:3001"
const SSR_TOKEN = "nimbus-ssr-secret-2026"

async function serverRequest(path: string, options: RequestInit = {}) {
  try {
    const res = await fetch(`${API}${path}`, {
      ...options,
      headers: {
        "Content-Type": "application/json",
        "X-Dashboard-Token": SSR_TOKEN,
        ...options.headers,
      },
      next: { revalidate: 10 },
    })
    if (!res.ok) return null
    return res.json()
  } catch {
    return null
  }
}

export const serverApi = {
  projects: {
    list: () => serverRequest("/api/projects"),
    get: (id: string) => serverRequest(`/api/projects/${id}`),
  },
  deployments: {
    list: (projectId: string) => serverRequest(`/api/projects/${projectId}/deployments`),
  },
  databases: {
    list: () => serverRequest("/api/databases"),
  },
  request: (path: string) => serverRequest(path),
}
