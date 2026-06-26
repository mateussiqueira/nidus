import Link from 'next/link'

export default function APIPT() {
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
        
        <h1 className="text-4xl font-bold mb-8">API</h1>
        
        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Base URL</h2>
          <pre className="bg-gray-900 p-4 rounded-lg mb-6 overflow-x-auto">
            <code>http://seu-servidor:3001</code>
          </pre>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Health Check</h2>
          <pre className="bg-gray-900 p-4 rounded-lg mb-6 overflow-x-auto">
{`GET /health

Response:
{
  "status": "ok",
  "name": "nidus-control-plane",
  "version": "0.2.0"
}`}
          </pre>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Auth</h2>
          
          <h3 className="text-xl font-medium mb-3">Register</h3>
          <pre className="bg-gray-900 p-4 rounded-lg mb-6 overflow-x-auto text-sm">
{`POST /api/auth/register

{
  "email": "user@example.com",
  "name": "Nome",
  "password": "senha123"
}`}
          </pre>

          <h3 className="text-xl font-medium mb-3">Login</h3>
          <pre className="bg-gray-900 p-4 rounded-lg mb-6 overflow-x-auto text-sm">
{`POST /api/auth/login

{
  "email": "user@example.com",
  "password": "senha123"
}`}
          </pre>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Projetos</h2>
          <pre className="bg-gray-900 p-4 rounded-lg mb-6 overflow-x-auto text-sm">
{`GET    /api/projects           - Listar
POST   /api/projects           - Criar
GET    /api/projects/:id       - Detalhes
PATCH  /api/projects/:id       - Atualizar
DELETE /api/projects/:id       - Deletar`}
          </pre>
        </section>
      </div>
    </main>
  )
}
