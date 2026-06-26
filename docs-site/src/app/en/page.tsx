import Link from 'next/link'

const docs = [
  { title: 'Home', href: '/en', description: 'Nidus overview' },
  { title: 'Quick Start', href: '/en/docs/quickstart', description: 'Install and run Nidus in minutes' },
  { title: 'Architecture', href: '/en/docs/architecture', description: 'Understand how Nidus works' },
  { title: 'Deployment', href: '/en/docs/deployment', description: 'Deploy your applications' },
  { title: 'Configuration', href: '/en/docs/configuration', description: 'Environment variables and settings' },
  { title: 'CLI', href: '/en/docs/cli', description: 'Use the command line' },
  { title: 'API', href: '/en/docs/api', description: 'Complete REST API reference' },
  { title: 'FAQ', href: '/en/docs/faq', description: 'Frequently asked questions' },
]

export default function ENHome() {
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
      
      <div className="container mx-auto px-4 py-16">
        <h1 className="text-4xl font-bold mb-4">Nidus</h1>
        <p className="text-xl text-gray-400 mb-8">
          Self-hosted deploy platform. Think Vercel that runs on your own machine.
        </p>
        
        <div className="grid md:grid-cols-2 gap-6">
          {docs.map((doc) => (
            <Link
              key={doc.href}
              href={doc.href}
              className="block p-6 bg-gray-900 rounded-lg border border-gray-800 hover:border-blue-500 transition"
            >
              <h2 className="text-xl font-semibold mb-2">{doc.title}</h2>
              <p className="text-gray-400">{doc.description}</p>
            </Link>
          ))}
        </div>
      </div>
    </main>
  )
}
