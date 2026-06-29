import Link from 'next/link'

const sidebar = [
  { title: 'Primeiros Passos', href: '/pt/docs/quickstart' },
  { title: 'Arquitetura', href: '/pt/docs/architecture' },
  { title: 'Deploy', href: '/pt/docs/deployment' },
  { title: 'Configuração', href: '/pt/docs/configuration' },
  { title: 'CLI', href: '/pt/docs/cli' },
  { title: 'API', href: '/pt/docs/api' },
  { title: 'FAQ', href: '/pt/docs/faq' },
]

export default function DocsLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen bg-[#0a0a0b]">
      <header className="fixed top-0 w-full z-50 bg-[#0a0a0b]/90 backdrop-blur-xl border-b border-[#1f1f23]">
        <div className="max-w-7xl mx-auto px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-6">
            <Link href="/" className="flex items-center gap-2.5 font-bold text-lg no-underline text-[#fafafa]">
              <svg width="24" height="24" viewBox="0 0 120 120" fill="none"><line x1="22" y1="90" x2="78" y2="18" stroke="#22c55e" strokeWidth="5" strokeLinecap="round"/><path d="M22 90L8 102 L0 114" stroke="#22c55e" strokeWidth="5" strokeLinecap="round"/><path d="M22 90L12 108 L6 120" stroke="#4ade80" strokeWidth="4.5" strokeLinecap="round"/><path d="M22 90L20 110 L18 120" stroke="#22c55e" strokeWidth="4" strokeLinecap="round"/><path d="M22 90L28 108 L30 120" stroke="#4ade80" strokeWidth="3.5" strokeLinecap="round"/><path d="M22 90L36 104 L42 120" stroke="#22c55e" strokeWidth="3" strokeLinecap="round"/><ellipse cx="22" cy="90" rx="6" ry="4" fill="#16a34a" transform="rotate(-55 22 90)"/><ellipse cx="26" cy="85" rx="5" ry="3" fill="#16a34a" transform="rotate(-55 26 85)"/><line x1="85" y1="15" x2="105" y2="5" stroke="#4ade80" strokeWidth="2" strokeLinecap="round" opacity="0.6"/></svg>
              Nimbus Docs
            </Link>
          </div>
          <div className="flex items-center gap-4 text-sm">
            <Link href="/en/docs" className="text-[#a1a1aa] hover:text-white no-underline transition">EN</Link>
            <Link href="/pt/docs" className="text-[#22c55e] font-medium no-underline">PT</Link>
            <a href="https://github.com/mateussiqueira/nidus" target="_blank" className="text-[#a1a1aa] hover:text-white no-underline transition">GitHub</a>
          </div>
        </div>
      </header>

      <div className="flex pt-16">
        <aside className="fixed top-16 left-0 w-56 h-[calc(100vh-4rem)] border-r border-[#1f1f23] bg-[#0a0a0b] overflow-y-auto hidden lg:block">
          <nav className="p-6">
            <Link href="/pt/docs" className="block text-sm font-semibold text-[#fafafa] mb-6 no-underline hover:text-[#22c55e] transition">Documentação</Link>
            {sidebar.map((item) => (
              <Link key={item.href} href={item.href} className="block py-1.5 text-sm text-[#a1a1aa] hover:text-white no-underline transition">
                {item.title}
              </Link>
            ))}
          </nav>
        </aside>

        <main className="flex-1 lg:ml-56 p-6 lg:p-12 max-w-3xl">
          {children}
        </main>
      </div>
    </div>
  )
}
