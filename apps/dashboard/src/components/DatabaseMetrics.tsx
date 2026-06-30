"use client"

import { useEffect, useState } from "react"
import { api } from "@/lib/api"
import { Database, Activity, HardDrive, Zap } from "lucide-react"

interface Props {
  dbId: string
}

export default function DatabaseMetrics({ dbId }: Props) {
  const [metrics, setMetrics] = useState<any>(null)

  useEffect(() => {
    api.request("/api/databases/" + dbId + "/metrics")
      .then(setMetrics)
      .catch(() => {})
  }, [dbId])

  if (!metrics) return null

  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-2 mb-2">
      <div className="p-2 rounded bg-zinc-900/50">
        <p className="text-[10px] text-zinc-500 flex items-center gap-1"><HardDrive size={10}/> Tamanho</p>
        <p className="text-sm font-medium text-zinc-200">{metrics.size}</p>
      </div>
      <div className="p-2 rounded bg-zinc-900/50">
        <p className="text-[10px] text-zinc-500 flex items-center gap-1"><Activity size={10}/> Conexoes</p>
        <p className="text-sm font-medium text-zinc-200">{metrics.activeConnections} ativas</p>
      </div>
      <div className="p-2 rounded bg-zinc-900/50">
        <p className="text-[10px] text-zinc-500 flex items-center gap-1"><Zap size={10}/> Cache Hit</p>
        <p className="text-sm font-medium text-zinc-200">{metrics.cacheHitRatio}%</p>
      </div>
      <div className="p-2 rounded bg-zinc-900/50">
        <p className="text-[10px] text-zinc-500 flex items-center gap-1"><Database size={10}/> Total</p>
        <p className="text-sm font-medium text-zinc-200">{metrics.totalConnections} conns</p>
      </div>
    </div>
  )
}
