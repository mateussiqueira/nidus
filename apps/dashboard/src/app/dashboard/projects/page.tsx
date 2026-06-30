import { serverApi } from "@/lib/api-server"
import Link from "next/link"
import { Plus, Box, Globe, GitBranch, Layers } from "lucide-react"
import { ProjectsClient } from "./client"

export const dynamic = "force-dynamic"
export const revalidate = 10

export default async function ProjectsPage() {
  let projects: any[] = []
  try { projects = await serverApi.projects.list() } catch {}

  return (
    <div>
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Projetos</h1>
          <p className="text-sm text-zinc-400 mt-1">{projects?.length ?? 0} projeto(s)</p>
        </div>
        <Link href="/dashboard/projects/new" className="btn btn-primary inline-flex items-center gap-2">
          <Plus size={16} />
          Novo Projeto
        </Link>
      </div>

      {!projects || projects.length === 0 ? (
        <div className="card text-center py-16">
          <Box size={48} className="mx-auto mb-4 text-zinc-600" />
          <h2 className="text-lg font-semibold mb-2">Nenhum projeto ainda</h2>
          <p className="text-sm text-zinc-400 mb-6">Crie seu primeiro projeto para começar a fazer deploy.</p>
          <Link href="/dashboard/projects/new" className="btn btn-primary inline-flex items-center gap-2">
            <Plus size={16} /> Criar Projeto
          </Link>
        </div>
      ) : (
        <div className="grid gap-4">
          {projects.map((project: any) => (
            <div key={project.id} className="card group hover:border-zinc-700 transition-all">
              <div className="flex items-start justify-between">
                <Link href={`/dashboard/projects/${project.id}`} className="flex-1 min-w-0">
                  <div className="flex items-center gap-3 mb-2">
                    <div className="w-10 h-10 rounded-lg bg-zinc-800 flex items-center justify-center shrink-0 group-hover:bg-zinc-700 transition">
                      <Box size={18} className="text-zinc-400" />
                    </div>
                    <div className="min-w-0">
                      <h3 className="font-semibold group-hover:text-accent transition truncate">{project.name}</h3>
                      <p className="text-xs text-zinc-500 truncate">{project.slug}</p>
                    </div>
                  </div>
                </Link>
                <div className="flex items-center gap-2 shrink-0">
                  {typeof project.port === "number" && project.port > 0 && (
                    <span className="text-xs px-2 py-1 rounded bg-zinc-800 text-zinc-400 font-mono">:{project.port}</span>
                  )}
                  <span className={`badge badge-${project.status === "ACTIVE" ? "active" : project.status === "FAILED" ? "failed" : "building"}`}>
                    {project.status?.toLowerCase?.() ?? "unknown"}
                  </span>
                </div>
              </div>

              <div className="flex items-center gap-4 mt-3 text-xs text-zinc-500">
                {project.framework && (
                  <span className="flex items-center gap-1"><Layers size={12} />{project.framework}</span>
                )}
                {project.repoUrl && (
                  <span className="flex items-center gap-1 truncate"><GitBranch size={12} />{project.repoUrl.replace("https://github.com/", "").replace(".git", "")}</span>
                )}
                {project.domain && (
                  <span className="flex items-center gap-1"><Globe size={12} />{project.domain}</span>
                )}
              </div>

              <div className="flex items-center gap-2 mt-3 pt-3 border-t border-zinc-800">
                <Link href={`/dashboard/projects/${project.id}`} className="btn btn-ghost text-xs">Detalhes</Link>
                <ProjectsClient projectId={project.id} projectName={project.name} />
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
