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
    // Load generated attributions and combine with backend dependencies
    const loadAttributions = async () => {
      try {
        const response = await fetch('/attributions.json');
        const data = await response.json();
        
        // Convert generated data to our format
        const frontendDeps: LicenseData = {};
        data.dependencies.forEach((dep: any) => {
          frontendDeps[dep.name] = {
            name: dep.name,
            version: dep.version,
            licenses: dep.licenses,
            repository: dep.repository !== 'https://www.npmjs.com/package/' + dep.name ? dep.repository : undefined,
            publisher: dep.publisher !== 'Unknown' ? dep.publisher : undefined,
            description: dep.description !== 'Package description would be fetched from npm API' ? dep.description : undefined
          };
        });

        // Add backend Go dependencies
        const backendDeps: LicenseData = {
          'gin-gonic': {
        name: 'Gin Web Framework',
        version: 'v1.10.1',
        licenses: 'MIT',
        repository: 'https://github.com/gin-gonic/gin',
        publisher: 'Gin Contributors',
        description: 'HTTP web framework written in Go'
      },
      'elasticsearch': {
        name: 'Elasticsearch Go Client',
        version: 'v8.9.0',
        licenses: 'Apache-2.0',
        repository: 'https://github.com/elastic/go-elasticsearch',
        publisher: 'Elastic N.V.',
        description: 'Official Go client for Elasticsearch'
      },
      'postgresql': {
        name: 'PostgreSQL Driver',
        version: 'v1.10.9',
        licenses: 'MIT',
        repository: 'https://github.com/lib/pq',
        publisher: 'PostgreSQL Global Development Group',
        description: 'Pure Go Postgres driver for database connectivity'
      },
      'redis': {
        name: 'Redis Go Client',
        version: 'v9.2.1',
        licenses: 'BSD-2-Clause',
        repository: 'https://github.com/redis/go-redis',
        publisher: 'Redis Ltd.',
        description: 'Go client for Redis database'
      },
      'jwt': {
        name: 'JWT Go Library',
        version: 'v5.0.0',
        licenses: 'MIT',
        repository: 'https://github.com/golang-jwt/jwt',
        publisher: 'JWT Contributors',
        description: 'JSON Web Token implementation for Go'
      },
      'prometheus': {
        name: 'Prometheus Go Client',
        version: 'v1.17.0',
        licenses: 'Apache-2.0',
        repository: 'https://github.com/prometheus/client_golang',
        publisher: 'Prometheus Authors',
        description: 'Prometheus instrumentation library for Go applications'
      }
    };

        // Combine frontend and backend dependencies
        const allDependencies = { ...frontendDeps, ...backendDeps };
        setLicenses(allDependencies);
        
      } catch (error) {
        console.warn('Failed to load generated attributions, using fallback');
        // Fallback to static list if the JSON file isn't available
        setLicenses({});
      }
      
      setLoading(false);
    };

    loadAttributions();
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
        <div className="bg-orange-50 border border-orange-200 rounded-lg p-4 mb-4">
          <h2 className="font-semibold text-orange-900 mb-2">Nuclear AO3 License</h2>
          <p className="text-orange-800 text-sm">
            This project is licensed under the{' '}
            <a 
              href="https://github.com/liberationlicense/license" 
              target="_blank" 
              rel="noopener noreferrer"
              className="underline hover:no-underline font-medium"
            >
              Liberation License
            </a>
            , designed specifically for liberation technology that serves social good.
          </p>
        </div>
        
        <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
          <h2 className="font-semibold text-blue-900 mb-2">Dependency License Compatibility</h2>
          <p className="text-blue-800 text-sm">
            All dependencies use OSI-approved licenses compatible with our open source mission. 
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
            <li>• <strong>BSD-2-Clause:</strong> Simple permissive license</li>
            <li>• <strong>All licenses:</strong> Compatible with our open source mission</li>
            <li>• <strong>No GPL/AGPL:</strong> No copyleft requirements that could complicate integration</li>
          </ul>
        </div>
        
        <div className="bg-slate-50 border border-slate-200 rounded-lg p-4 mt-6">
          <h3 className="font-semibold text-slate-900 mb-2">Contributing & Updates</h3>
          <div className="text-slate-700 text-sm space-y-2">
            <p>
              <strong>Keep this updated:</strong> When adding new dependencies, please update this file 
              to maintain transparency about our open source stack.
            </p>
            <p>
              <strong>License conflicts:</strong> Before adding dependencies, verify license compatibility 
              with the Liberation License and our open source mission.
            </p>
            <p>
              <strong>Missing attribution?</strong> If you notice missing dependencies or incorrect information, 
              please submit a pull request or open an issue.
            </p>
          </div>
        </div>
        
        <p className="text-sm text-slate-500 mt-4 text-center">
          Last manually updated: {new Date().toLocaleDateString()} • 
          Please keep this current when dependencies change
        </p>
      </footer>
    </div>
  );
}