"use client"
export const dynamic = "force-dynamic"

import { useEffect, useState } from "react"
import { useParams, useRouter } from "next/navigation"
import { api } from "@/lib/api"
import {
  ArrowLeft, GitBranch, Clock, Rocket, ExternalLink, Play, Settings,
  Eye, EyeOff, Activity, Cpu, MemoryStick, Timer, Webhook,
} from "lucide-react"
import Link from "next/link"

export default function ProjectPage() {
  const { id } = useParams<{ id: string }>()
  const router = useRouter()
  const [project, setProject] = useState<any>(null)
  const [deployments, setDeployments] = useState<any[]>([])
  const [previews, setPreviews] = useState<any[]>([])
  const [deploying, setDeploying] = useState(false)
  const [envText, setEnvText] = useState("")
  const [showEnv, setShowEnv] = useState(false)
  const [savingEnv, setSavingEnv] = useState(false)
  const [envSaved, setEnvSaved] = useState(false)
  const [metrics, setMetrics] = useState<any>(null)
  const [activeTab, setActiveTab] = useState<"production" | "previews">("production")

  function load() {
    if (!id) return
    api.projects.get(id).then((p: any) => {
      setProject(p)
      setEnvText(p.envVars || "")
    }).catch(() => router.push("/dashboard"))
    api.deployments.list(id).then(setDeployments).catch(() => {})
    api.deployments.listPreviews(id).then(setPreviews).catch(() => {})
    api.deployments.metrics(id).then(setMetrics).catch(() => {})
  }

  useEffect(load, [id, router])

  async function handleDeploy() {
    if (!id) return
    setDeploying(true)
    try {
      await api.deployments.deploy(id)
      load()
    } catch {}
    setDeploying(false)
  }

  async function handleDeployPreview(branch: string) {
    if (!id) return
    try {
      await api.deployments.deploy(id, branch)
      load()
    } catch {}
  }

  async function handleSaveEnv() {
    if (!id) return
    setSavingEnv(true)
    try {
      await api.request(`/api/projects/${id}`, { method: "PATCH", body: JSON.stringify({ envVars: envText }) })
      setEnvSaved(true)
      setTimeout(() => setEnvSaved(false), 2000)
    } catch {}
    setSavingEnv(false)
  }

  if (!project) return null

  const statusBadge = project.status === "ACTIVE" ? "active" : project.status === "FAILED" ? "failed" : "building"

  return (
    <div>
      <Link href="/dashboard" className="inline-flex items-center gap-2 text-sm text-zinc-400 hover:text-white mb-8 transition">
        <ArrowLeft size={16} />
        Voltar
      </Link>

      <div className="flex items-start justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{project.name}</h1>
          <div className="flex items-center gap-3 mt-2">
            <span className={`badge badge-${statusBadge}`}>{project.status.toLowerCase()}</span>
            {project.framework && <span className="text-sm text-zinc-400">{project.framework}</span>}
            {project.repoUrl && (
              <a href={project.repoUrl} target="_blank" className="text-sm text-zinc-400 hover:text-white flex items-center gap-1 transition">
                <GitBranch size={12} />
                {project.repoUrl.split("/").slice(-2).join("/")}
                <ExternalLink size={10} />
              </a>
            )}
          </div>
        </div>
        <button onClick={handleDeploy} disabled={deploying} className="btn btn-primary">
          <Play size={14} />
          {deploying ? "Deploying..." : "Deploy Now"}
        </button>
      </div>

      {metrics && (
        <div className="card mb-6">
          <h2 className="text-lg font-semibold mb-4 flex items-center gap-2"><Activity size={16} /> Container</h2>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="p-3 rounded-lg bg-zinc-900/50">
              <div className="flex items-center gap-2 text-xs text-zinc-500 mb-1"><Activity size={12} /> Status</div>
              <span className={`badge badge-${metrics.running ? "active" : "paused"}`}>{metrics.status}</span>
            </div>
            <div className="p-3 rounded-lg bg-zinc-900/50">
              <div className="flex items-center gap-2 text-xs text-zinc-500 mb-1"><Cpu size={12} /> CPU</div>
              <p className="text-lg font-semibold">{metrics.cpu.toFixed(1)}%</p>
            </div>
            <div className="p-3 rounded-lg bg-zinc-900/50">
              <div className="flex items-center gap-2 text-xs text-zinc-500 mb-1"><MemoryStick size={12} /> Memória</div>
              <p className="text-lg font-semibold">{metrics.memory.percent.toFixed(1)}%</p>
              <p className="text-xs text-zinc-500">{metrics.memory.usage} / {metrics.memory.limit}</p>
            </div>
            <div className="p-3 rounded-lg bg-zinc-900/50">
              <div className="flex items-center gap-2 text-xs text-zinc-500 mb-1"><Timer size={12} /> Uptime</div>
              <p className="text-lg font-semibold">{metrics.uptime > 86400 ? `${(metrics.uptime / 86400).toFixed(1)}d` : metrics.uptime > 3600 ? `${(metrics.uptime / 3600).toFixed(1)}h` : `${(metrics.uptime / 60).toFixed(0)}m`}</p>
            </div>
          </div>
          {metrics.restartCount > 0 && (
            <p className="mt-2 text-xs text-yellow-400">⚠️ Reiniciado {metrics.restartCount}x</p>
          )}
        </div>
      )}

      {project.repoUrl && (
        <div className="card mb-6">
          <h2 className="text-lg font-semibold mb-4 flex items-center gap-2"><Webhook size={16} /> Git Auto-Deploy</h2>
          <p className="text-sm text-zinc-400 mb-3">
            Push em qualquer branch faz deploy automático. Branch principal ({project.branch || "main"}) vai para produção;
            branches diferentes viram preview deployments.
          </p>
          <div className="flex items-center gap-2">
            <code className="flex-1 p-2 rounded bg-black/40 text-xs text-zinc-300 font-mono break-all">http://2.24.204.31:3001/api/webhook/github</code>
            <button onClick={() => navigator.clipboard.writeText("http://2.24.204.31:3001/api/webhook/github")} className="btn btn-ghost text-xs">Copiar</button>
          </div>
        </div>
      )}

      <div className="card mb-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold flex items-center gap-2"><Settings size={16} /> Environment Variables</h2>
          <button onClick={() => setShowEnv(!showEnv)} className="btn btn-ghost text-xs">
            {showEnv ? <EyeOff size={14} /> : <Eye size={14} />} {showEnv ? "Ocultar" : "Mostrar"}
          </button>
        </div>
        <textarea
          className="input font-mono text-xs min-h-[100px]"
          value={envText}
          onChange={(e) => { setEnvText(e.target.value); setEnvSaved(false) }}
          placeholder="DATABASE_URL=postgresql://..."
        />
        <div className="flex justify-end mt-3">
          <button onClick={handleSaveEnv} disabled={savingEnv} className="btn btn-primary text-xs">
            {savingEnv ? "Salvando..." : envSaved ? "✓ Salvo" : "Salvar Variáveis"}
          </button>
        </div>
      </div>

      <div className="mb-4 flex gap-2 border-b border-zinc-800">
        <button
          onClick={() => setActiveTab("production")}
          className={`px-4 py-2 text-sm font-medium transition border-b-2 -mb-[1px] ${
            activeTab === "production"
              ? "text-white border-accent"
              : "text-zinc-400 border-transparent hover:text-zinc-200"
          }`}
        >
          <Rocket size={14} className="inline mr-1.5" />
          Production
        </button>
        <button
          onClick={() => setActiveTab("previews")}
          className={`px-4 py-2 text-sm font-medium transition border-b-2 -mb-[1px] ${
            activeTab === "previews"
              ? "text-white border-accent"
              : "text-zinc-400 border-transparent hover:text-zinc-200"
          }`}
        >
          <GitBranch size={14} className="inline mr-1.5" />
          Previews
          {previews.length > 0 && (
            <span className="ml-1.5 px-1.5 py-0.5 text-[10px] rounded-full bg-zinc-800 text-zinc-300">
              {previews.length}
            </span>
          )}
        </button>
      </div>

      {activeTab === "production" && (
        <div>
          {deployments.length === 0 && (
            <div className="card text-center py-12">
              <p className="text-sm text-zinc-400">Nenhum deployment ainda.</p>
            </div>
          )}
          {deployments.map((dep: any) => (
            <DeploymentCard key={dep.id} dep={dep} />
          ))}
        </div>
      )}

      {activeTab === "previews" && (
        <div>
          {previews.length === 0 && (
            <div className="card text-center py-12">
              <GitBranch size={24} className="mx-auto mb-2 text-zinc-600" />
              <p className="text-sm text-zinc-400">Nenhum preview deployment.</p>
              <p className="text-xs text-zinc-500 mt-1">Faça push de uma branch diferente de <code className="text-zinc-300">{project.branch || "main"}</code> para criar um preview.</p>
            </div>
          )}
          {previews.map((dep: any) => (
            <DeploymentCard key={dep.id} dep={dep} />
          ))}
        </div>
      )}
    </div>
  )
}

function DeploymentCard({ dep }: { dep: any }) {
  const [showLogs, setShowLogs] = useState(false)
  return (
    <div className="card mb-2">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3 min-w-0">
          {dep.branch && dep.branch !== "main" && (
            <code className="text-xs px-2 py-0.5 rounded bg-zinc-800 text-accent font-mono shrink-0">
              {dep.branch}
            </code>
          )}
          <span className={`badge badge-${dep.status === "success" ? "active" : dep.status === "building" || dep.status === "pending" ? "building" : "failed"} shrink-0`}>
            {dep.status}
          </span>
          <span className="text-sm text-zinc-400 flex items-center gap-1 shrink-0">
            <Clock size={12} />
            {new Date(dep.createdAt).toLocaleString("pt-BR")}
          </span>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          {dep.url && (
            <a href={dep.url} target="_blank" className="text-sm text-accent hover:underline flex items-center gap-1">
              <ExternalLink size={12} />
              {dep.url.replace(/^https?:\/\//, "")}
            </a>
          )}
          {dep.logs && (
            <button onClick={() => setShowLogs(!showLogs)} className="text-xs text-zinc-500 hover:text-zinc-300 transition">
              {showLogs ? "ocultar logs" : "logs"}
            </button>
          )}
        </div>
      </div>
      {showLogs && dep.logs && (
        <pre className="mt-2 p-3 rounded-lg bg-black/40 text-xs text-zinc-300 overflow-x-auto max-h-48 overflow-y-auto font-mono leading-relaxed">{dep.logs}</pre>
      )}
    </div>
  )
}
