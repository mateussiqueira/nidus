"use client"

import { useEffect, useState } from "react"
import { api } from "@/lib/api"
import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend,
  PieChart, Pie, Cell,
} from "recharts"
import { TrendingUp, Clock, CheckCircle, XCircle } from "lucide-react"

interface Props {
  projectId?: string
  compact?: boolean
}

export default function DeploymentHistoryChart({ projectId, compact }: Props) {
  const [deployments, setDeployments] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    async function fetchData() {
      try {
        if (projectId) {
          const deps = await api.deployments.list(projectId)
          setDeployments(deps || [])
        } else {
          const projects = (await api.projects.list()) || []
          const all = await Promise.all(
            projects.map((p: any) =>
              api.deployments.list(p.id).then((deps: any[]) =>
                (deps || []).map((d: any) => ({ ...d, projectName: p.name }))
              )
            )
          )
          setDeployments(all.flat())
        }
      } catch {}
      setLoading(false)
    }
    fetchData()
  }, [projectId])

  if (loading) return null
  if (deployments.length === 0) return <p className="text-sm text-zinc-500 text-center py-4">Nenhum deploy para exibir no gráfico.</p>

  const total = deployments.length
  const success = deployments.filter((d) => d.status === "success").length
  const failed = deployments.filter((d) => d.status === "failed").length
  const successRate = total > 0 ? Math.round((success / total) * 100) : 0

  const withDuration = deployments.filter((d) => d.finishedAt && d.createdAt)
  const avgDuration = withDuration.length > 0
    ? withDuration.reduce((sum, d) => {
        const start = new Date(d.createdAt).getTime()
        const end = new Date(d.finishedAt).getTime()
        return sum + (end - start)
      }, 0) / withDuration.length
    : 0

  const avgDurationStr = avgDuration > 0
    ? avgDuration > 60000
      ? `${(avgDuration / 60000).toFixed(1)}min`
      : `${Math.round(avgDuration / 1000)}s`
    : "—"

  const grouped: Record<string, { date: string; total: number; success: number; failed: number }> = {}
  deployments.forEach((d) => {
    const date = new Date(d.createdAt).toLocaleDateString("pt-BR", { day: "2-digit", month: "2-digit" })
    if (!grouped[date]) grouped[date] = { date, total: 0, success: 0, failed: 0 }
    grouped[date].total++
    if (d.status === "success") grouped[date].success++
    else if (d.status === "failed") grouped[date].failed++
  })

  const timelineData = Object.values(grouped).sort((a, b) => {
    const [da, ma] = a.date.split("/")
    const [db, mb] = b.date.split("/")
    return parseInt(da) + parseInt(ma) * 30 - (parseInt(db) + parseInt(mb) * 30)
  })

  const pieData = [
    { name: "Sucesso", value: success, color: "#22d3ee" },
    { name: "Falha", value: failed, color: "#f87171" },
  ].filter((d) => d.value > 0)

  const cardClass = compact ? "mb-0" : "card mb-6"

  return (
    <div className={cardClass}>
      <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
        <TrendingUp size={16} /> Histórico de Deploys
      </h2>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-4">
        <div className="p-3 rounded-lg bg-zinc-900/50">
          <p className="text-xs text-zinc-500 mb-1">Total</p>
          <p className="text-xl font-bold text-zinc-100">{total}</p>
        </div>
        <div className="p-3 rounded-lg bg-zinc-900/50">
          <div className="flex items-center gap-1 text-xs text-zinc-500 mb-1">
            <CheckCircle size={10} className="text-cyan-400" /> Sucesso
          </div>
          <p className="text-xl font-bold text-cyan-400">{successRate}%</p>
        </div>
        <div className="p-3 rounded-lg bg-zinc-900/50">
          <div className="flex items-center gap-1 text-xs text-zinc-500 mb-1">
            <XCircle size={10} className="text-red-400" /> Falha
          </div>
          <p className="text-xl font-bold text-red-400">{100 - successRate}%</p>
        </div>
        <div className="p-3 rounded-lg bg-zinc-900/50">
          <div className="flex items-center gap-1 text-xs text-zinc-500 mb-1">
            <Clock size={10} /> Duração média
          </div>
          <p className="text-xl font-bold text-zinc-100">{avgDurationStr}</p>
        </div>
      </div>

      {!compact && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
          <div className="lg:col-span-2 p-3 rounded-lg bg-zinc-900/50">
            <p className="text-xs text-zinc-500 mb-2">Deploys por dia</p>
            <ResponsiveContainer width="100%" height={200}>
              <BarChart data={timelineData}>
                <CartesianGrid strokeDasharray="3 3" stroke="#27272a" />
                <XAxis dataKey="date" tick={{ fontSize: 10, fill: "#71717a" }} />
                <YAxis tick={{ fontSize: 10, fill: "#71717a" }} allowDecimals={false} />
                <Tooltip
                  contentStyle={{ background: "#18181b", border: "1px solid #27272a", borderRadius: "8px" }}
                  labelStyle={{ color: "#a1a1aa" }}
                />
                <Legend />
                <Bar dataKey="success" name="Sucesso" stackId="a" fill="#22d3ee" radius={[2, 2, 0, 0]} />
                <Bar dataKey="failed" name="Falha" stackId="a" fill="#f87171" radius={[2, 2, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
          <div className="p-3 rounded-lg bg-zinc-900/50">
            <p className="text-xs text-zinc-500 mb-2">Taxa de sucesso</p>
            <ResponsiveContainer width="100%" height={200}>
              <PieChart>
                <Pie
                  data={pieData}
                  cx="50%"
                  cy="50%"
                  innerRadius={50}
                  outerRadius={80}
                  paddingAngle={2}
                  dataKey="value"
                >
                  {pieData.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={entry.color} />
                  ))}
                </Pie>
                <Tooltip
                  contentStyle={{ background: "#18181b", border: "1px solid #27272a", borderRadius: "8px" }}
                />
                <Legend
                  formatter={(value) => <span className="text-xs text-zinc-400">{value}</span>}
                />
              </PieChart>
            </ResponsiveContainer>
          </div>
        </div>
      )}
    </div>
  )
}
