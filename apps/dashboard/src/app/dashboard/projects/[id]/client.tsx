"use client"

import { useState, useEffect } from "react"
import { useRouter } from "next/navigation"
import { api } from "@/lib/api"
import { Play, Trash2, Settings, ExternalLink } from "lucide-react"
import { DeployLogViewer } from "@/components/DeployLogViewer"

export function ProjectDetailClient({ projectId, projectName }: { projectId: string; projectName: string }) {
  const router = useRouter()
  const [deploying, setDeploying] = useState(false)
  const [deployId, setDeployId] = useState<string | null>(null)
  const [repoUrl, setRepoUrl] = useState("")
  const [savingRepo, setSavingRepo] = useState(false)
  const [showLogs, setShowLogs] = useState(false)

  useEffect(() => {
    api.projects.get(projectId).then((p: any) => setRepoUrl(p.repoUrl || "")).catch(() => {})
  }, [projectId])

  async function handleDeploy() {
    setDeploying(true)
    setDeployId(null)
    setShowLogs(true)
    try {
      const result = await api.deployments.deploy(projectId)
      setDeployId(result.id)
    } catch (err: any) {
      alert(err.message || "Erro ao iniciar deploy")
      setDeploying(false)
    }
  }

  async function handleSaveRepo() {
    setSavingRepo(true)
    try {
      await api.request(`/api/projects/${projectId}`, { method: "PATCH", body: JSON.stringify({ repoUrl }) })
      router.refresh()
    } catch {}
    setSavingRepo(false)
  }

  async function handleDelete() {
    if (!confirm(`Deletar "${projectName}"? Esta acao nao pode ser desfeita.`)) return
    try {
      await api.request(`/api/projects/${projectId}`, { method: "DELETE" })
      router.push("/dashboard/projects")
    } catch (err: any) {
      alert(err.message || "Erro ao deletar")
    }
  }

  return (
    <div className="space-y-4">
      <div className="card">
        <h2 className="text-lg font-semibold mb-3 flex items-center gap-2"><Settings size={16} /> Acoes</h2>
        <div className="flex gap-2">
          <button onClick={handleDeploy} disabled={deploying && !deployId} className="btn btn-primary text-sm inline-flex items-center gap-1">
            <Play size={14} />{deploying && !deployId ? "Iniciando..." : "Deploy Now"}
          </button>
          <button onClick={handleDelete} className="btn btn-ghost text-sm text-red-400 hover:text-red-300 inline-flex items-center gap-1">
            <Trash2 size={14} />Deletar
          </button>
          {deployId && (
            <a href="/dashboard/deployments/logs" className="btn btn-ghost text-sm text-emerald-400 inline-flex items-center gap-1">
              <ExternalLink size={14} /> Logs
            </a>
          )}
        </div>
      </div>

      {showLogs && deployId && (
        <DeployLogViewer
          deploymentId={deployId}
          onComplete={() => { setDeploying(false); router.refresh() }}
        />
      )}

      <div className="card">
        <h2 className="text-lg font-semibold mb-3">Repositorio Git</h2>
        <div className="flex gap-2">
          <input className="input flex-1" value={repoUrl} onChange={(e) => setRepoUrl(e.target.value)} placeholder="https://github.com/user/repo.git" />
          <button onClick={handleSaveRepo} disabled={savingRepo} className="btn btn-primary text-sm">{savingRepo ? "..." : "Salvar"}</button>
        </div>
      </div>
    </div>
  )
}
