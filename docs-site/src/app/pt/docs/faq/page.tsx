import Link from 'next/link'

export default function FAQPT() {
  return (
    <div>
      
      <div className="container mx-auto px-4 py-16 max-w-3xl">
        
        <h1 className="text-4xl font-bold mb-8">FAQ</h1>
        
        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">O que é o StackRun?</h2>
          <p className="text-[#a1a1aa]">StackRun é uma plataforma de deploy self-hosted, similar ao Vercel mas roda no seu próprio servidor.</p>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">O StackRun é gratuito?</h2>
          <p className="text-[#a1a1aa]">Sim! StackRun é open-source sob a licença MIT. Você paga apenas pelo servidor onde roda.</p>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Quais frameworks são suportados?</h2>
          <ul className="list-disc list-inside text-[#a1a1aa] space-y-2">
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
          <p className="text-[#a1a1aa]">Três formas:</p>
          <ol className="list-decimal list-inside text-[#a1a1aa] space-y-2 mt-2">
            <li><strong>GitHub</strong> — Configure um webhook e faça push</li>
            <li><strong>CLI</strong> — Use <code>stackrun deploy</code></li>
            <li><strong>API</strong> — Chame o endpoint REST</li>
          </ol>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Como configurar um domínio customizado?</h2>
          <p className="text-[#a1a1aa]">Aponte o DNS do domínio para o IP do servidor e configure o domínio no projeto.</p>
        </section>
      </div>
</div>
  )
}
