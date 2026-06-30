"use client"
export const dynamic = "force-dynamic"

import { useEffect, useState } from "react"
import { api } from "@/lib/api"
import { Activity, Cpu, HardDrive, MemoryStick, Server, RefreshCw, BarChart3 } from "lucide-react"

interface SystemMetrics {
  memory: {
    total: number
    used: number
    free: number
    percent: number
  }
  disk: {
    total: number
    used: number
    free: number
    percent: number
  }
  uptime: number
  containers: {
    running: number
    total: number
  }
  deploys: {
    total: number
    active: number
    success: number
    failed: number
  }
}

export default function MonitoringPage() {
  const [metrics, setMetrics] = useState<SystemMetrics | null>(null)
  const [loading, setLoading] = useState(true)
  const [lastRefresh, setLastRefresh] = useState<Date>(new Date())

  useEffect(() => {
    loadMetrics()
    const interval = setInterval(loadMetrics, 30000) // Refresh every 30s
    return () => clearInterval(interval)
  }, [])

  async function loadMetrics() {
    try {
      const data = await api.request("/api/metrics")
      setMetrics(data)
      setLastRefresh(new Date())
    } catch (err) {
      console.error("Failed to load metrics:", err)
    } finally {
      setLoading(false)
    }
  }

  function formatBytes(bytes: number): string {
    if (bytes === 0) return "0 B"
    const k = 1024
    const sizes = ["B", "KB", "MB", "GB", "TB"]
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i]
  }

  function formatUptime(seconds: number): string {
    const days = Math.floor(seconds / 86400)
    const hours = Math.floor((seconds % 86400) / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    return `${days}d ${hours}h ${minutes}m`
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Monitoramento</h1>
          <p className="text-zinc-500 text-sm mt-1">Métricas do sistema em tempo real</p>
        </div>
        <div className="flex items-center gap-3">
          <span className="text-xs text-zinc-500">
            Última atualização: {lastRefresh.toLocaleTimeString("pt-BR")}
          </span>
          <a
            href="https://metrics.stackrun.vercel.app/d/abcjdl/stackrun-platform"
            target="_blank"
            className="flex items-center gap-2 px-3 py-2 text-xs bg-zinc-800 hover:bg-zinc-700 rounded-lg transition-colors"
          >
            <BarChart3 size={14} /> Grafana
          </a>
          <button
            onClick={loadMetrics}
            className="p-2 rounded-md hover:bg-zinc-800 text-zinc-500 hover:text-white transition"
          >
            <RefreshCw size={16} />
          </button>
        </div>
      </div>

      {loading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="card animate-pulse h-32" />
          ))}
        </div>
      ) : metrics ? (
        <>
          {/* System Resources */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            {/* CPU/Memory */}
            <div className="card">
              <div className="flex items-center gap-3 mb-3">
                <div className="w-10 h-10 rounded-lg bg-green-500/10 flex items-center justify-center">
                  <MemoryStick size={20} className="text-green-500" />
                </div>
                <div>
                  <p className="text-sm text-zinc-500">Memória</p>
                  <p className="text-lg font-semibold">{metrics.memory.percent.toFixed(1)}%</p>
                </div>
              </div>
              <div className="w-full bg-zinc-800 rounded-full h-2">
                <div
                  className="bg-green-500 h-2 rounded-full transition-all"
                  style={{ width: `${metrics.memory.percent}%` }}
                />
              </div>
              <p className="text-xs text-zinc-500 mt-2">
                {formatBytes(metrics.memory.used)} / {formatBytes(metrics.memory.total)}
              </p>
            </div>

            {/* Disk */}
            <div className="card">
              <div className="flex items-center gap-3 mb-3">
                <div className="w-10 h-10 rounded-lg bg-blue-500/10 flex items-center justify-center">
                  <HardDrive size={20} className="text-blue-500" />
                </div>
                <div>
                  <p className="text-sm text-zinc-500">Disco</p>
                  <p className="text-lg font-semibold">{metrics.disk.percent.toFixed(1)}%</p>
                </div>
              </div>
              <div className="w-full bg-zinc-800 rounded-full h-2">
                <div
                  className="bg-blue-500 h-2 rounded-full transition-all"
                  style={{ width: `${metrics.disk.percent}%` }}
                />
              </div>
              <p className="text-xs text-zinc-500 mt-2">
                {formatBytes(metrics.disk.used)} / {formatBytes(metrics.disk.total)}
              </p>
            </div>

            {/* Uptime */}
            <div className="card">
              <div className="flex items-center gap-3 mb-3">
                <div className="w-10 h-10 rounded-lg bg-purple-500/10 flex items-center justify-center">
                  <Activity size={20} className="text-purple-500" />
                </div>
                <div>
                  <p className="text-sm text-zinc-500">Uptime</p>
                  <p className="text-lg font-semibold">{formatUptime(metrics.uptime)}</p>
                </div>
              </div>
              <p className="text-xs text-zinc-500">Sistema online</p>
            </div>

            {/* Containers */}
            <div className="card">
              <div className="flex items-center gap-3 mb-3">
                <div className="w-10 h-10 rounded-lg bg-orange-500/10 flex items-center justify-center">
                  <Server size={20} className="text-orange-500" />
                </div>
                <div>
                  <p className="text-sm text-zinc-500">Containers</p>
                  <p className="text-lg font-semibold">
                    {metrics.containers.running}/{metrics.containers.total}
                  </p>
                </div>
              </div>
              <p className="text-xs text-zinc-500"> rodando</p>
            </div>
          </div>

          {/* Deploy Statistics */}
          <div className="card">
            <h2 className="text-lg font-semibold mb-4">Estatísticas de Deploys</h2>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <div className="text-center p-4 bg-zinc-900 rounded-lg">
                <p className="text-3xl font-bold text-white">{metrics.deploys.total}</p>
                <p className="text-sm text-zinc-500">Total</p>
              </div>
              <div className="text-center p-4 bg-zinc-900 rounded-lg">
                <p className="text-3xl font-bold text-green-500">{metrics.deploys.success}</p>
                <p className="text-sm text-zinc-500">Sucesso</p>
              </div>
              <div className="text-center p-4 bg-zinc-900 rounded-lg">
                <p className="text-3xl font-bold text-yellow-500">{metrics.deploys.active}</p>
                <p className="text-sm text-zinc-500">Ativos</p>
              </div>
              <div className="text-center p-4 bg-zinc-900 rounded-lg">
                <p className="text-3xl font-bold text-red-500">{metrics.deploys.failed}</p>
                <p className="text-sm text-zinc-500">Falhas</p>
              </div>
            </div>
          </div>
        </>
      ) : (
        <div className="card text-center py-12">
          <Activity size={48} className="mx-auto text-zinc-600 mb-4" />
          <h3 className="text-lg font-medium mb-2">Erro ao carregar métricas</h3>
          <p className="text-zinc-500 text-sm">
            Verifique se o sistema está rodando corretamente.
          </p>
        </div>
      )}
    </div>
  )
}
