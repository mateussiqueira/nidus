import Link from 'next/link'

const docs = [
  { title: 'Primeiros Passos', href: '/pt/docs/quickstart', description: 'Instale e rode o Nimbus em minutos' },
  { title: 'Arquitetura', href: '/pt/docs/architecture', description: 'Como o Nimbus funciona por dentro' },
  { title: 'Deploy', href: '/pt/docs/deployment', description: 'Faça deploy via Git, CLI ou API' },
  { title: 'Configuração', href: '/pt/docs/configuration', description: 'Variáveis de ambiente e ajustes' },
  { title: 'CLI', href: '/pt/docs/cli', description: 'Referência da interface de linha de comando' },
  { title: 'API', href: '/pt/docs/api', description: 'Documentação completa da API REST' },
  { title: 'FAQ', href: '/pt/docs/faq', description: 'Perguntas frequentes' },
]

export default function DocsIndex() {
  return (
    <div>
      <h1 className="text-4xl font-bold mb-4 text-[#fafafa]">Documentação</h1>
      <p className="text-lg text-[#a1a1aa] mb-10">
        Tudo que você precisa para fazer deploy e gerenciar apps com o Nimbus.
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
