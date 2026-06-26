import Link from 'next/link'

export default function FAQPT() {
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
        
        <h1 className="text-4xl font-bold mb-8">FAQ</h1>
        
        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">O que é o Nidus?</h2>
          <p className="text-gray-300">Nidus é uma plataforma de deploy self-hosted, similar ao Vercel mas roda no seu próprio servidor.</p>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">O Nidus é gratuito?</h2>
          <p className="text-gray-300">Sim! Nidus é open-source sob a licença MIT. Você paga apenas pelo servidor onde roda.</p>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Quais frameworks são suportados?</h2>
          <ul className="list-disc list-inside text-gray-300 space-y-2">
            <li>Next.js</li>
            <li>React (Vite)</li>
            <li>Vue.js (Vite)</li>
            <li>Svelte</li>
            <li>Node.js (Express, Fastify, NestJS)</li>
            <li>Go</li>
            <li>Docker (qualquer linguagem)</li>
          </ul>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Como fazer deploy?</h2>
          <p className="text-gray-300">Três formas:</p>
          <ol className="list-decimal list-inside text-gray-300 space-y-2 mt-2">
            <li><strong>GitHub</strong> — Configure um webhook e faça push</li>
            <li><strong>CLI</strong> — Use <code>nidus deploy</code></li>
            <li><strong>API</strong> — Chame o endpoint REST</li>
          </ol>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Como configurar um domínio customizado?</h2>
          <p className="text-gray-300">Aponte o DNS do domínio para o IP do servidor e configure o domínio no projeto.</p>
        </section>
      </div>
    </main>
  )
}
