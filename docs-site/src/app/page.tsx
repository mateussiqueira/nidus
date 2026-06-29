export default function Home() {
  return (
    <main className="min-h-screen bg-[#0a0a0b] text-[#fafafa]" style={{ fontFamily: 'Inter, system-ui, sans-serif' }}>
      {/* Nav */}
      <header className="fixed top-0 w-full z-50 bg-[#0a0a0b]/90 backdrop-blur-xl border-b border-[#1f1f23]">
        <nav className="max-w-6xl mx-auto px-6 py-4 flex items-center justify-between">
          <a href="/" className="flex items-center gap-2.5 font-bold text-lg no-underline text-[#fafafa]">
            <img src="/logo.png" alt="Nimbus" className="h-8" />
            Nimbus
          </a>
          <div className="flex items-center gap-6">
            <a href="/en/docs" className="text-sm text-[#a1a1aa] hover:text-white no-underline font-medium transition">Docs</a>
            <a href="https://github.com/mateussiqueira/nidus" target="_blank" className="text-sm text-[#a1a1aa] hover:text-white no-underline font-medium transition">GitHub</a>
            <a href="/en/docs/quickstart" className="inline-flex items-center gap-2 px-5 py-2.5 bg-[#22c55e] text-black rounded-lg font-semibold text-sm no-underline hover:bg-[#16a34a] transition">Get Started</a>
          </div>
        </nav>
      </header>

      {/* Hero */}
      <section className="pt-40 pb-20 px-6 max-w-6xl mx-auto text-center">
        <span className="inline-block px-4 py-1.5 bg-[#22c55e]/15 text-[#22c55e] rounded-full text-sm font-semibold mb-6 border border-[#22c55e]/20">
          Open Source PaaS
        </span>
        <h1 className="text-5xl md:text-6xl font-extrabold leading-tight tracking-tight mb-6">
          Deploy like Vercel.<br/>
          <span className="bg-gradient-to-r from-[#22c55e] to-[#4ade80] text-transparent bg-clip-text">Run on your own server.</span>
        </h1>
        <p className="text-lg text-[#a1a1aa] max-w-xl mx-auto mb-10">
          Self-hosted platform for deploying apps, databases, and domains. 
          Git push to production in seconds. No vendor lock-in.
        </p>
        <div className="flex gap-3 justify-center flex-wrap">
          <a href="/en/docs/quickstart" className="inline-flex items-center gap-2 px-7 py-3.5 bg-[#22c55e] text-black rounded-lg font-semibold text-base no-underline hover:bg-[#16a34a] transition">
            Start Deploying
          </a>
          <a href="https://github.com/mateussiqueira/nidus" target="_blank" className="inline-flex items-center gap-2 px-7 py-3.5 border border-[#1f1f23] text-white rounded-lg font-medium text-base no-underline hover:bg-[#111113] transition">
            View on GitHub
          </a>
        </div>
      </section>

      {/* Stats */}
      <section className="py-16 px-6 max-w-6xl mx-auto">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-8 text-center">
          {[
            { value: 'Go', label: 'Core Language' },
            { value: '16MB', label: 'API RAM Idle' },
            { value: '2ms', label: 'Avg Response' },
            { value: 'MIT', label: 'License' },
          ].map((s) => (
            <div key={s.label}>
              <div className="text-4xl font-extrabold bg-gradient-to-r from-[#22c55e] to-[#4ade80] text-transparent bg-clip-text mb-2">
                {s.value}
              </div>
              <div className="text-sm text-[#a1a1aa]">{s.label}</div>
            </div>
          ))}
        </div>
      </section>

      {/* Features */}
      <section className="py-20 px-6 max-w-6xl mx-auto">
        <h2 className="text-3xl font-bold text-center mb-4">Everything you need</h2>
        <p className="text-center text-[#a1a1aa] mb-16 max-w-lg mx-auto">
          Managed infrastructure for your apps — from databases to domains, monitoring to email.
        </p>
        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {[
            { icon: 'R', title: 'Git Deploy', desc: 'Push to GitHub. We build, containerize, and serve automatically.' },
            { icon: 'D', title: 'Managed DBs', desc: 'PostgreSQL databases provisioned instantly with connection strings.' },
            { icon: 'M', title: 'Built-in Mail', desc: 'Transactional email API with templates. MCP server for AI agents.' },
            { icon: 'G', title: 'Monitoring', desc: 'Prometheus + Grafana dashboards with per-project CPU/Memory metrics.' },
            { icon: 'C', title: 'Custom Domains', desc: 'Add your own domain, SSL auto-provisioned via Caddy.' },
            { icon: 'I', title: 'Instant Rollback', desc: 'Roll back to any previous deployment in one click.' },
          ].map((f) => (
            <div key={f.title} className="bg-[#111113] border border-[#1f1f23] rounded-xl p-8 text-center hover:border-[#22c55e]/50 hover:-translate-y-0.5 transition-all">
              <div className="w-12 h-12 bg-[#22c55e]/15 rounded-xl flex items-center justify-center mx-auto mb-5 text-lg font-bold text-[#22c55e]">
                {f.icon}
              </div>
              <h3 className="text-lg font-semibold mb-2">{f.title}</h3>
              <p className="text-sm text-[#a1a1aa] leading-relaxed">{f.desc}</p>
            </div>
          ))}
        </div>
      </section>

      {/* How it works */}
      <section className="py-20 px-6 max-w-6xl mx-auto text-center">
        <h2 className="text-3xl font-bold mb-12">How it works</h2>
        <div className="grid sm:grid-cols-3 gap-8">
          {[
            { step: '1', title: 'Connect Git', desc: 'Link your GitHub repo. We handle webhooks, builds, and Docker containers.' },
            { step: '2', title: 'Push to Deploy', desc: 'git push triggers an automatic build. Your app is live in seconds.' },
            { step: '3', title: 'Scale & Monitor', desc: 'Add domains, databases, and monitor everything from the dashboard.' },
          ].map((s) => (
            <div key={s.step}>
              <div className="w-12 h-12 bg-[#22c55e] text-black rounded-full flex items-center justify-center mx-auto mb-5 text-lg font-bold">{s.step}</div>
              <h3 className="text-lg font-semibold mb-2">{s.title}</h3>
              <p className="text-sm text-[#a1a1aa]">{s.desc}</p>
            </div>
          ))}
        </div>
      </section>

      {/* CTA */}
      <section className="py-20 px-6 max-w-6xl mx-auto text-center">
        <div className="bg-[#111113] border border-[#1f1f23] rounded-2xl p-12 md:p-16">
          <h2 className="text-3xl font-bold mb-4">Ready to deploy?</h2>
          <p className="text-[#a1a1aa] mb-8 max-w-md mx-auto">
            Self-host on your VPS, homelab, or Raspberry Pi. Takes 2 minutes.
          </p>
          <div className="flex gap-3 justify-center flex-wrap">
            <a href="/en/docs/quickstart" className="inline-flex items-center gap-2 px-7 py-3.5 bg-[#22c55e] text-black rounded-lg font-semibold text-base no-underline hover:bg-[#16a34a] transition">
              Quick Start
            </a>
            <a href="/en/docs" className="inline-flex items-center gap-2 px-7 py-3.5 border border-[#1f1f23] text-white rounded-lg font-medium text-base no-underline hover:bg-[#111113] transition">
              Read the Docs
            </a>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-[#1f1f23] py-8 px-6 text-center text-sm text-[#a1a1aa]">
        <p>Nimbus — Open Source PaaS. MIT License.</p>
      </footer>
    </main>
  )
}
