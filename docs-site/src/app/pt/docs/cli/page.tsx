import Link from 'next/link'

export default function CLIPT() {
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
        
        <h1 className="text-4xl font-bold mb-8">CLI</h1>
        
        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Instalação</h2>
          <pre className="bg-gray-900 p-4 rounded-lg mb-6 overflow-x-auto">
            <code>npm install -g nidus</code>
          </pre>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Comandos</h2>
          
          <h3 className="text-xl font-medium mb-3">Login</h3>
          <pre className="bg-gray-900 p-4 rounded-lg mb-6 overflow-x-auto">
            <code>nidus login --url http://seu-servidor:3001</code>
          </pre>

          <h3 className="text-xl font-medium mb-3">Deploy</h3>
          <pre className="bg-gray-900 p-4 rounded-lg mb-6 overflow-x-auto">
{`cd meu-projeto
nidus deploy`}
          </pre>

          <h3 className="text-xl font-medium mb-3">Status</h3>
          <pre className="bg-gray-900 p-4 rounded-lg mb-6 overflow-x-auto">
            <code>nidus status</code>
          </pre>

          <h3 className="text-xl font-medium mb-3">Projects</h3>
          <pre className="bg-gray-900 p-4 rounded-lg mb-6 overflow-x-auto">
{`nidus projects list
nidus projects create --name "Meu App"
nidus projects delete --id abc123`}
          </pre>
        </section>
      </div>
    </main>
  )
}
