import Link from 'next/link'

export default function QuickstartEN() {
  return (
    <div>
      
      <div className="container mx-auto px-4 py-16 max-w-3xl">
        
        <h1 className="text-4xl font-bold mb-8">Quick Start</h1>
        
        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Prerequisites</h2>
          
          <div className="bg-[#111113] p-4 rounded-lg mb-6 border border-[#1f1f23]">
            <h3 className="text-lg font-medium mb-3">System Requirements</h3>
            <div className="grid md:grid-cols-2 gap-4">
              <div className="bg-[#0a0a0b] p-4 rounded-lg">
                <h4 className="font-semibold text-[#22c55e] mb-2">Lite Version</h4>
                <ul className="text-[#a1a1aa] text-sm space-y-1">
                  <li>• RAM: <strong>512MB</strong> minimum</li>
                  <li>• Disk: 2GB</li>
                  <li>• API + Worker only</li>
                </ul>
              </div>
              <div className="bg-[#0a0a0b] p-4 rounded-lg">
                <h4 className="font-semibold text-[#22c55e] mb-2">Full Version</h4>
                <ul className="text-[#a1a1aa] text-sm space-y-1">
                  <li>• RAM: <strong>2GB</strong> minimum</li>
                  <li>• Disk: 10GB</li>
                  <li>• All components</li>
                </ul>
              </div>
            </div>
          </div>

          <h3 className="text-xl font-medium mb-3">Required Software</h3>
          <ul className="list-disc list-inside text-[#a1a1aa] space-y-2">
            <li>Docker and Docker Compose</li>
            <li>Git</li>
            <li>(Optional) Node.js 18+ for local development</li>
          </ul>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Installation with Docker (Recommended)</h2>
          
          <h3 className="text-xl font-medium mb-3">1. Clone the repository</h3>
          <pre className="bg-[#111113] p-4 rounded-lg mb-6 overflow-x-auto">
            <code>git clone https://github.com/mateussiqueira/nidus.git{'\n'}cd nidus</code>
          </pre>

          <h3 className="text-xl font-medium mb-3">2. Configure environment variables</h3>
          <pre className="bg-[#111113] p-4 rounded-lg mb-6 overflow-x-auto">
            <code>cp .env.example .env</code>
          </pre>

          <h3 className="text-xl font-medium mb-3">3. Start services</h3>
          <pre className="bg-[#111113] p-4 rounded-lg mb-6 overflow-x-auto">
            <code>docker compose up -d</code>
          </pre>

          <h3 className="text-xl font-medium mb-3">4. Verify it&apos;s running</h3>
          <pre className="bg-[#111113] p-4 rounded-lg mb-6 overflow-x-auto">
            <code>curl http://localhost:3001/health</code>
          </pre>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Access</h2>
          <ul className="list-disc list-inside text-[#a1a1aa] space-y-2">
            <li><strong>Dashboard:</strong> http://localhost:3000</li>
            <li><strong>API:</strong> http://localhost:3001</li>
            <li><strong>Proxy:</strong> http://localhost:3080</li>
          </ul>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Default Login</h2>
          <ul className="list-disc list-inside text-[#a1a1aa] space-y-2">
            <li>Email: <code>demo@nidus.dev</code></li>
            <li>Password: <code>demo123</code></li>
          </ul>
        </section>
      </div>
</div>
  )
}
