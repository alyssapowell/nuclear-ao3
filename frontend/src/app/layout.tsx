import Navigation from '@/components/Navigation'
import Footer from '@/components/Footer'
import { ApolloProvider } from '@/components/ApolloProvider'
import GDPRConsentBanner from '@/components/GDPRConsentBanner'
import ServiceStatus from '@/components/ServiceStatus'
import PWAInit from '@/components/PWAInit'
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
        <link rel="manifest" href="/manifest.json" />
        <meta name="theme-color" content="#3b82f6" />
        <meta name="apple-mobile-web-app-capable" content="yes" />
        <meta name="apple-mobile-web-app-status-bar-style" content="default" />
        <meta name="apple-mobile-web-app-title" content="Nuclear AO3" />
        <link rel="apple-touch-icon" href="/icon-192x192.svg" />
      </head>
      <body className="min-h-screen bg-gray-50 font-sans antialiased">
        <ApolloProvider>
          <div className="min-h-screen flex flex-col">
            <Navigation />
            <main className="flex-1">
              {children}
            </main>
            <Footer />
          </div>
          <GDPRConsentBanner />
          <ServiceStatus />
          <PWAInit />
        </ApolloProvider>
      </body>
    </html>
  )
}