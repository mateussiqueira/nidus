import Link from 'next/link'

export default function FAQEN() {
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
        
        <h1 className="text-4xl font-bold mb-8">FAQ</h1>
        
        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">What is Nidus?</h2>
          <p className="text-gray-300">Nidus is a self-hosted deploy platform, similar to Vercel but runs on your own server.</p>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Is Nidus free?</h2>
          <p className="text-gray-300">Yes! Nidus is open-source under the MIT license. You only pay for the server where it runs.</p>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">What frameworks are supported?</h2>
          <ul className="list-disc list-inside text-gray-300 space-y-2">
            <li>Next.js</li>
            <li>React (Vite)</li>
            <li>Vue.js (Vite)</li>
            <li>Svelte</li>
            <li>Node.js (Express, Fastify, NestJS)</li>
            <li>Go</li>
            <li>Docker (any language)</li>
          </ul>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">How to deploy?</h2>
          <p className="text-gray-300">Three ways:</p>
          <ol className="list-decimal list-inside text-gray-300 space-y-2 mt-2">
            <li><strong>GitHub</strong> — Set up a webhook and push</li>
            <li><strong>CLI</strong> — Use <code>nidus deploy</code></li>
            <li><strong>API</strong> — Call the REST endpoint</li>
          </ol>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">How to set up a custom domain?</h2>
          <p className="text-gray-300">Point the domain DNS to the server IP and configure the domain in the project.</p>
        </section>
      </div>
    </main>
  )
}
