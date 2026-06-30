"use client"

import { useEffect, useState } from "react"
import { Cpu, MemoryStick, Activity } from "lucide-react"
import MetricsChart from "@/components/MetricsChart"

interface LiveMetricsProps {
  metrics: any
  projectId: string
}

export default function LiveMetrics({ metrics: initial, projectId }: LiveMetricsProps) {
  const [current, setCurrent] = useState(initial)

  useEffect(() => {
    const interval = setInterval(async () => {
      try {
        const res = await fetch(
          process.env.NEXT_PUBLIC_API_URL + "/api/projects/" + projectId + "/metrics",
          { headers: { Authorization: "Bearer " + (localStorage.getItem("nidus_token") || "") } }
        )
        if (res.ok) setCurrent(await res.json())
      } catch {}
    }, 5000)
    return () => clearInterval(interval)
  }, [projectId])

  if (!current?.running) return null

  const cpu = parseFloat(current.cpu) || 0
  const memPercent = parseFloat(current.memory?.percent) || 0
  const memUsage = current.memory?.usage || "0"

  return (
    <div className="mb-6">
      <div className="grid grid-cols-3 gap-3 mb-4">
        <div className="card !p-4">
          <div className="flex items-center gap-2 text-xs text-zinc-500 mb-2">
            <Cpu size={14} className="text-cyan-400" /> CPU
          </div>
          <p className="text-2xl font-bold">{cpu.toFixed(1)}<span className="text-sm text-zinc-500 font-normal">%</span></p>
          <div className="mt-2 h-1.5 bg-zinc-800 rounded-full overflow-hidden">
            <div className="h-full bg-cyan-400 rounded-full transition-all duration-500" style={{ width: Math.min(cpu, 100) + "%" }} />
          </div>
        </div>
        <div className="card !p-4">
          <div className="flex items-center gap-2 text-xs text-zinc-500 mb-2">
            <MemoryStick size={14} className="text-violet-400" /> Memoria
          </div>
          <p className="text-2xl font-bold">{memPercent.toFixed(1)}<span className="text-sm text-zinc-500 font-normal">%</span></p>
          <p className="text-xs text-zinc-500 mt-1">{memUsage}</p>
          <div className="mt-2 h-1.5 bg-zinc-800 rounded-full overflow-hidden">
            <div className="h-full bg-violet-400 rounded-full transition-all duration-500" style={{ width: Math.min(memPercent, 100) + "%" }} />
          </div>
        </div>
        <div className="card !p-4">
          <div className="flex items-center gap-2 text-xs text-zinc-500 mb-2">
            <Activity size={14} className="text-green-400" /> Status
          </div>
          <p className="text-sm font-medium text-green-400 flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-green-400 inline-block animate-pulse" />
            Online
          </p>
          <p className="text-xs text-zinc-500 mt-1">Restarts: {current.restartCount || 0}</p>
        </div>
      </div>

      <MetricsChart projectId={projectId} />
    </div>
  )
}
