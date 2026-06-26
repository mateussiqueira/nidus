import Link from 'next/link'

export default function ArchitecturePT() {
  return (
    <main className="min-h-screen bg-gray-950 text-white">
      <nav className="border-b border-gray-800 p-4">
        <div className="container mx-auto flex justify-between items-center">
          <Link href="/pt" className="text-xl font-bold">Nidus Docs</Link>
          <div className="flex gap-4">
            <Link href="/pt" className="text-blue-400">PT</Link>
            <Link href="/en" className="text-gray-400 hover:text-white">EN</Link>
          </div>
        </div>
      </nav>
      
      <div className="container mx-auto px-4 py-16 max-w-3xl">
        <Link href="/pt" className="text-blue-400 hover:underline mb-8 block">&larr; Voltar</Link>
        
        <h1 className="text-4xl font-bold mb-8">Arquitetura</h1>
        
        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Visão Geral</h2>
          <pre className="bg-gray-900 p-4 rounded-lg mb-6 overflow-x-auto text-sm">
{`┌──────────────────────────────────────────────────────────┐
│                    CONTROL PLANE (Go)                     │
│  API REST + Auth + WebSocket + Deploy Queue              │
│  Porta 3001                                              │
├──────────────────────────────────────────────────────────┤
│                   DEPLOY WORKER (Go)                      │
│  Worker pool: NumCPU goroutines (max 16)                 │
│  Docker SDK nativo + BuildKit streaming                  │
├──────────────────────────────────────────────────────────┤
│                    DATA PLANE (Rust)                      │
│  Reverse proxy de alta performance para apps deployados  │
│  Rate limiting + TLS + WebSocket proxy                   │
│  Porta 3080                                              │
├──────────────────────────────────────────────────────────┤
│                   DASHBOARD (Next.js)                     │
│  Frontend SPA — interface do usuário                     │
│  Porta 3000                                              │
└──────────────────────────────────────────────────────────┘`}
          </pre>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Componentes</h2>
          
          <h3 className="text-xl font-medium mb-3">Control Plane (Go)</h3>
          <p className="text-gray-300 mb-4">A API principal que gerencia autenticação, projetos, deploys e webhooks.</p>
          
          <h3 className="text-xl font-medium mb-3">Deploy Worker (Go)</h3>
          <p className="text-gray-300 mb-4">Worker que processa builds e deploys com pool de goroutines concorrentes.</p>
          
          <h3 className="text-xl font-medium mb-3">Data Plane (Rust)</h3>
          <p className="text-gray-300 mb-4">Reverse proxy de alta performance com rate limiting e TLS.</p>
          
          <h3 className="text-xl font-medium mb-3">Dashboard (Next.js)</h3>
          <p className="text-gray-300">Interface do usuário para gerenciar projetos e deploys.</p>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Fluxo de Deploy</h2>
          <ol className="list-decimal list-inside text-gray-300 space-y-2">
            <li>Push para GitHub</li>
            <li>Webhook recebe evento</li>
            <li>API enfileira job no Redis</li>
            <li>Worker consome job</li>
            <li>Git clone do repositório</li>
            <li>Docker build da imagem</li>
            <li>Container é iniciado</li>
            <li>Proxy roteia tráfego</li>
            <li>App está online!</li>
          </ol>
        </section>
      </div>
    </main>
  )
}
