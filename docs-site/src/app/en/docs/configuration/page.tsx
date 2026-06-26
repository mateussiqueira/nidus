import Link from 'next/link'

export default function ConfigurationEN() {
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
        
        <h1 className="text-4xl font-bold mb-8">Configuration</h1>
        
        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Environment Variables</h2>
          <p className="text-gray-300 mb-4">Copy <code>.env.example</code> to <code>.env</code> and adjust as needed.</p>
          
          <h3 className="text-xl font-medium mb-3">Database</h3>
          <pre className="bg-gray-900 p-4 rounded-lg mb-6 overflow-x-auto text-sm">
{`DATABASE_URL=postgresql://user:password@localhost:5432/nidus`}
          </pre>

          <h3 className="text-xl font-medium mb-3">Redis</h3>
          <pre className="bg-gray-900 p-4 rounded-lg mb-6 overflow-x-auto text-sm">
{`REDIS_URL=redis://:password@localhost:6379
REDIS_PASSWORD=your-redis-password`}
          </pre>

          <h3 className="text-xl font-medium mb-3">Authentication</h3>
          <pre className="bg-gray-900 p-4 rounded-lg mb-6 overflow-x-auto text-sm">
{`JWT_SECRET=your-jwt-secret-here`}
          </pre>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Docker Compose - Ports</h2>
          <table className="w-full text-left">
            <thead>
              <tr className="border-b border-gray-800">
                <th className="py-2">Service</th>
                <th className="py-2">Port</th>
                <th className="py-2">Description</th>
              </tr>
            </thead>
            <tbody className="text-gray-300">
              <tr className="border-b border-gray-800">
                <td className="py-2">Dashboard</td>
                <td className="py-2">3000</td>
                <td className="py-2">Web interface</td>
              </tr>
              <tr className="border-b border-gray-800">
                <td className="py-2">API</td>
                <td className="py-2">3001</td>
                <td className="py-2">Go backend</td>
              </tr>
              <tr className="border-b border-gray-800">
                <td className="py-2">Proxy</td>
                <td className="py-2">3080</td>
                <td className="py-2">Reverse proxy</td>
              </tr>
              <tr className="border-b border-gray-800">
                <td className="py-2">PostgreSQL</td>
                <td className="py-2">5432</td>
                <td className="py-2">Database</td>
              </tr>
              <tr>
                <td className="py-2">Redis</td>
                <td className="py-2">6379</td>
                <td className="py-2">Cache/queue</td>
              </tr>
            </tbody>
          </table>
        </section>
      </div>
    </main>
  )
}
