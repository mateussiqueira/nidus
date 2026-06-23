import "./globals.css"

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="pt-BR">
      <head>
        <title>Nidus</title>
        <meta name="description" content="Sua PaaS pessoal — deploy full-stack com suporte a Dart/Vaden" />
      </head>
      <body className="bg-zinc-950 text-zinc-100 antialiased">{children}</body>
    </html>
  )
}
