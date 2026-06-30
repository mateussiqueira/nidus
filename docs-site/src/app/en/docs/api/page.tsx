import Link from 'next/link'

export default function APIEN() {
  return (
    <div>
      
      <div className="container mx-auto px-4 py-16 max-w-3xl">
        
        <h1 className="text-4xl font-bold mb-8">API</h1>
        
        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Base URL</h2>
          <pre className="bg-[#111113] p-4 rounded-lg mb-6 overflow-x-auto">
            <code>http://your-server:3001</code>
          </pre>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Health Check</h2>
          <pre className="bg-[#111113] p-4 rounded-lg mb-6 overflow-x-auto">
{`GET /health

Response:
{
  "status": "ok",
  "name": "stackrun-control-plane",
  "version": "0.2.0"
}`}
          </pre>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Auth</h2>
          
          <h3 className="text-xl font-medium mb-3">Register</h3>
          <pre className="bg-[#111113] p-4 rounded-lg mb-6 overflow-x-auto text-sm">
{`POST /api/auth/register

{
  "email": "user@example.com",
  "name": "Name",
  "password": "password123"
}`}
          </pre>

          <h3 className="text-xl font-medium mb-3">Login</h3>
          <pre className="bg-[#111113] p-4 rounded-lg mb-6 overflow-x-auto text-sm">
{`POST /api/auth/login

{
  "email": "user@example.com",
  "password": "password123"
}`}
          </pre>
        </section>

        <section className="mb-12">
          <h2 className="text-2xl font-semibold mb-4">Projects</h2>
          <pre className="bg-[#111113] p-4 rounded-lg mb-6 overflow-x-auto text-sm">
{`GET    /api/projects           - List
POST   /api/projects           - Create
GET    /api/projects/:id       - Details
PATCH  /api/projects/:id       - Update
DELETE /api/projects/:id       - Delete`}
          </pre>
        </section>
      </div>
</div>
  )
}
