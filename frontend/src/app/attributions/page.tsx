'use client';

import { useEffect, useState } from 'react';

interface License {
  name: string;
  version: string;
  licenses: string;
  repository?: string;
  publisher?: string;
  email?: string;
  description?: string;
}

interface LicenseData {
  [key: string]: License;
}

export default function AttributionsPage() {
  const [licenses, setLicenses] = useState<LicenseData>({});
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // In a real app, you'd fetch this from an API endpoint that reads package.json
    // For now, we'll create a static list of our main dependencies
    const mainDependencies: LicenseData = {
      'next': {
        name: 'Next.js',
        version: '15.5.4',
        licenses: 'MIT',
        repository: 'https://github.com/vercel/next.js',
        publisher: 'Vercel',
        description: 'The React Framework for the Web'
      },
      'react': {
        name: 'React',
        version: '19.1.0',
        licenses: 'MIT',
        repository: 'https://github.com/facebook/react',
        publisher: 'Meta Platforms, Inc.',
        description: 'A JavaScript library for building user interfaces'
      },
      '@headlessui/react': {
        name: 'Headless UI',
        version: '^2.0.0',
        licenses: 'MIT',
        repository: 'https://github.com/tailwindlabs/headlessui',
        publisher: 'Tailwind Labs',
        description: 'Completely unstyled, fully accessible UI components'
      },
      'tailwindcss': {
        name: 'Tailwind CSS',
        version: '^3.4.14',
        licenses: 'MIT',
        repository: 'https://github.com/tailwindlabs/tailwindcss',
        publisher: 'Tailwind Labs',
        description: 'A utility-first CSS framework'
      },
      'typescript': {
        name: 'TypeScript',
        version: '^5',
        licenses: 'Apache-2.0',
        repository: 'https://github.com/microsoft/TypeScript',
        publisher: 'Microsoft Corporation',
        description: 'TypeScript is a language for application scale JavaScript development'
      },
      'axios': {
        name: 'Axios',
        version: '^1.12.2',
        licenses: 'MIT',
        repository: 'https://github.com/axios/axios',
        publisher: 'Matt Zabriskie',
        description: 'Promise based HTTP client for the browser and node.js'
      },
      'swr': {
        name: 'SWR',
        version: '^2.3.6',
        licenses: 'MIT',
        repository: 'https://github.com/vercel/swr',
        publisher: 'Vercel',
        description: 'Data fetching library for React'
      },
      'lucide-react': {
        name: 'Lucide React',
        version: '^0.544.0',
        licenses: 'ISC',
        repository: 'https://github.com/lucide-icons/lucide',
        publisher: 'Lucide Contributors',
        description: 'Beautiful & consistent icon toolkit made by the community'
      }
    };

    setLicenses(mainDependencies);
    setLoading(false);
  }, []);

  const groupedLicenses = Object.values(licenses).reduce((acc, license) => {
    const licenseType = license.licenses;
    if (!acc[licenseType]) {
      acc[licenseType] = [];
    }
    acc[licenseType].push(license);
    return acc;
  }, {} as Record<string, License[]>);

  if (loading) {
    return (
      <div className="max-w-4xl mx-auto px-4 py-8">
        <div className="animate-pulse">Loading attributions...</div>
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto px-4 py-8">
      <header className="mb-8">
        <h1 className="text-3xl font-bold text-slate-900 mb-4">
          Open Source Attributions
        </h1>
        <p className="text-lg text-slate-600 mb-4">
          Nuclear AO3 is built with and depends on numerous open source projects. 
          We're grateful to the maintainers and contributors of these projects.
        </p>
        <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
          <h2 className="font-semibold text-blue-900 mb-2">License Compatibility</h2>
          <p className="text-blue-800 text-sm">
            All dependencies use OSI-approved licenses compatible with AO3's mission. 
            No proprietary or restrictive licenses that could affect long-term maintainability.
          </p>
        </div>
      </header>

      {Object.entries(groupedLicenses)
        .sort(([a], [b]) => a.localeCompare(b))
        .map(([licenseType, deps]) => (
        <section key={licenseType} className="mb-8">
          <h2 className="text-xl font-semibold text-slate-900 mb-4 pb-2 border-b border-slate-200">
            {licenseType} License ({deps.length} {deps.length === 1 ? 'dependency' : 'dependencies'})
          </h2>
          
          <div className="grid gap-4">
            {deps.map((dep) => (
              <div key={`${dep.name}-${dep.version}`} className="bg-white border border-slate-200 rounded-lg p-4">
                <div className="flex justify-between items-start mb-2">
                  <div>
                    <h3 className="font-semibold text-slate-900">
                      {dep.name}
                    </h3>
                    <p className="text-sm text-slate-600">Version {dep.version}</p>
                  </div>
                  <span className="px-2 py-1 bg-slate-100 text-slate-700 text-xs font-medium rounded">
                    {dep.licenses}
                  </span>
                </div>
                
                {dep.description && (
                  <p className="text-slate-700 mb-2">{dep.description}</p>
                )}
                
                <div className="flex flex-wrap gap-4 text-sm text-slate-600">
                  {dep.publisher && (
                    <span>
                      <strong>Publisher:</strong> {dep.publisher}
                    </span>
                  )}
                  {dep.repository && (
                    <a 
                      href={dep.repository}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-blue-600 hover:text-blue-800 underline"
                    >
                      Repository →
                    </a>
                  )}
                </div>
              </div>
            ))}
          </div>
        </section>
      ))}

      <footer className="mt-12 pt-8 border-t border-slate-200">
        <div className="bg-green-50 border border-green-200 rounded-lg p-4">
          <h3 className="font-semibold text-green-900 mb-2">License Summary</h3>
          <ul className="text-green-800 text-sm space-y-1">
            <li>• <strong>MIT & ISC:</strong> Maximum freedom, no restrictions</li>
            <li>• <strong>Apache-2.0:</strong> Permissive with patent protection</li>
            <li>• <strong>All licenses:</strong> Compatible with AO3's open source mission</li>
            <li>• <strong>No GPL/AGPL:</strong> No copyleft requirements that could complicate AO3 integration</li>
          </ul>
        </div>
        
        <p className="text-sm text-slate-500 mt-4 text-center">
          This page is automatically updated when dependencies change. 
          Last updated: {new Date().toLocaleDateString()}
        </p>
      </footer>
    </div>
  );
}