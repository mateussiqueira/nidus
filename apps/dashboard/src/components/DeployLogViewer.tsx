"use client"

import { useEffect, useState, useRef } from "react"
import { Terminal, Wifi, WifiOff, Copy, Check } from "lucide-react"
import { DeployProgress, detectStage } from "./DeployProgress"

interface LogViewerProps {
  deploymentId: string
  initialLogs?: string
  onComplete?: () => void
}

export function DeployLogViewer({ deploymentId, initialLogs, onComplete }: LogViewerProps) {
  const [logs, setLogs] = useState(initialLogs || "")
  const [currentStage, setCurrentStage] = useState("")
  const [wsConnected, setWsConnected] = useState(false)
  const [copied, setCopied] = useState(false)
  const [failed, setFailed] = useState(false)
  const bottomRef = useRef<HTMLDivElement>(null)
  const wsRef = useRef<WebSocket | null>(null)

  useEffect(() => {
    const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:3001"
    const wsUrl = apiUrl.replace(/^http/, "ws")
    const ws = new WebSocket(`${wsUrl}/api/ws/deployments/${deploymentId}/logs`)
    wsRef.current = ws
    ws.onopen = () => setWsConnected(true)
    ws.onmessage = (event) => {
      const newLogs = (prev: string) => {
        const updated = prev + event.data
        const stage = detectStage(updated)
        if (stage) setCurrentStage(stage)
        if (updated.includes("❌") || updated.includes("Deploy failed")) setFailed(true)
        if (updated.includes("✅ Deploy concluido")) {
          setCurrentStage("done")
          setTimeout(() => onComplete?.(), 1000)
        }
        return updated
      }
      setLogs(newLogs)
    }
    ws.onclose = () => setWsConnected(false)
    ws.onerror = () => setWsConnected(false)
    return () => ws.close()
  }, [deploymentId, onComplete])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" })
  }, [logs])

  useEffect(() => {
    const stage = detectStage(logs)
    if (stage) setCurrentStage(stage)
    if (logs.includes("✅ Deploy concluido")) setCurrentStage("done")
  }, [logs])

  function copyLogs() {
    navigator.clipboard.writeText(logs)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="card">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-semibold flex items-center gap-2">
          <Terminal size={14} /> Deploy Logs
          <span className={`flex items-center gap-1 text-xs ${wsConnected ? "text-green-400" : "text-yellow-400"}`}>
            {wsConnected ? <Wifi size={10} /> : <WifiOff size={10} />}
            {wsConnected ? "ao vivo" : "conectando..."}
          </span>
        </h3>
        <button onClick={copyLogs} className="flex items-center gap-1 text-xs text-zinc-500 hover:text-white transition">
          {copied ? <Check size={12} className="text-green-500" /> : <Copy size={12} />}
          {copied ? "Copiado!" : "Copiar"}
        </button>
      </div>

      {(currentStage || failed) && (
        <DeployProgress currentStage={currentStage} failed={failed} />
      )}

      <pre className="bg-zinc-900 rounded-lg p-4 text-xs font-mono text-zinc-400 overflow-x-auto max-h-80 overflow-y-auto whitespace-pre-wrap leading-relaxed">
        {logs || "(aguardando logs...)"}
        <div ref={bottomRef} />
      </pre>
    </div>
  )
}
