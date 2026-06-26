export default function Home() {
  return (
    <main className="min-h-screen bg-gray-950 text-white">
      <div className="container mx-auto px-4 py-16">
        <h1 className="text-4xl font-bold mb-8">Nidus Docs</h1>
        <p className="text-xl text-gray-400 mb-8">
          Self-hosted deploy platform. Think Vercel that runs on your own machine.
        </p>
        <div className="flex gap-4">
          <a
            href="/pt"
            className="bg-blue-600 hover:bg-blue-700 px-6 py-3 rounded-lg font-medium transition"
          >
            Português
          </a>
          <a
            href="/en"
            className="bg-gray-800 hover:bg-gray-700 px-6 py-3 rounded-lg font-medium transition"
          >
            English
          </a>
        </div>
      </div>
    </main>
  )
}
