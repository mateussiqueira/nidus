"use client"
export const dynamic = "force-dynamic"

import { useEffect, useState } from "react"
import { api } from "@/lib/api"
import { Database, Plus, Trash2, ExternalLink, Copy, Check } from "lucide-react"

interface DatabaseItem {
  id: string
  name: string
  url: string
  projectId?: string
  createdAt: string
}

export default function DatabasesPage() {
  const [databases, setDatabases] = useState<DatabaseItem[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreate, setShowCreate] = useState(false)
  const [dbName, setDbName] = useState("")
  const [creating, setCreating] = useState(false)
  const [copied, setCopied] = useState<string | null>(null)

  useEffect(() => {
    loadDatabases()
  }, [])

  async function loadDatabases() {
    try {
      const data = await api.request("/api/databases")
      setDatabases(data)
    } catch (err) {
      console.error("Failed to load databases:", err)
    } finally {
      setLoading(false)
    }
  }

  async function handleCreate() {
    if (!dbName.trim()) return
    setCreating(true)
    try {
      await api.request("/api/databases", {
        method: "POST",
        body: JSON.stringify({ name: dbName }),
      })
      setDbName("")
      setShowCreate(false)
      loadDatabases()
    } catch (err) {
      console.error("Failed to create database:", err)
    } finally {
      setCreating(false)
    }
  }

  async function handleDelete(id: string, name: string) {
    if (!confirm(`Tem certeza que deseja deletar o banco "${name}"? Esta ação não pode ser desfeita.`)) {
      return
    }
    try {
      await api.request(`/api/databases/${id}`, { method: "DELETE" })
      loadDatabases()
    } catch (err) {
      console.error("Failed to delete database:", err)
    }
  }

  function copyToClipboard(text: string, id: string) {
    navigator.clipboard.writeText(text)
    setCopied(id)
    setTimeout(() => setCopied(null), 2000)
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Bancos de Dados</h1>
          <p className="text-zinc-500 text-sm mt-1">Gerencie seus bancos de dados PostgreSQL</p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="btn btn-primary flex items-center gap-2"
        >
          <Plus size={16} />
          Criar Banco
        </button>
      </div>

      {/* Create Modal */}
      {showCreate && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-surface border border-border rounded-lg p-6 w-full max-w-md">
            <h2 className="text-lg font-semibold mb-4">Criar Novo Banco</h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-2">Nome do Banco</label>
                <input
                  type="text"
                  value={dbName}
                  onChange={(e) => setDbName(e.target.value)}
                  placeholder="meu-banco"
                  className="input w-full"
                  autoFocus
                />
              </div>
              <div className="flex justify-end gap-3">
                <button
                  onClick={() => setShowCreate(false)}
                  className="btn btn-ghost"
                >
                  Cancelar
                </button>
                <button
                  onClick={handleCreate}
                  disabled={creating || !dbName.trim()}
                  className="btn btn-primary"
                >
                  {creating ? "Criando..." : "Criar"}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Database List */}
      {loading ? (
        <div className="space-y-3">
          {[1, 2, 3].map((i) => (
            <div key={i} className="card animate-pulse h-24" />
          ))}
        </div>
      ) : databases.length === 0 ? (
        <div className="card text-center py-12">
          <Database size={48} className="mx-auto text-zinc-600 mb-4" />
          <h3 className="text-lg font-medium mb-2">Nenhum banco criado</h3>
          <p className="text-zinc-500 text-sm mb-4">
            Crie seu primeiro banco de dados PostgreSQL para seus projetos.
          </p>
          <button
            onClick={() => setShowCreate(true)}
            className="btn btn-primary"
          >
            Criar Primeiro Banco
          </button>
        </div>
      ) : (
        <div className="space-y-3">
          {databases.map((db) => (
            <div key={db.id} className="card">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-lg bg-blue-500/10 flex items-center justify-center">
                    <Database size={20} className="text-blue-500" />
                  </div>
                  <div>
                    <h3 className="font-medium">{db.name}</h3>
                    <p className="text-xs text-zinc-500">
                      Criado em {new Date(db.createdAt).toLocaleDateString("pt-BR")}
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => copyToClipboard(db.url, db.id)}
                    className="p-2 rounded-md hover:bg-zinc-800 text-zinc-500 hover:text-white transition"
                    title="Copiar URL de conexão"
                  >
                    {copied === db.id ? <Check size={14} className="text-green-500" /> : <Copy size={14} />}
                  </button>
                  <button
                    onClick={() => handleDelete(db.id, db.name)}
                    className="p-2 rounded-md hover:bg-zinc-800 text-zinc-500 hover:text-red-500 transition"
                    title="Deletar banco"
                  >
                    <Trash2 size={14} />
                  </button>
                </div>
              </div>
              <div className="mt-3 p-2 bg-zinc-900 rounded text-xs font-mono text-zinc-400 break-all">
                {db.url}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
