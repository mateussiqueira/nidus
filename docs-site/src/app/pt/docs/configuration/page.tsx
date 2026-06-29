import Link from 'next/link'

export default function ConfigurationPT() {
  return (
    <div>
      
      <div className="container mx-auto px-4 py-16 max-w-3xl">
        
        <h1 className="text-4xl font-bold mb-8">Configuração</h1>
        
        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Variáveis de Ambiente</h2>
          <p className="text-[#a1a1aa] mb-4">Copie <code>.env.example</code> para <code>.env</code> e ajuste conforme necessário.</p>
          
          <h3 className="text-xl font-medium mb-3">Banco de Dados</h3>
          <pre className="bg-[#111113] p-4 rounded-lg mb-6 overflow-x-auto text-sm">
{`DATABASE_URL=postgresql://user:password@localhost:5432/nidus`}
          </pre>

          <h3 className="text-xl font-medium mb-3">Redis</h3>
          <pre className="bg-[#111113] p-4 rounded-lg mb-6 overflow-x-auto text-sm">
{`REDIS_URL=redis://:password@localhost:6379
REDIS_PASSWORD=sua-senha-redis`}
          </pre>

          <h3 className="text-xl font-medium mb-3">Autenticação</h3>
          <pre className="bg-[#111113] p-4 rounded-lg mb-6 overflow-x-auto text-sm">
{`JWT_SECRET=seu-jwt-secret-aqui`}
          </pre>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Docker Compose - Portas</h2>
          <table className="w-full text-left">
            <thead>
              <tr className="border-b border-[#1f1f23]">
                <th className="py-2">Serviço</th>
                <th className="py-2">Porta</th>
                <th className="py-2">Descrição</th>
              </tr>
            </thead>
            <tbody className="text-[#a1a1aa]">
              <tr className="border-b border-[#1f1f23]">
                <td className="py-2">Dashboard</td>
                <td className="py-2">3000</td>
                <td className="py-2">Interface web</td>
              </tr>
              <tr className="border-b border-[#1f1f23]">
                <td className="py-2">API</td>
                <td className="py-2">3001</td>
                <td className="py-2">Backend Go</td>
              </tr>
              <tr className="border-b border-[#1f1f23]">
                <td className="py-2">Proxy</td>
                <td className="py-2">3080</td>
                <td className="py-2">Reverse proxy</td>
              </tr>
              <tr className="border-b border-[#1f1f23]">
                <td className="py-2">PostgreSQL</td>
                <td className="py-2">5432</td>
                <td className="py-2">Banco de dados</td>
              </tr>
              <tr>
                <td className="py-2">Redis</td>
                <td className="py-2">6379</td>
                <td className="py-2">Cache/fila</td>
              </tr>
            </tbody>
          </table>
        </section>
      </div>
</div>
  )
}
