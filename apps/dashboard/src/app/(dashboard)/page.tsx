"use client"

import { useEffect, useState } from "react"
import Link from "next/link"
import { api } from "@/lib/api"
import { Plus, ExternalLink, GitBranch, Clock, Box } from "lucide-react"

type Project = {
  id: string
  name: string
  slug: string
  framework: string | null
  status: string
  domain: string | null
  createdAt: string
}

export default function ProjectsPage() {
  const [projects, setProjects] = useState<Project[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api.projects.list()
      .then(setProjects)
      .catch(console.error)
      .finally(() => setLoading(false))
  }, [])

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Projetos</h1>
          <p className="text-sm text-zinc-400 mt-1">Gerencie seus projetos e deploys</p>
        </div>
        <Link href="/projects/new" className="btn btn-primary">
          <Plus size={16} />
          Novo Projeto
        </Link>
      </div>

      {/* Empty state */}
      {!loading && projects.length === 0 && (
        <div className="card flex flex-col items-center justify-center py-20 text-center">
          <Box size={48} className="text-zinc-600 mb-4" />
          <h2 className="text-lg font-semibold mb-2">Nenhum projeto ainda</h2>
          <p className="text-sm text-zinc-400 mb-6 max-w-md">Importe um repositório Git e faça deploy em segundos.</p>
          <Link href="/projects/new" className="btn btn-primary">Importar Projeto</Link>
        </div>
      )}

      {/* Loading */}
      {loading && (
        <div className="space-y-3">
          {[1, 2, 3].map((i) => (
            <div key={i} className="card animate-pulse">
              <div className="h-5 bg-zinc-800 rounded w-1/3 mb-3"></div>
              <div className="h-3 bg-zinc-800 rounded w-1/4"></div>
            </div>
          ))}
        </div>
      )}

      {/* Project list */}
      {projects.length > 0 && (
        <div className="space-y-3">
          {projects.map((project) => (
            <Link key={project.id} href={`/projects/${project.id}`} className="card flex items-center justify-between hover:bg-zinc-800/50 transition block">
              <div className="flex items-center gap-4">
                <div className="w-10 h-10 rounded-lg bg-zinc-800 flex items-center justify-center shrink-0">
                  <Box size={18} className="text-zinc-400" />
                </div>
                <div>
                  <h3 className="font-medium text-sm">{project.name}</h3>
                  <div className="flex items-center gap-3 mt-1">
                    <span className={`badge badge-${project.status === "ACTIVE" ? "active" : project.status === "BUILDING" ? "building" : project.status === "FAILED" ? "failed" : "paused"}`}>
                      {project.status.toLowerCase()}
                    </span>
                    {project.framework && (
                      <span className="text-xs text-zinc-500">{project.framework}</span>
                    )}
                    <span className="text-xs text-zinc-500 flex items-center gap-1">
                      <Clock size={12} />
                      {new Date(project.createdAt).toLocaleDateString("pt-BR")}
                    </span>
                  </div>
                </div>
              </div>
              <ExternalLink size={14} className="text-zinc-600" />
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}
