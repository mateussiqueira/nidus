import Link from 'next/link'

export default function QuickstartEN() {
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
        
        <h1 className="text-4xl font-bold mb-8">Quick Start</h1>
        
        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Prerequisites</h2>
          <ul className="list-disc list-inside text-gray-300 space-y-2">
            <li>Docker and Docker Compose</li>
            <li>Git</li>
            <li>(Optional) Node.js 18+ for local development</li>
          </ul>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Installation with Docker (Recommended)</h2>
          
          <h3 className="text-xl font-medium mb-3">1. Clone the repository</h3>
          <pre className="bg-gray-900 p-4 rounded-lg mb-6 overflow-x-auto">
            <code>git clone https://github.com/mateussiqueira/nidus.git{'\n'}cd nidus</code>
          </pre>

          <h3 className="text-xl font-medium mb-3">2. Configure environment variables</h3>
          <pre className="bg-gray-900 p-4 rounded-lg mb-6 overflow-x-auto">
            <code>cp .env.example .env</code>
          </pre>

          <h3 className="text-xl font-medium mb-3">3. Start services</h3>
          <pre className="bg-gray-900 p-4 rounded-lg mb-6 overflow-x-auto">
            <code>docker compose up -d</code>
          </pre>

          <h3 className="text-xl font-medium mb-3">4. Verify it&apos;s running</h3>
          <pre className="bg-gray-900 p-4 rounded-lg mb-6 overflow-x-auto">
            <code>curl http://localhost:3001/health</code>
          </pre>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Access</h2>
          <ul className="list-disc list-inside text-gray-300 space-y-2">
            <li><strong>Dashboard:</strong> http://localhost:3000</li>
            <li><strong>API:</strong> http://localhost:3001</li>
            <li><strong>Proxy:</strong> http://localhost:3080</li>
          </ul>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Default Login</h2>
          <ul className="list-disc list-inside text-gray-300 space-y-2">
            <li>Email: <code>demo@nidus.dev</code></li>
            <li>Password: <code>demo123</code></li>
          </ul>
        </section>
      </div>
    </main>
  )
}
