import { serverApi } from "@/lib/api-server"
import Link from "next/link"
import { ArrowLeft, Box, Layers, Cpu, MemoryStick } from "lucide-react"
import { ProjectDetailClient } from "./client"
import LiveMetrics from "./live-metrics"
import DeploymentHistoryChart from "@/components/DeploymentHistoryChart"

export const dynamic = "force-dynamic"
export const revalidate = 10

export default async function ProjectPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  let project: any = null
  let deployments: any[] = []
  let metrics: any = null
  try {
    project = await serverApi.projects.get(id)
    deployments = await serverApi.deployments.list(id)
    metrics = await serverApi.request(`/api/projects/${id}/metrics`)
  } catch {}

  if (!project || !project.id) {
    return (
      <div className="text-center py-16">
        <Box size={48} className="mx-auto mb-4 text-zinc-600" />
        <h2 className="text-lg font-semibold">Projeto nao encontrado</h2>
        <Link href="/dashboard/projects" className="text-sm text-accent mt-2 inline-block">Voltar</Link>
      </div>
    )
  }

  return (
    <div>
      <Link href="/dashboard/projects" className="inline-flex items-center gap-2 text-sm text-zinc-400 hover:text-white mb-8 transition">
        <ArrowLeft size={16} /> Voltar
      </Link>

      <div className="flex items-start justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{project.name}</h1>
          <div className="flex items-center gap-3 mt-2">
            <span className={`badge badge-${project.status === "ACTIVE" ? "active" : "building"}`}>{project.status?.toLowerCase?.() ?? "?"}</span>
            {project.framework && <span className="text-sm text-zinc-400"><Layers size={12} className="inline mr-1"/>{project.framework}</span>}
            {typeof project.port === "number" && project.port > 0 && (
              <span className="text-sm text-zinc-400 font-mono">:{project.port}</span>
            )}
          </div>
        </div>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-6">
        {project.repoUrl && (
          <div className="p-3 rounded-lg bg-zinc-900/50">
            <p className="text-xs text-zinc-500 mb-1">Repositorio</p>
            <p className="text-sm font-medium truncate">{project.repoUrl.replace("https://github.com/", "").replace(".git", "")}</p>
          </div>
        )}
        {project.branch && (
          <div className="p-3 rounded-lg bg-zinc-900/50">
            <p className="text-xs text-zinc-500 mb-1">Branch</p>
            <p className="text-sm font-medium">{project.branch}</p>
          </div>
        )}
        {project.domain && (
          <div className="p-3 rounded-lg bg-zinc-900/50">
            <p className="text-xs text-zinc-500 mb-1">Dominio</p>
            <p className="text-sm font-medium">{project.domain}</p>
          </div>
        )}
        {metrics?.running && (
          <div className="p-3 rounded-lg bg-zinc-900/50">
            <p className="text-xs text-zinc-500 mb-1">Uptime</p>
            <p className="text-sm font-medium">
              {Math.floor((Date.now() - new Date(metrics.startedAt + "Z").getTime()) / 1000 / 60)} min
            </p>
          </div>
        )}
      </div>

      {metrics?.running && <LiveMetrics metrics={metrics} projectId={id} />}

      <DeploymentHistoryChart projectId={id} />

      <div className="card mb-6">
        <h2 className="text-lg font-semibold mb-4">Deployments ({deployments?.length ?? 0})</h2>
        {!deployments || deployments.length === 0 ? (
          <p className="text-sm text-zinc-400 text-center py-8">Nenhum deployment ainda.</p>
        ) : (
          <div className="space-y-2">
            {deployments.slice(0, 5).map((dep: any) => (
              <div key={dep.id} className="flex items-center justify-between p-3 rounded-lg bg-zinc-900/30 text-sm">
                <div className="flex items-center gap-3">
                  <span className={`badge badge-${dep.status === "success" ? "active" : dep.status === "failed" ? "failed" : "building"}`}>
                    {dep.status}
                  </span>
                  <span className="text-zinc-500">
                    {dep.createdAt ? new Date(dep.createdAt).toLocaleString("pt-BR") : "\u2014"}
                  </span>
                </div>
                {dep.url && (
                  <a href={dep.url} target="_blank" className="text-accent text-xs hover:underline">{dep.url}</a>
                )}
              </div>
            ))}
          </div>
        )}
      </div>

      <ProjectDetailClient projectId={project.id} projectName={project.name} />
    </div>
  )
}
