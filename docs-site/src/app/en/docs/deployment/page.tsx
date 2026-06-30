import Link from 'next/link'

export default function DeploymentEN() {
  return (
    <div>
      
      <div className="container mx-auto px-4 py-16 max-w-3xl">
        
        <h1 className="text-4xl font-bold mb-8">Deployment</h1>
        
        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Via GitHub (Recommended)</h2>
          
          <h3 className="text-xl font-medium mb-3">1. Set up the webhook</h3>
          <p className="text-[#a1a1aa] mb-4">In your GitHub repository, go to Settings → Webhooks → Add webhook:</p>
          <ul className="list-disc list-inside text-[#a1a1aa] space-y-2 mb-4">
            <li><strong>Payload URL:</strong> http://your-server:3001/api/webhook</li>
            <li><strong>Content type:</strong> application/json</li>
            <li><strong>Events:</strong> Just the push event</li>
          </ul>

          <h3 className="text-xl font-medium mb-3">2. Create the project</h3>
          <pre className="bg-[#111113] p-4 rounded-lg mb-6 overflow-x-auto text-sm">
{`curl -X POST http://your-server:3001/api/projects \\
  -H "Authorization: Bearer YOUR_TOKEN" \\
  -H "Content-Type: application/json" \\
  -d '{"name":"my-app","framework":"next","repoUrl":"https://github.com/user/repo"}'`}
          </pre>

          <h3 className="text-xl font-medium mb-3">3. Push</h3>
          <pre className="bg-[#111113] p-4 rounded-lg mb-6 overflow-x-auto">
            <code>git push origin main</code>
          </pre>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Via CLI</h2>
          <pre className="bg-[#111113] p-4 rounded-lg mb-6 overflow-x-auto">
{`npm install -g stackrun
stackrun login --url http://your-server:3001
cd my-project
stackrun deploy`}
          </pre>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Deploy Status</h2>
          <table className="w-full text-left">
            <thead>
              <tr className="border-b border-[#1f1f23]">
                <th className="py-2">Status</th>
                <th className="py-2">Description</th>
              </tr>
            </thead>
            <tbody className="text-[#a1a1aa]">
              <tr className="border-b border-[#1f1f23]">
                <td className="py-2"><code>pending</code></td>
                <td className="py-2">In the processing queue</td>
              </tr>
              <tr className="border-b border-[#1f1f23]">
                <td className="py-2"><code>building</code></td>
                <td className="py-2">Building Docker image</td>
              </tr>
              <tr className="border-b border-[#1f1f23]">
                <td className="py-2"><code>deploying</code></td>
                <td className="py-2">Starting container</td>
              </tr>
              <tr className="border-b border-[#1f1f23]">
                <td className="py-2"><code>ready</code></td>
                <td className="py-2">App is live and responding</td>
              </tr>
              <tr>
                <td className="py-2"><code>failed</code></td>
                <td className="py-2">Error during the process</td>
              </tr>
            </tbody>
          </table>
        </section>
      </div>
</div>
  )
}
