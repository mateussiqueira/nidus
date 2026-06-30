"use client"

import { Check, Loader2 } from "lucide-react"

export interface DeployStage {
  id: string
  label: string
  emoji: string
}

export const DEPLOY_STAGES: DeployStage[] = [
  { id: "clone", label: "Clone", emoji: "📦" },
  { id: "build", label: "Build", emoji: "🐳" },
  { id: "run", label: "Run", emoji: "🚀" },
  { id: "health", label: "Health", emoji: "🏥" },
  { id: "done", label: "Done", emoji: "✅" },
]

export function detectStage(logText: string): string {
  if (logText.includes("✅ Deploy concluido")) return "done"
  if (logText.includes("✅ Build concluido") || logText.includes("🏥 Health check OK")) return "health"
  if (logText.includes("🚀 Iniciando container") || logText.includes("🔄 Removendo container")) return "run"
  if (logText.includes("🐳 Build") || logText.includes("🔨 Comando:")) return "build"
  if (logText.includes("📦 Clonando") || logText.includes("🔍 Framework")) return "clone"
  return ""
}

export function DeployProgress({ currentStage, failed }: { currentStage: string; failed?: boolean }) {
  const currentIdx = DEPLOY_STAGES.findIndex((s) => s.id === currentStage)
  return (
    <div className="flex items-center gap-2 mb-4">
      {DEPLOY_STAGES.map((stage, i) => {
        const isActive = i === currentIdx
        const isPast = i < currentIdx
        const isFail = failed && i === currentIdx
        return (
          <div key={stage.id} className="flex items-center gap-2 flex-1 last:flex-none">
            <div
              className={`flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-medium transition-all ${
                isFail
                  ? "bg-red-900/40 text-red-400"
                  : isActive
                    ? "bg-emerald-900/40 text-emerald-400 ring-1 ring-emerald-500/50"
                    : isPast
                      ? "bg-emerald-900/20 text-emerald-500"
                      : "bg-zinc-800/50 text-zinc-600"
              }`}
            >
              <span className="text-sm leading-none">
                {isFail ? "✗" : isPast ? <Check size={12} /> : isActive ? <Loader2 size={12} className="animate-spin" /> : stage.emoji}
              </span>
              <span>{stage.label}</span>
            </div>
            {i < DEPLOY_STAGES.length - 1 && (
              <div className={`flex-1 h-px ${isPast ? "bg-emerald-500/50" : "bg-zinc-700"}`} />
            )}
          </div>
        )
      })}
    </div>
  )
}
