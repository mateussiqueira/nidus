"use client"

import { useState, useEffect } from "react"
import { Rocket, ArrowLeft } from "lucide-react"
import Link from "next/link"
import { api } from "@/lib/api"

type Deployment = {
  id: string
  status: string
  url?: string
  createdAt: string
  projectName?: string
}

export default function DeploymentsPage() {
  const [deployments, setDeployments] = useState<Deployment[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api.projects.list().then(async (projects: any[]) => {
      const all = await Promise.all(
        projects.map((p) =>
          api.deployments.list(p.id).then((deps: any[]) =>
            deps.map((d: any) => ({ ...d, projectName: p.name }))
          )
        )
      )
      setDeployments(all.flat().sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()))
    }).catch(() => {}).finally(() => setLoading(false))
  }, [])

  return (
    <div>
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Deployments</h1>
          <p className="text-sm text-zinc-400 mt-1">Histórico de deploys de todos os projetos</p>
        </div>
      </div>

      {loading && <p className="text-sm text-zinc-500">Carregando...</p>}

      {!loading && deployments.length === 0 && (
        <div className="card text-center py-12">
          <Rocket size={32} className="text-zinc-600 mx-auto mb-3" />
          <p className="text-sm text-zinc-400">Nenhum deployment ainda.</p>
        </div>
      )}

      <div className="space-y-2">
        {deployments.map((dep) => (
          <div key={dep.id} className="card flex items-center justify-between">
            <div className="flex items-center gap-3">
              <span className={`badge badge-${dep.status === "success" ? "active" : dep.status === "building" ? "building" : "failed"}`}>
                {dep.status}
              </span>
              {dep.projectName && <span className="text-sm font-medium">{dep.projectName}</span>}
              <span className="text-xs text-zinc-500">{new Date(dep.createdAt).toLocaleString("pt-BR")}</span>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
