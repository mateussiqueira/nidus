import Link from "next/link"

export default function Home() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center p-8">
      <main className="flex flex-col items-center gap-8">
        <div className="text-center">
          <h1 className="text-5xl font-bold tracking-tight text-nidus-400">
            Nidus
          </h1>
          <p className="mt-4 text-lg text-zinc-400">
            Sua PaaS pessoal — deploy full-stack com suporte nativo a Dart & Vaden
          </p>
        </div>

        <div className="flex gap-4">
          <Link href="/login" className="btn btn-primary px-8 py-3 bg-green-500 text-black rounded-lg font-semibold hover:bg-green-600 transition">
            Acessar Dashboard →
          </Link>
        </div>

        <div className="mt-8 grid grid-cols-1 gap-4 sm:grid-cols-3">
          <Card icon="🚀" title="Deploy" description="Frontends, backends e APIs em segundos" />
          <Card icon="🗄️" title="Database" description="PostgreSQL gerenciado por projeto" />
          <Card icon="🔐" title="Auth" description="Autenticação pronta com JWT + OAuth" />
        </div>

        <div className="mt-12 flex gap-4">
          <span className="rounded-full bg-zinc-800 px-4 py-2 text-sm text-zinc-400">
            Seedbox — Fase 1
          </span>
        </div>
      </main>
    </div>
  )
}

function Card({ icon, title, description }: { icon: string; title: string; description: string }) {
  return (
    <div className="rounded-lg border border-zinc-800 bg-zinc-900/50 p-6 backdrop-blur-sm transition hover:border-zinc-700">
      <div className="mb-3 text-2xl">{icon}</div>
      <h2 className="mb-2 font-semibold text-zinc-100">{title}</h2>
      <p className="text-sm text-zinc-400">{description}</p>
    </div>
  )
}
