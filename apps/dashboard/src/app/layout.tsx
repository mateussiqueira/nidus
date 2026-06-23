import type { Metadata } from "next"
import "./globals.css"

export const dynamic = "force-dynamic"

export const metadata: Metadata = {
  title: "Nidus",
  description: "Sua PaaS pessoal — deploy full-stack com suporte a Dart/Vaden",
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="pt-BR">
      <body className="bg-zinc-950 text-zinc-100 antialiased">{children}</body>
    </html>
  )
}
