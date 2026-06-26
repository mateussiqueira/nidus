import type { Metadata } from 'next'
import './globals.css'

export const metadata: Metadata = {
  title: 'Nidus Docs',
  description: 'Documentation for Nidus - Self-hosted deploy platform',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  )
}
