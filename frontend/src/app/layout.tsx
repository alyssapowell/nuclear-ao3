import Navigation from '@/components/Navigation'
import { ApolloProvider } from '@/components/ApolloProvider'
import GDPRConsentBanner from '@/components/GDPRConsentBanner'
import ServiceStatus from '@/components/ServiceStatus'
import './globals.css'

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en">
      <head>
        <link rel="stylesheet" href="/styles.css" />
      </head>
      <body className="min-h-screen bg-gray-50 font-sans antialiased">
        <ApolloProvider>
          <div className="min-h-screen flex flex-col">
            <Navigation />
            <main className="flex-1">
              {children}
            </main>
          </div>
          <GDPRConsentBanner />
          <ServiceStatus />
        </ApolloProvider>
      </body>
    </html>
  )
}