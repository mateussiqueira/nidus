import Link from 'next/link'

export default function CLIEN() {
  return (
    <div>
      
      <div className="container mx-auto px-4 py-16 max-w-3xl">
        
        <h1 className="text-4xl font-bold mb-8">CLI</h1>
        
        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Installation</h2>
          <pre className="bg-[#111113] p-4 rounded-lg mb-6 overflow-x-auto">
            <code>npm install -g nidus</code>
          </pre>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Commands</h2>
          
          <h3 className="text-xl font-medium mb-3">Login</h3>
          <pre className="bg-[#111113] p-4 rounded-lg mb-6 overflow-x-auto">
            <code>nidus login --url http://your-server:3001</code>
          </pre>

          <h3 className="text-xl font-medium mb-3">Deploy</h3>
          <pre className="bg-[#111113] p-4 rounded-lg mb-6 overflow-x-auto">
{`cd my-project
nidus deploy`}
          </pre>

          <h3 className="text-xl font-medium mb-3">Status</h3>
          <pre className="bg-[#111113] p-4 rounded-lg mb-6 overflow-x-auto">
            <code>nidus status</code>
          </pre>

          <h3 className="text-xl font-medium mb-3">Projects</h3>
          <pre className="bg-[#111113] p-4 rounded-lg mb-6 overflow-x-auto">
{`nidus projects list
nidus projects create --name "My App"
nidus projects delete --id abc123`}
          </pre>
        </section>
      </div>
</div>
  )
}
