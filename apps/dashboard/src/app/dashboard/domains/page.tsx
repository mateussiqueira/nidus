"use client"

import { useEffect, useState } from "react"
import { api } from "@/lib/api"
import { Globe, Plus, Trash2, ShieldCheck, Clock, ExternalLink } from "lucide-react"

interface Domain {
  id: string
  domain: string
  verified: boolean
  sslStatus: string
  createdAt: string
}

interface Project {
  id: string
  name: string
  slug: string
}

export default function DomainsPage() {
  const [projects, setProjects] = useState<Project[]>([])
  const [loading, setLoading] = useState(true)
  const [showAdd, setShowAdd] = useState(false)
  const [domainInput, setDomainInput] = useState("")
  const [selectedProject, setSelectedProject] = useState("")
  const [adding, setAdding] = useState(false)
  const [domains, setDomains] = useState<Record<string, Domain[]>>({})
  const [verifying, setVerifying] = useState<string | null>(null)
  const [expandedProject, setExpandedProject] = useState<string | null>(null)

  useEffect(() => { loadData() }, [])

  async function loadData() {
    try {
      const projs = await api.projects.list()
      setProjects(projs || [])
      const allDomains: Record<string, Domain[]> = {}
      for (const p of (projs || [])) {
        try {
          const d = await api.request("/api/projects/" + p.id + "/domains")
          allDomains[p.id] = d || []
        } catch {}
      }
      setDomains(allDomains)
    } catch {}
    setLoading(false)
  }

  async function handleAdd() {
    if (!domainInput.trim() || !selectedProject) return
    setAdding(true)
    try {
      await api.request("/api/projects/" + selectedProject + "/domains", {
        method: "POST",
        body: JSON.stringify({ domain: domainInput.trim() }),
      })
      setDomainInput("")
      setShowAdd(false)
      loadData()
    } catch {}
    setAdding(false)
  }

  async function handleDelete(projectId: string, domainId: string, domain: string) {
    if (!confirm("Remover dominio " + domain + "?")) return
    try {
      await api.request("/api/projects/" + projectId + "/domains/" + domainId, { method: "DELETE" })
      loadData()
    } catch {}
  }

  async function handleVerify(projectId: string, domainId: string) {
    setVerifying(domainId)
    try {
      const result = await api.request("/api/projects/" + projectId + "/domains/" + domainId + "/verify", { method: "POST" })
      alert("Verificacao: " + (result.verified ? "OK" : "Falhou") + "\nIP: " + result.ip + "\nTXT esperado: " + result.expectedTxt)
      loadData()
    } catch {}
    setVerifying(null)
  }

  function getSslColor(status: string) {
    switch (status) {
      case "active": case "verified": return "text-green-400"
      case "pending": return "text-yellow-400"
      case "error": case "failed": return "text-red-400"
      default: return "text-zinc-500"
    }
  }

  const totalDomains = Object.values(domains).flat().length

  if (loading) return <div className="animate-pulse space-y-3">{[1,2,3].map(i => <div key={i} className="card h-16" />)}</div>

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Dominios</h1>
          <p className="text-zinc-500 text-sm mt-1">{totalDomains} dominios em {projects.length} projetos</p>
        </div>
        <button onClick={() => setShowAdd(true)} className="btn btn-primary flex items-center gap-2">
          <Plus size={16} /> Adicionar Dominio
        </button>
      </div>

      {projects.map((project) => {
        const projectDomains = domains[project.id] || []
        const isExpanded = expandedProject === project.id
        return (
          <div key={project.id} className="card">
            <div className="flex items-center justify-between cursor-pointer" onClick={() => setExpandedProject(isExpanded ? null : project.id)}>
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-lg bg-cyan-500/10 flex items-center justify-center">
                  <Globe size={20} className="text-cyan-400" />
                </div>
                <div>
                  <h3 className="font-medium">{project.name}</h3>
                  <p className="text-xs text-zinc-500">
                    {projectDomains.length} dominios
                    {project.slug && <span className="ml-2 text-cyan-400/70">{project.slug}.nidus.app</span>}
                  </p>
                </div>
              </div>
              <span className="text-xs text-zinc-600">{isExpanded ? "\u25B2" : "\u25BC"}</span>
            </div>

            {isExpanded && (
              <div className="mt-3 border-t border-zinc-800 pt-3">
                {project.slug && (
                  <div className="flex items-center justify-between p-2 rounded bg-zinc-900/50 mb-2 text-sm">
                    <div className="flex items-center gap-2">
                      <ShieldCheck size={14} className="text-green-400" />
                      <span className="font-mono text-zinc-300">{project.slug}.nidus.app</span>
                      <span className="badge badge-active text-[10px]">auto</span>
                    </div>
                    <a href={"https://" + project.slug + ".nidus.app"} target="_blank" className="text-cyan-400 hover:underline text-xs flex items-center gap-1">
                      <ExternalLink size={12} /> Visitar
                    </a>
                  </div>
                )}

                {projectDomains.length === 0 && (
                  <p className="text-xs text-zinc-500 text-center py-3">Nenhum dominio customizado.</p>
                )}

                {projectDomains.map((d: Domain) => (
                  <div key={d.id} className="flex items-center justify-between p-2 rounded bg-zinc-900/50 mb-1 text-sm">
                    <div className="flex items-center gap-2">
                      {d.verified ? <ShieldCheck size={14} className="text-green-400" /> : <Clock size={14} className="text-yellow-400" />}
                      <span className="font-mono text-zinc-300">{d.domain}</span>
                      <span className={"text-[10px] " + getSslColor(d.sslStatus)}>{d.sslStatus || "\u2014"}</span>
                    </div>
                    <div className="flex items-center gap-2">
                      {!d.verified && (
                        <button onClick={() => handleVerify(project.id, d.id)} disabled={verifying === d.id} className="text-xs text-yellow-400 hover:text-yellow-300 bg-yellow-400/10 px-2 py-1 rounded">
                          {verifying === d.id ? "..." : "Verificar"}
                        </button>
                      )}
                      <button onClick={() => handleDelete(project.id, d.id, d.domain)} className="text-zinc-600 hover:text-red-400">
                        <Trash2 size={14} />
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )
      })}

      {showAdd && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-surface border border-border rounded-lg p-6 w-full max-w-md">
            <h2 className="text-lg font-semibold mb-4">Adicionar Dominio</h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-2">Projeto</label>
                <select value={selectedProject} onChange={e => setSelectedProject(e.target.value)} className="input w-full">
                  <option value="">Selecione...</option>
                  {projects.map(p => <option key={p.id} value={p.id}>{p.name}</option>)}
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">Dominio</label>
                <input type="text" value={domainInput} onChange={e => setDomainInput(e.target.value)} placeholder="meusite.com" className="input w-full" autoFocus />
              </div>
              <div className="bg-zinc-900/50 rounded p-3 text-xs text-zinc-500">
                <p>Apos adicionar, aponte o DNS para <code className="text-cyan-400">2.24.204.31</code> e clique Verificar.</p>
              </div>
              <div className="flex justify-end gap-3">
                <button onClick={() => setShowAdd(false)} className="btn btn-ghost">Cancelar</button>
                <button onClick={handleAdd} disabled={adding || !domainInput.trim() || !selectedProject} className="btn btn-primary">
                  {adding ? "Adicionando..." : "Adicionar"}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
