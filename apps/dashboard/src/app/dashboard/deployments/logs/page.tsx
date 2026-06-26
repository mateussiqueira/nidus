"use client"
export const dynamic = "force-dynamic"

import { useEffect, useState } from "react"
import { api } from "@/lib/api"
import { FileText, Search, Filter, ChevronDown, ChevronUp, Copy, Check } from "lucide-react"

interface Deployment {
  id: string
  projectId: string
  projectName: string
  status: string
  branch: string
  url?: string
  logs?: string
  createdAt: string
  finishedAt?: string
}

export default function DeploymentLogsPage() {
  const [deployments, setDeployments] = useState<Deployment[]>([])
  const [loading, setLoading] = useState(true)
  const [expandedId, setExpandedId] = useState<string | null>(null)
  const [search, setSearch] = useState("")
  const [statusFilter, setStatusFilter] = useState<string>("all")
  const [copied, setCopied] = useState<string | null>(null)

  useEffect(() => {
    loadDeployments()
  }, [])

  async function loadDeployments() {
    try {
      // Get all projects first
      const projects = await api.request("/api/projects")
      const allDeployments: Deployment[] = []

      // Get deployments for each project
      for (const project of projects) {
        const deploys = await api.request(`/api/projects/${project.id}/deployments`)
        allDeployments.push(
          ...deploys.map((d: Deployment) => ({
            ...d,
            projectName: project.name,
          }))
        )
      }

      // Sort by date descending
      allDeployments.sort(
        (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
      )

      setDeployments(allDeployments)
    } catch (err) {
      console.error("Failed to load deployments:", err)
    } finally {
      setLoading(false)
    }
  }

  function copyToClipboard(text: string, id: string) {
    navigator.clipboard.writeText(text)
    setCopied(id)
    setTimeout(() => setCopied(null), 2000)
  }

  const filteredDeployments = deployments.filter((d) => {
    const matchesSearch =
      search === "" ||
      d.projectName.toLowerCase().includes(search.toLowerCase()) ||
      d.branch.toLowerCase().includes(search.toLowerCase()) ||
      d.id.toLowerCase().includes(search.toLowerCase())
    const matchesStatus = statusFilter === "all" || d.status === statusFilter
    return matchesSearch && matchesStatus
  })

  function getStatusColor(status: string) {
    switch (status) {
      case "success":
        return "badge-active"
      case "building":
      case "deploying":
        return "badge-building"
      case "failed":
        return "badge-failed"
      default:
        return "badge"
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Logs de Deploy</h1>
        <p className="text-zinc-500 text-sm mt-1">Histórico completo de deploys com logs</p>
      </div>

      {/* Filters */}
      <div className="flex flex-col sm:flex-row gap-3">
        <div className="relative flex-1">
          <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-500" />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Buscar por projeto, branch ou ID..."
            className="input w-full pl-10"
          />
        </div>
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="input"
        >
          <option value="all">Todos os status</option>
          <option value="success">Sucesso</option>
          <option value="building">Buildando</option>
          <option value="failed">Falhou</option>
        </select>
      </div>

      {/* Deployments List */}
      {loading ? (
        <div className="space-y-3">
          {[1, 2, 3, 4, 5].map((i) => (
            <div key={i} className="card animate-pulse h-20" />
          ))}
        </div>
      ) : filteredDeployments.length === 0 ? (
        <div className="card text-center py-12">
          <FileText size={48} className="mx-auto text-zinc-600 mb-4" />
          <h3 className="text-lg font-medium mb-2">
            {search || statusFilter !== "all" ? "Nenhum resultado encontrado" : "Nenhum deploy registrado"}
          </h3>
          <p className="text-zinc-500 text-sm">
            {search || statusFilter !== "all"
              ? "Tente ajustar os filtros de busca."
              : "Faça seu primeiro deploy para ver o histórico aqui."}
          </p>
        </div>
      ) : (
        <div className="space-y-3">
          {filteredDeployments.map((deploy) => (
            <div key={deploy.id} className="card">
              <div
                className="flex items-center justify-between cursor-pointer"
                onClick={() => setExpandedId(expandedId === deploy.id ? null : deploy.id)}
              >
                <div className="flex items-center gap-3">
                  <div className="flex-1">
                    <div className="flex items-center gap-2">
                      <span className="font-medium">{deploy.projectName}</span>
                      <span className={`badge ${getStatusColor(deploy.status)}`}>
                        {deploy.status}
                      </span>
                    </div>
                    <div className="flex items-center gap-3 mt-1 text-xs text-zinc-500">
                      <span>{deploy.branch}</span>
                      <span>•</span>
                      <span>{new Date(deploy.createdAt).toLocaleString("pt-BR")}</span>
                      {deploy.finishedAt && (
                        <>
                          <span>•</span>
                          <span>
                            {Math.round(
                              (new Date(deploy.finishedAt).getTime() -
                                new Date(deploy.createdAt).getTime()) /
                                1000
                            )}
                            s
                          </span>
                        </>
                      )}
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  {deploy.url && (
                    <a
                      href={deploy.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      onClick={(e) => e.stopPropagation()}
                      className="p-2 rounded-md hover:bg-zinc-800 text-zinc-500 hover:text-white transition"
                    >
                      <FileText size={14} />
                    </a>
                  )}
                  {expandedId === deploy.id ? (
                    <ChevronUp size={16} className="text-zinc-500" />
                  ) : (
                    <ChevronDown size={16} className="text-zinc-500" />
                  )}
                </div>
              </div>

              {/* Expanded Logs */}
              {expandedId === deploy.id && deploy.logs && (
                <div className="mt-4 pt-4 border-t border-border">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-sm font-medium">Logs</span>
                    <button
                      onClick={(e) => {
                        e.stopPropagation()
                        copyToClipboard(deploy.logs || "", deploy.id)
                      }}
                      className="flex items-center gap-1 text-xs text-zinc-500 hover:text-white transition"
                    >
                      {copied === deploy.id ? (
                        <>
                          <Check size={12} className="text-green-500" /> Copiado!
                        </>
                      ) : (
                        <>
                          <Copy size={12} /> Copiar logs
                        </>
                      )}
                    </button>
                  </div>
                  <pre className="bg-zinc-900 rounded-lg p-4 text-xs font-mono text-zinc-400 overflow-x-auto max-h-96 overflow-y-auto">
                    {deploy.logs}
                  </pre>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
