import Link from 'next/link'

const docs = [
  { title: 'Quick Start', href: '/en/docs/quickstart', description: 'Install and run Nimbus in minutes' },
  { title: 'Architecture', href: '/en/docs/architecture', description: 'How Nimbus works under the hood' },
  { title: 'Deployment', href: '/en/docs/deployment', description: 'Deploy your apps via Git, CLI, or API' },
  { title: 'Configuration', href: '/en/docs/configuration', description: 'Environment variables and settings' },
  { title: 'CLI', href: '/en/docs/cli', description: 'Command-line interface reference' },
  { title: 'API Reference', href: '/en/docs/api', description: 'Complete REST API documentation' },
  { title: 'FAQ', href: '/en/docs/faq', description: 'Frequently asked questions' },
]

export default function DocsIndex() {
  return (
    <div>
      <h1 className="text-4xl font-bold mb-4 text-[#fafafa]">Documentation</h1>
      <p className="text-lg text-[#a1a1aa] mb-10">
        Everything you need to deploy and manage apps with Nimbus.
      </p>
      <div className="grid md:grid-cols-2 gap-4">
        {docs.map((doc) => (
          <Link
            key={doc.href}
            href={doc.href}
            className="block p-6 bg-[#111113] rounded-xl border border-[#1f1f23] hover:border-[#22c55e]/40 hover:-translate-y-0.5 transition-all no-underline"
          >
            <h2 className="text-lg font-semibold text-[#fafafa] mb-2">{doc.title}</h2>
            <p className="text-sm text-[#a1a1aa]">{doc.description}</p>
          </Link>
        ))}
      </div>
    </div>
  )
}
