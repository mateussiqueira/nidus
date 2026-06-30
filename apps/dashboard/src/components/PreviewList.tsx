"use client"

import { useEffect, useState } from "react"
import { api } from "@/lib/api"
import { GitBranch, ExternalLink, Trash2, Clock } from "lucide-react"

interface Props {
  projectId: string
  projectSlug: string
}

export default function PreviewList({ projectId, projectSlug }: Props) {
  const [previews, setPreviews] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api.request("/api/projects/" + projectId + "/deployments")
      .then((deps: any[]) => setPreviews((deps || []).filter((d: any) => d.type === "preview").slice(0, 10)))
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [projectId])

  if (loading) return null
  if (previews.length === 0) return null

  async function handleDelete(depId: string) {
    if (!confirm("Deletar preview?")) return
    try {
      await api.request("/api/projects/" + projectId + "/deployments/" + depId, { method: "DELETE" })
      setPreviews(previews.filter(p => p.id !== depId))
    } catch {}
  }

  return (
    <div className="card mb-6">
      <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
        <GitBranch size={16} /> Preview Deployments ({previews.length})
      </h2>
      <div className="space-y-2">
        {previews.map((p: any) => (
          <div key={p.id} className="flex items-center justify-between p-3 rounded-lg bg-zinc-900/30 text-sm">
            <div className="flex items-center gap-3">
              <span className={`badge badge-${p.status === "success" ? "active" : p.status === "failed" ? "failed" : "building"}`}>
                {p.status}
              </span>
              <span className="font-mono text-xs text-zinc-400">{p.branch}</span>
              <span className="text-xs text-zinc-500 flex items-center gap-1">
                <Clock size={10} />
                {p.createdAt ? new Date(p.createdAt).toLocaleDateString("pt-BR") : "—"}
              </span>
            </div>
            <div className="flex items-center gap-2">
              {p.url && (
                <a href={p.url} target="_blank" className="text-cyan-400 hover:underline text-xs flex items-center gap-1">
                  <ExternalLink size={12} /> Preview
                </a>
              )}
              {p.status !== "building" && (
                <button onClick={() => handleDelete(p.id)} className="text-zinc-600 hover:text-red-400">
                  <Trash2 size={14} />
                </button>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
