"use client"

import { useEffect, useState } from "react"
import { useParams, useRouter } from "next/navigation"
import { api } from "@/lib/api"
import { ArrowLeft, GitBranch, Clock, Rocket, ExternalLink } from "lucide-react"
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
  createdAt: string
  url?: string
}

export default function ProjectDetailPage() {
  const { id } = useParams<{ id: string }>()
  const router = useRouter()
  const [project, setProject] = useState<Project | null>(null)
  const [deployments, setDeployments] = useState<Deployment[]>([])

  useEffect(() => {
    if (!id) return
    api.projects.get(id).then(setProject).catch(() => router.push("/"))
    api.deployments.list(id).then(setDeployments).catch(() => {})
  }, [id, router])

  if (!project) return null

  return (
    <div>
      <Link href="/" className="inline-flex items-center gap-2 text-sm text-zinc-400 hover:text-white mb-8 transition">
        <ArrowLeft size={16} />
        Voltar
      </Link>

      {/* Project header */}
      <div className="flex items-start justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{project.name}</h1>
          <div className="flex items-center gap-3 mt-2">
            <span className={`badge badge-${project.status === "ACTIVE" ? "active" : "building"}`}>{project.status.toLowerCase()}</span>
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
      </div>

      {/* Deployments */}
      <div>
        <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
          <Rocket size={16} />
          Deployments
        </h2>

        {deployments.length === 0 && (
          <div className="card text-center py-12">
            <p className="text-sm text-zinc-400">Nenhum deployment ainda. Conecte um repositório Git para começar.</p>
          </div>
        )}

        {deployments.map((dep) => (
          <div key={dep.id} className="card flex items-center justify-between mb-2">
            <div className="flex items-center gap-3">
              <span className={`badge badge-${dep.status === "success" ? "active" : dep.status === "building" ? "building" : dep.status === "failed" ? "failed" : "paused"}`}>
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
        ))}
      </div>
    </div>
  )
}
