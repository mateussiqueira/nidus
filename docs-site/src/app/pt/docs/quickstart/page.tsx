import Link from 'next/link'

export default function QuickstartPT() {
  return (
    <div>
      
      <div className="container mx-auto px-4 py-16 max-w-3xl">
        
        <h1 className="text-4xl font-bold mb-8">Primeiros Passos</h1>
        
        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Pré-requisitos</h2>
          
          <div className="bg-[#111113] p-4 rounded-lg mb-6 border border-[#1f1f23]">
            <h3 className="text-lg font-medium mb-3">Requisitos de Sistema</h3>
            <div className="grid md:grid-cols-2 gap-4">
              <div className="bg-[#0a0a0b] p-4 rounded-lg">
                <h4 className="font-semibold text-[#22c55e] mb-2">Versão Lite</h4>
                <ul className="text-[#a1a1aa] text-sm space-y-1">
                  <li>• RAM: <strong>512MB</strong> mínimo</li>
                  <li>• Disco: 2GB</li>
                  <li>• API + Worker apenas</li>
                </ul>
              </div>
              <div className="bg-[#0a0a0b] p-4 rounded-lg">
                <h4 className="font-semibold text-[#22c55e] mb-2">Versão Completa</h4>
                <ul className="text-[#a1a1aa] text-sm space-y-1">
                  <li>• RAM: <strong>2GB</strong> mínimo</li>
                  <li>• Disco: 10GB</li>
                  <li>• Todos os componentes</li>
                </ul>
              </div>
            </div>
          </div>

          <h3 className="text-xl font-medium mb-3">Software Necessário</h3>
          <ul className="list-disc list-inside text-[#a1a1aa] space-y-2">
            <li>Docker e Docker Compose</li>
            <li>Git</li>
            <li>(Opcional) Node.js 18+ para develop local</li>
          </ul>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Instalação com Docker (Recomendado)</h2>
          
          <h3 className="text-xl font-medium mb-3">1. Clone o repositório</h3>
          <pre className="bg-[#111113] p-4 rounded-lg mb-6 overflow-x-auto">
            <code>git clone https://github.com/mateussiqueira/stackrun.git{'\n'}cd nidus</code>
          </pre>

          <h3 className="text-xl font-medium mb-3">2. Configure as variáveis de ambiente</h3>
          <pre className="bg-[#111113] p-4 rounded-lg mb-6 overflow-x-auto">
            <code>cp .env.example .env</code>
          </pre>

          <h3 className="text-xl font-medium mb-3">3. Inicie os serviços</h3>
          <pre className="bg-[#111113] p-4 rounded-lg mb-6 overflow-x-auto">
            <code>docker compose up -d</code>
          </pre>

          <h3 className="text-xl font-medium mb-3">4. Verifique se está rodando</h3>
          <pre className="bg-[#111113] p-4 rounded-lg mb-6 overflow-x-auto">
            <code>curl http://localhost:3001/health</code>
          </pre>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Acesse</h2>
          <ul className="list-disc list-inside text-[#a1a1aa] space-y-2">
            <li><strong>Dashboard:</strong> http://localhost:3000</li>
            <li><strong>API:</strong> http://localhost:3001</li>
            <li><strong>Proxy:</strong> http://localhost:3080</li>
          </ul>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Login Padrão</h2>
          <ul className="list-disc list-inside text-[#a1a1aa] space-y-2">
            <li>Email: <code>demo@stackrun.dev</code></li>
            <li>Senha: <code>demo123</code></li>
          </ul>
        </section>
      </div>
</div>
  )
}
