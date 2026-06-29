import Link from 'next/link'

export default function DeploymentPT() {
  return (
    <div>
      
      <div className="container mx-auto px-4 py-16 max-w-3xl">
        
        <h1 className="text-4xl font-bold mb-8">Deploy</h1>
        
        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Via GitHub (Recomendado)</h2>
          
          <h3 className="text-xl font-medium mb-3">1. Configure o webhook</h3>
          <p className="text-[#a1a1aa] mb-4">No seu repositório GitHub, vá em Settings → Webhooks → Add webhook:</p>
          <ul className="list-disc list-inside text-[#a1a1aa] space-y-2 mb-4">
            <li><strong>Payload URL:</strong> http://seu-servidor:3001/api/webhook</li>
            <li><strong>Content type:</strong> application/json</li>
            <li><strong>Events:</strong> Just the push event</li>
          </ul>

          <h3 className="text-xl font-medium mb-3">2. Crie o projeto</h3>
          <pre className="bg-[#111113] p-4 rounded-lg mb-6 overflow-x-auto text-sm">
{`curl -X POST http://seu-servidor:3001/api/projects \\
  -H "Authorization: Bearer SEU_TOKEN" \\
  -H "Content-Type: application/json" \\
  -d '{"name":"meu-app","framework":"next","repoUrl":"https://github.com/usuario/repo"}'`}
          </pre>

          <h3 className="text-xl font-medium mb-3">3. Faça push</h3>
          <pre className="bg-[#111113] p-4 rounded-lg mb-6 overflow-x-auto">
            <code>git push origin main</code>
          </pre>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Via CLI</h2>
          <pre className="bg-[#111113] p-4 rounded-lg mb-6 overflow-x-auto">
{`npm install -g nidus
nidus login --url http://seu-servidor:3001
cd meu-projeto
nidus deploy`}
          </pre>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Status do Deploy</h2>
          <table className="w-full text-left">
            <thead>
              <tr className="border-b border-[#1f1f23]">
                <th className="py-2">Status</th>
                <th className="py-2">Descrição</th>
              </tr>
            </thead>
            <tbody className="text-[#a1a1aa]">
              <tr className="border-b border-[#1f1f23]">
                <td className="py-2"><code>pending</code></td>
                <td className="py-2">Na fila de processamento</td>
              </tr>
              <tr className="border-b border-[#1f1f23]">
                <td className="py-2"><code>building</code></td>
                <td className="py-2">Fazendo build da imagem Docker</td>
              </tr>
              <tr className="border-b border-[#1f1f23]">
                <td className="py-2"><code>deploying</code></td>
                <td className="py-2">Iniciando container</td>
              </tr>
              <tr className="border-b border-[#1f1f23]">
                <td className="py-2"><code>ready</code></td>
                <td className="py-2">App está online e respondendo</td>
              </tr>
              <tr>
                <td className="py-2"><code>failed</code></td>
                <td className="py-2">Erro durante o processo</td>
              </tr>
            </tbody>
          </table>
        </section>
      </div>
</div>
  )
}
