"use client"
export const dynamic = "force-dynamic"

import { useEffect, useState } from "react"
import { api, isAuthenticated, clearToken } from "@/lib/api"
import { Settings, User, Key, Shield } from "lucide-react"

export default function SettingsPage() {
  const [user, setUser] = useState<any>(null)

  useEffect(() => {
    if (isAuthenticated()) {
      api.auth.me().then(setUser).catch(() => {})
    }
  }, [])

  return (
    <div>
      <h1 className="text-2xl font-bold tracking-tight mb-2">Configuracoes</h1>
      <p className="text-sm text-zinc-400 mb-8">Gerencie sua conta e preferencias.</p>

      <div className="card mb-4">
        <h2 className="text-lg font-semibold mb-4 flex items-center gap-2"><User size={16} /> Perfil</h2>
        {user && (
          <div className="space-y-2 text-sm">
            <p><span className="text-zinc-500">Nome:</span> {user.name}</p>
            <p><span className="text-zinc-500">Email:</span> {user.email}</p>
            <p><span className="text-zinc-500">ID:</span> <code className="text-xs text-zinc-400">{user.id}</code></p>
          </div>
        )}
      </div>

      <div className="card mb-4">
        <h2 className="text-lg font-semibold mb-4 flex items-center gap-2"><Shield size={16} /> Seguranca</h2>
        <p className="text-sm text-zinc-400 mb-4">
          O token JWT expira automaticamente. Faca logout e login novamente para renovar.
        </p>
        <button
          onClick={() => { clearToken(); window.location.href = "/login" }}
          className="btn btn-ghost text-red-400"
        >
          Sair da conta
        </button>
      </div>

      <div className="card">
        <h2 className="text-lg font-semibold mb-4 flex items-center gap-2"><Key size={16} /> API</h2>
        <div className="text-sm space-y-2">
          <p><span className="text-zinc-500">Endpoint:</span> <code className="text-xs bg-zinc-800 px-2 py-0.5 rounded">{process.env.NEXT_PUBLIC_API_URL || "http://localhost:3001"}</code></p>
          <p><span className="text-zinc-500">Auth:</span> JWT Bearer Token</p>
          <p><span className="text-zinc-500">CORS:</span> Configurado para dashboard</p>
        </div>
      </div>
    </div>
  )
}
