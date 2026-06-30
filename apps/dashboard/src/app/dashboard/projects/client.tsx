"use client"

import { useState } from "react"
import { useRouter } from "next/navigation"
import { api } from "@/lib/api"
import { Play, Trash2 } from "lucide-react"

export function ProjectsClient({ projectId, projectName }: { projectId: string; projectName: string }) {
  const router = useRouter()
  const [deploying, setDeploying] = useState(false)

  async function handleDeploy() {
    setDeploying(true)
    try {
      await api.deployments.deploy(projectId)
      setTimeout(() => router.refresh(), 2000)
    } catch (err: any) {
      alert(err.message || "Erro ao iniciar deploy")
    }
    setDeploying(false)
  }

  async function handleDelete() {
    if (!confirm(`Deletar "${projectName}"? Esta acao nao pode ser desfeita.`)) return
    try {
      await api.request(`/api/projects/${projectId}`, { method: "DELETE" })
      router.refresh()
    } catch (err: any) {
      alert(err.message || "Erro ao deletar")
    }
  }

  return (
    <>
      <button onClick={handleDeploy} disabled={deploying} className="btn btn-primary text-xs inline-flex items-center gap-1">
        <Play size={12} />{deploying ? "..." : "Deploy"}
      </button>
      <div className="flex-1" />
      <button onClick={handleDelete} className="btn btn-ghost text-xs text-red-400 hover:text-red-300">
        <Trash2 size={12} />
      </button>
    </>
  )
}
