"use client"
import { useEffect, useState } from "react"
import { api } from "@/lib/api"
import { CreditCard, Check, Zap } from "lucide-react"

export default function BillingPage() {
  const [plans, setPlans] = useState<any[]>([])
  const [usage, setUsage] = useState<any>(null)
  useEffect(() => {
    api.request("/api/plans").then(setPlans)
    api.request("/api/billing/usage").then(setUsage)
  }, [])

  async function subscribe(planId: string) {
    await api.request("/api/billing/subscribe", { method: "POST", body: JSON.stringify({ planId }) })
    api.request("/api/billing/usage").then(setUsage)
  }

  const formatPrice = (cents: number) => cents === 0 ? "Grátis" : `R$${(cents/100).toFixed(2)}/mês`

  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold">Planos</h1><p className="text-zinc-500 text-sm mt-1">Plano atual: {usage?.plan || "free"}</p></div>
      {usage && (
        <div className="grid grid-cols-3 gap-3 mb-6">
          <div className="card !p-4"><p className="text-xs text-zinc-500">Deploys</p><p className="text-xl font-bold">{usage.deploys}</p></div>
          <div className="card !p-4"><p className="text-xs text-zinc-500">Projetos</p><p className="text-xl font-bold">{usage.projects}</p></div>
          <div className="card !p-4"><p className="text-xs text-zinc-500">Bancos</p><p className="text-xl font-bold">{usage.databases}</p></div>
        </div>
      )}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {plans.map((plan) => (
          <div key={plan.id} className={`card p-6 ${usage?.plan === plan.id ? 'border-cyan-400/30' : ''}`}>
            <h3 className="text-lg font-semibold">{plan.name}</h3>
            <p className="text-3xl font-bold mt-2">{formatPrice(plan.priceCents)}</p>
            <ul className="mt-4 space-y-2 text-sm text-zinc-400">
              <li className="flex items-center gap-2"><Check size={14} className="text-green-400" /> {plan.maxProjects} projetos</li>
              <li className="flex items-center gap-2"><Check size={14} className="text-green-400" /> {plan.maxDatabases} bancos</li>
            </ul>
            <button onClick={() => subscribe(plan.id)} disabled={usage?.plan === plan.id} className={`btn w-full mt-4 ${usage?.plan === plan.id ? 'btn-ghost' : 'btn-primary'}`}>
              {usage?.plan === plan.id ? 'Atual' : 'Assinar'}
            </button>
          </div>
        ))}
      </div>
    </div>
  )
}
