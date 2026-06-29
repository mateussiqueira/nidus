"use client"

import { useEffect, useState } from "react"
import { api } from "@/lib/api"
import { Cpu, MemoryStick, TrendingUp } from "lucide-react"
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend,
} from "recharts"

export default function MetricsChart({ projectId }: { projectId: string }) {
  const [history, setHistory] = useState<any>(null)
  const [timeRange, setTimeRange] = useState<"1h" | "6h" | "24h">("1h")

  useEffect(() => {
    if (!projectId) return
    api.deployments.metricsHistory(projectId).then(setHistory).catch(() => {})
  }, [projectId, timeRange])

  if (!history) return null

  const cpuData = (history.cpu || []).map((p: any) => ({
    time: new Date(p.t * 1000).toLocaleTimeString("pt-BR"),
    cpu: p.v,
  }))

  const memData = (history.memory || []).map((p: any) => ({
    time: new Date(p.t * 1000).toLocaleTimeString("pt-BR"),
    memory: p.v,
  }))

  const combined = cpuData.map((c: any, i: number) => ({
    time: c.time,
    cpu: c.cpu,
    memory: memData[i]?.memory ?? 0,
  }))

  if (combined.length === 0) return null

  const formatMem = (v: number) => {
    if (v >= 1073741824) return `${(v / 1073741824).toFixed(1)}GB`
    if (v >= 1048576) return `${(v / 1048576).toFixed(1)}MB`
    if (v >= 1024) return `${(v / 1024).toFixed(0)}KB`
    return `${v.toFixed(0)}B`
  }

  return (
    <div className="card mb-6">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold flex items-center gap-2">
          <TrendingUp size={16} /> Métricas Históricas
        </h2>
        <div className="flex gap-1">
          {(["1h", "6h", "24h"] as const).map((range) => (
            <button
              key={range}
              onClick={() => setTimeRange(range)}
              className={`px-2 py-1 text-xs rounded transition ${
                timeRange === range
                  ? "bg-accent/20 text-accent border border-accent/30"
                  : "text-zinc-400 border border-zinc-700 hover:text-zinc-200"
              }`}
            >
              {range}
            </button>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <div className="p-3 rounded-lg bg-zinc-900/50">
          <div className="flex items-center gap-2 text-xs text-zinc-500 mb-2">
            <Cpu size={12} /> CPU (%)
          </div>
          <ResponsiveContainer width="100%" height={180}>
            <LineChart data={combined}>
              <CartesianGrid strokeDasharray="3 3" stroke="#27272a" />
              <XAxis dataKey="time" tick={{ fontSize: 10, fill: "#71717a" }} />
              <YAxis tick={{ fontSize: 10, fill: "#71717a" }} domain={[0, "auto"]} />
              <Tooltip
                contentStyle={{ background: "#18181b", border: "1px solid #27272a", borderRadius: "8px" }}
                labelStyle={{ color: "#a1a1aa" }}
              />
              <Line
                type="monotone"
                dataKey="cpu"
                stroke="#22d3ee"
                strokeWidth={2}
                dot={false}
                name="CPU %"
              />
            </LineChart>
          </ResponsiveContainer>
        </div>

        <div className="p-3 rounded-lg bg-zinc-900/50">
          <div className="flex items-center gap-2 text-xs text-zinc-500 mb-2">
            <MemoryStick size={12} /> Memória
          </div>
          <ResponsiveContainer width="100%" height={180}>
            <LineChart data={combined}>
              <CartesianGrid strokeDasharray="3 3" stroke="#27272a" />
              <XAxis dataKey="time" tick={{ fontSize: 10, fill: "#71717a" }} />
              <YAxis tick={{ fontSize: 10, fill: "#71717a" }} tickFormatter={formatMem} />
              <Tooltip
                contentStyle={{ background: "#18181b", border: "1px solid #27272a", borderRadius: "8px" }}
                labelStyle={{ color: "#a1a1aa" }}
                formatter={(value: any) => [formatMem(Number(value)), "Memória"] as any}
              />
              <Line
                type="monotone"
                dataKey="memory"
                stroke="#a78bfa"
                strokeWidth={2}
                dot={false}
                name="Memória"
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </div>
    </div>
  )
}
