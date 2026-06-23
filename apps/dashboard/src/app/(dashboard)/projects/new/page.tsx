"use client"

import { useState } from "react"
import { useRouter } from "next/navigation"
import { api } from "@/lib/api"
import { ArrowLeft, GitBranch, Link2 } from "lucide-react"
import Link from "next/link"

export default function NewProjectPage() {
  const router = useRouter()
  const [name, setName] = useState("")
  const [slug, setSlug] = useState("")
  const [repoUrl, setRepoUrl] = useState("")
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)

  function handleNameChange(val: string) {
    setName(val)
    setSlug(val.toLowerCase().replace(/[^a-z0-9-]/g, "-").replace(/-+/g, "-").replace(/^-|-$/g, ""))
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError("")
    setLoading(true)
    try {
      const project = await api.projects.create({ name, slug, repoUrl: repoUrl || undefined })
      router.push(`/projects/${project.id}`)
    } catch (err: any) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="max-w-2xl">
      <Link href="/" className="inline-flex items-center gap-2 text-sm text-zinc-400 hover:text-white mb-8 transition">
        <ArrowLeft size={16} />
        Voltar
      </Link>

      <h1 className="text-2xl font-bold tracking-tight mb-8">Novo Projeto</h1>

      <form onSubmit={handleSubmit} className="card space-y-6">
        <div>
          <label className="mb-1 block text-sm font-medium text-zinc-400">Nome do projeto</label>
          <input className="input" value={name} onChange={(e) => handleNameChange(e.target.value)} placeholder="Meu App" required />
        </div>

        <div>
          <label className="mb-1 block text-sm font-medium text-zinc-400">Slug</label>
          <div className="flex items-center gap-2">
            <span className="text-sm text-zinc-500">canopy.dev/</span>
            <input className="input" value={slug} onChange={(e) => setSlug(e.target.value)} placeholder="meu-app" required />
          </div>
        </div>

        <div>
          <label className="mb-1 block text-sm font-medium text-zinc-400">
            <span className="flex items-center gap-2">
              <GitBranch size={14} />
              URL do repositório Git (opcional)
            </span>
          </label>
          <input className="input" value={repoUrl} onChange={(e) => setRepoUrl(e.target.value)} placeholder="https://github.com/user/repo" />
        </div>

        {error && <p className="text-sm text-red-400">{error}</p>}

        <div className="flex gap-3 pt-2">
          <button type="submit" disabled={loading} className="btn btn-primary">
            {loading ? "Criando..." : "Criar Projeto"}
          </button>
          <Link href="/" className="btn btn-ghost">Cancelar</Link>
        </div>
      </form>
    </div>
  )
}
