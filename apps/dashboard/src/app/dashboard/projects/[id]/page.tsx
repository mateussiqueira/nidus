"use client"
export const dynamic = "force-dynamic"

import { useEffect, useState } from "react"
import { useParams, useRouter } from "next/navigation"
import { api } from "@/lib/api"
import { ArrowLeft, GitBranch, Clock, Rocket, ExternalLink, Play } from "lucide-react"
import Link from "next/link"

type Project = {
  id: string
  name: string
  slug: string
  framework: string | null
  status: string
  domain: string | null
  repoUrl: string | null
  createdAt: string
}

type Deployment = {
  id: string
  status: string
  url?: string
  logs?: string
  createdAt: string
  finishedAt?: string
}

export default function ProjectDetailPage() {
  const { id } = useParams<{ id: string }>()
  const router = useRouter()
  const [project, setProject] = useState<Project | null>(null)
  const [deployments, setDeployments] = useState<Deployment[]>([])
  const [deploying, setDeploying] = useState(false)

  function load() {
    if (!id) return
    api.projects.get(id).then(setProject).catch(() => router.push("/dashboard"))
    api.deployments.list(id).then(setDeployments).catch(() => {})
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

  if (!project) return null

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
            <span className={`badge badge-${project.status === "ACTIVE" ? "active" : project.status === "BUILDING" || project.status === "DEPLOYING" ? "building" : project.status === "FAILED" ? "failed" : "paused"}`}>
              {project.status.toLowerCase()}
            </span>
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

      <div>
        <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
          <Rocket size={16} />
          Deployments
        </h2>

        {deployments.length === 0 && (
          <div className="card text-center py-12">
            <p className="text-sm text-zinc-400">Nenhum deployment ainda. Clique em "Deploy Now" para começar.</p>
          </div>
        )}

        {deployments.map((dep) => (
          <div key={dep.id} className="card mb-2">
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center gap-3">
                <span className={`badge badge-${dep.status === "success" ? "active" : dep.status === "building" ? "building" : "failed"}`}>
                  {dep.status}
                </span>
                <span className="text-sm text-zinc-400 flex items-center gap-1">
                  <Clock size={12} />
                  {new Date(dep.createdAt).toLocaleString("pt-BR")}
                </span>
              </div>
              {dep.url && (
                <a href={dep.url} target="_blank" className="text-sm text-accent hover:underline flex items-center gap-1">
                  {dep.url} <ExternalLink size={12} />
                </a>
              )}
            </div>
            {dep.logs && (
              <pre className="mt-2 p-3 rounded-lg bg-black/40 text-xs text-zinc-300 overflow-x-auto max-h-48 overflow-y-auto font-mono leading-relaxed">
                {dep.logs}
              </pre>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
