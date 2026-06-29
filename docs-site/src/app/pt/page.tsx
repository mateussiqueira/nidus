import Link from 'next/link'

const docs = [
  { title: 'Início', href: '/pt', description: 'Visão geral do Nidus' },
  { title: 'Primeiros Passos', href: '/pt/docs/quickstart', description: 'Instale e rode o Nidus em minutos' },
  { title: 'Arquitetura', href: '/pt/docs/architecture', description: 'Entenda como o Nidus funciona' },
  { title: 'Deploy', href: '/pt/docs/deployment', description: 'Faça deploy de suas aplicações' },
  { title: 'Configuração', href: '/pt/docs/configuration', description: 'Variáveis de ambiente e settings' },
  { title: 'CLI', href: '/pt/docs/cli', description: 'Use a linha de comando' },
  { title: 'API', href: '/pt/docs/api', description: 'Referência completa da API REST' },
  { title: 'FAQ', href: '/pt/docs/faq', description: 'Perguntas frequentes' },
]

export default function PTHome() {
  return (
    <main className="min-h-screen bg-gray-950 text-white">
      <nav className="border-b border-gray-800 p-4">
        <div className="container mx-auto flex justify-between items-center">
          <Link href="/" className="text-xl font-bold">Nimbus Docs</Link>
          <div className="flex gap-4">
            <Link href="/pt" className="text-blue-400">PT</Link>
            <Link href="/en" className="text-gray-400 hover:text-white">EN</Link>
          </div>
        </div>
      </nav>
      
      <div className="container mx-auto px-4 py-16">
        <h1 className="text-4xl font-bold mb-4">Nidus</h1>
        <p className="text-xl text-gray-400 mb-8">
          Plataforma de deploy self-hosted. Think Vercel that runs on your own machine.
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
