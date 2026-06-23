"use client"

import { useEffect, useState } from "react"
import { useRouter, usePathname } from "next/navigation"
import Link from "next/link"
import { isAuthenticated, clearToken, api } from "@/lib/api"
import { Box, Rocket, LayoutDashboard, Settings, LogOut, ChevronDown } from "lucide-react"

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter()
  const pathname = usePathname()
  const [ready, setReady] = useState(false)
  const [user, setUser] = useState<{ name: string; email: string } | null>(null)

  useEffect(() => {
    if (!isAuthenticated()) {
      router.push("/login")
      return
    }
    api.auth.me().then(setUser).catch(() => {
      clearToken()
      router.push("/login")
    })
    setReady(true)
  }, [router])

  if (!ready) return null

  function handleLogout() {
    clearToken()
    router.push("/login")
  }

  const links = [
    { href: "/", label: "Projetos", icon: LayoutDashboard },
    { href: "/deployments", label: "Deployments", icon: Rocket },
  ]

  return (
    <div className="flex h-screen overflow-hidden">
      {/* Sidebar */}
      <aside className="w-56 shrink-0 bg-sidebar border-r border-border flex flex-col">
        <div className="flex items-center gap-3 px-4 h-14 border-b border-border">
          <div className="w-7 h-7 rounded-md bg-accent flex items-center justify-center">
            <span className="text-xs font-bold text-black">C</span>
          </div>
          <span className="font-semibold text-sm">Canopy</span>
        </div>

        <nav className="flex-1 p-3 space-y-1">
          {links.map((link) => {
            const active = pathname === link.href
            return (
              <Link key={link.href} href={link.href} className={`sidebar-link ${active ? "active" : ""}`}>
                <link.icon size={16} />
                {link.label}
              </Link>
            )
          })}
        </nav>

        <div className="p-3 border-t border-border">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2 min-w-0">
              <div className="w-7 h-7 rounded-full bg-zinc-700 flex items-center justify-center shrink-0">
                <span className="text-xs font-medium text-white">{user?.name?.charAt(0) ?? "?"}</span>
              </div>
              <div className="min-w-0">
                <p className="text-sm font-medium truncate">{user?.name ?? "Usuário"}</p>
                <p className="text-xs text-zinc-500 truncate">{user?.email}</p>
              </div>
            </div>
            <button onClick={handleLogout} className="shrink-0 p-1.5 rounded-md hover:bg-zinc-800 text-zinc-500 hover:text-white transition" title="Sair">
              <LogOut size={14} />
            </button>
          </div>
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-y-auto bg-surface">
        <div className="max-w-6xl mx-auto p-8">
          {children}
        </div>
      </main>
    </div>
  )
}
