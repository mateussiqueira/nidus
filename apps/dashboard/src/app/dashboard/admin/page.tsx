"use client"
import { useEffect, useState } from "react"
import { api } from "@/lib/api"
import { Shield, Users, Box, Rocket, DollarSign, TrendingUp, CreditCard, Activity } from "lucide-react"

export default function AdminPage() {
  const [stats, setStats] = useState<any>(null)
  const [users, setUsers] = useState<any[]>([])
  const [payments, setPayments] = useState<any[]>([])
  const [tab, setTab] = useState<"overview"|"users"|"payments">("overview")

  useEffect(() => {
    api.request("/api/admin/stats").then(setStats).catch(() => {})
    api.request("/api/admin/users").then(setUsers).catch(() => {})
    api.request("/api/admin/payments").then(setPayments).catch(() => {})
  }, [])

  const formatCents = (c: number) => c ? `R$${(c/100).toFixed(2)}` : "R$0,00"
  const formatDate = (d: string) => d ? new Date(d).toLocaleDateString("pt-BR") : "—"

  if (!stats) return <div className="animate-pulse space-y-3">{[1,2,3].map(i => <div key={i} className="card h-24"/>)}</div>

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2"><Shield size={20} className="text-amber-400"/> Admin Panel</h1>
          <p className="text-zinc-500 text-sm mt-1">StackRun — Product Owner Dashboard</p>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 rounded-lg bg-zinc-900 p-1 w-fit">
        {(["overview","users","payments"] as const).map(t => (
          <button key={t} onClick={() => setTab(t)}
            className={`px-4 py-1.5 rounded-md text-sm font-medium ${tab===t?"bg-zinc-700 text-white":"text-zinc-400 hover:text-white"}`}>
            {t === "overview" ? "Visao Geral" : t === "users" ? "Usuarios" : "Pagamentos"}
          </button>
        ))}
      </div>

      {tab === "overview" && (
        <>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
            <div className="card !p-4"><Users size={16} className="text-cyan-400 mb-2"/><p className="text-2xl font-bold">{stats.totalUsers}</p><p className="text-xs text-zinc-500">usuarios</p></div>
            <div className="card !p-4"><Box size={16} className="text-green-400 mb-2"/><p className="text-2xl font-bold">{stats.totalProjects}</p><p className="text-xs text-zinc-500">projetos</p></div>
            <div className="card !p-4"><Rocket size={16} className="text-violet-400 mb-2"/><p className="text-2xl font-bold">{stats.deploys24h}</p><p className="text-xs text-zinc-500">deploys 24h</p></div>
            <div className="card !p-4"><Activity size={16} className="text-amber-400 mb-2"/><p className="text-2xl font-bold">{stats.healthyProjects}</p><p className="text-xs text-zinc-500">projetos UP</p></div>
          </div>

          <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
            <div className="card !p-4"><DollarSign size={16} className="text-green-400 mb-2"/><p className="text-2xl font-bold">{formatCents(stats.totalRevenueCents)}</p><p className="text-xs text-zinc-500">receita total</p></div>
            <div className="card !p-4"><CreditCard size={16} className="text-blue-400 mb-2"/><p className="text-2xl font-bold">{stats.activeSubscriptions}</p><p className="text-xs text-zinc-500">assinaturas ativas</p></div>
            <div className="card !p-4"><TrendingUp size={16} className="text-cyan-400 mb-2"/><p className="text-2xl font-bold">{stats.newUsers7d}</p><p className="text-xs text-zinc-500">novos 7d</p></div>
            <div className="card !p-4"><Rocket size={16} className="text-zinc-400 mb-2"/><p className="text-2xl font-bold">{stats.totalDeploys}</p><p className="text-xs text-zinc-500">deploys total</p></div>
          </div>

          <div className="card">
            <h3 className="font-semibold mb-3">Usuarios por Plano</h3>
            <div className="space-y-2">
              {Object.entries(stats.usersByPlan || {}).map(([plan, count]: any) => (
                <div key={plan} className="flex items-center justify-between">
                  <span className="text-sm capitalize">{plan || "free"}</span>
                  <div className="flex items-center gap-2">
                    <div className="h-2 bg-zinc-800 rounded-full w-48"><div className="h-2 bg-cyan-400 rounded-full" style={{width: `${Math.min((count/stats.totalUsers)*100, 100)}%`}}/></div>
                    <span className="text-sm text-zinc-400 w-8 text-right">{count}</span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </>
      )}

      {tab === "users" && (
        <div className="card overflow-x-auto">
          <table className="w-full text-sm">
            <thead><tr className="text-left text-zinc-500 border-b border-zinc-800"><th className="pb-2">Email</th><th className="pb-2">Nome</th><th className="pb-2">Plano</th><th className="pb-2">Role</th><th className="pb-2">Desde</th></tr></thead>
            <tbody>{users.map(u => (
              <tr key={u.id} className="border-b border-zinc-900"><td className="py-2 font-mono text-xs">{u.email}</td><td className="py-2">{u.name}</td><td className="py-2"><span className="badge badge-active text-[10px]">{u.plan||"free"}</span></td><td className="py-2"><span className="text-[10px] text-zinc-500">{u.role}</span></td><td className="py-2 text-xs text-zinc-500">{formatDate(u.createdAt)}</td></tr>
            ))}</tbody>
          </table>
        </div>
      )}

      {tab === "payments" && (
        <div className="card overflow-x-auto">
          <table className="w-full text-sm">
            <thead><tr className="text-left text-zinc-500 border-b border-zinc-800"><th className="pb-2">Usuario</th><th className="pb-2">Plano</th><th className="pb-2">Gateway</th><th className="pb-2">Valor</th><th className="pb-2">Status</th><th className="pb-2">Data</th></tr></thead>
            <tbody>{payments.map(p => (
              <tr key={p.id} className="border-b border-zinc-900"><td className="py-2 font-mono text-xs">{p.user}</td><td className="py-2">{p.plan}</td><td className="py-2"><span className="text-[10px] text-zinc-500">{p.gateway}</span></td><td className="py-2">{formatCents(p.amount)}</td><td className="py-2"><span className={`badge badge-${p.status==="paid"?"active":"building"} text-[10px]`}>{p.status}</span></td><td className="py-2 text-xs text-zinc-500">{formatDate(p.createdAt)}</td></tr>
            ))}</tbody>
          </table>
        </div>
      )}
    </div>
  )
}
