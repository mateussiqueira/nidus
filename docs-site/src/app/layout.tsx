import type { Metadata } from 'next'
import './globals.css'

export const metadata: Metadata = {
  title: 'Nimbus — Self-Hosted PaaS',
  description: 'Deploy like Vercel. Run on your own server. Open-source platform for apps, databases, and domains.',
  icons: { icon: '/favicon.png', apple: '/logo.png' },
  openGraph: {
    title: 'Nimbus — Self-Hosted PaaS',
    description: 'Deploy like Vercel. Run on your own server.',
    url: 'https://nimbus200.vercel.app',
    images: ['/logo.png'],
  },
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  )
}
