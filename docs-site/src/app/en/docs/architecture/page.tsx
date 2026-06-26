import Link from 'next/link'

export default function ArchitectureEN() {
  return (
    <main className="min-h-screen bg-gray-950 text-white">
      <nav className="border-b border-gray-800 p-4">
        <div className="container mx-auto flex justify-between items-center">
          <Link href="/" className="text-xl font-bold">Nidus Docs</Link>
          <div className="flex gap-4">
            <Link href="/pt" className="text-gray-400 hover:text-white">PT</Link>
            <Link href="/en" className="text-blue-400">EN</Link>
          </div>
        </div>
      </nav>
      
      <div className="container mx-auto px-4 py-16 max-w-3xl">
        <Link href="/en" className="text-blue-400 hover:underline mb-8 block">&larr; Back</Link>
        
        <h1 className="text-4xl font-bold mb-8">Architecture</h1>
        
        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Overview</h2>
          <pre className="bg-gray-900 p-4 rounded-lg mb-6 overflow-x-auto text-sm">
{`┌──────────────────────────────────────────────────────────┐
│                    CONTROL PLANE (Go)                     │
│  API REST + Auth + WebSocket + Deploy Queue              │
│  Port 3001                                               │
├──────────────────────────────────────────────────────────┤
│                   DEPLOY WORKER (Go)                      │
│  Worker pool: NumCPU goroutines (max 16)                 │
│  Native Docker SDK + BuildKit streaming                  │
├──────────────────────────────────────────────────────────┤
│                    DATA PLANE (Rust)                      │
│  High-performance reverse proxy for deployed apps        │
│  Rate limiting + TLS + WebSocket proxy                   │
│  Port 3080                                               │
├──────────────────────────────────────────────────────────┤
│                   DASHBOARD (Next.js)                     │
│  SPA Frontend — user interface                           │
│  Port 3000                                               │
└──────────────────────────────────────────────────────────┘`}
          </pre>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Components</h2>
          
          <h3 className="text-xl font-medium mb-3">Control Plane (Go)</h3>
          <p className="text-gray-300 mb-4">The main API that manages authentication, projects, deploys, and webhooks.</p>
          
          <h3 className="text-xl font-medium mb-3">Deploy Worker (Go)</h3>
          <p className="text-gray-300 mb-4">Worker that processes builds and deploys with concurrent goroutine pool.</p>
          
          <h3 className="text-xl font-medium mb-3">Data Plane (Rust)</h3>
          <p className="text-gray-300 mb-4">High-performance reverse proxy with rate limiting and TLS.</p>
          
          <h3 className="text-xl font-medium mb-3">Dashboard (Next.js)</h3>
          <p className="text-gray-300">User interface to manage projects and deployments.</p>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Deploy Flow</h2>
          <ol className="list-decimal list-inside text-gray-300 space-y-2">
            <li>Push to GitHub</li>
            <li>Webhook receives event</li>
            <li>API queues job in Redis</li>
            <li>Worker consumes job</li>
            <li>Git clone repository</li>
            <li>Docker build image</li>
            <li>Container is started</li>
            <li>Proxy routes traffic</li>
            <li>App is live!</li>
          </ol>
        </section>
      </div>
    </main>
  )
}
