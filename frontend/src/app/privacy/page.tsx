'use client';

import Link from 'next/link';

export default function PrivacyPolicyPage() {
  const lastUpdated = new Date().toLocaleDateString('en-US', { 
    year: 'numeric', 
    month: 'long', 
    day: 'numeric' 
  });

  return (
    <div className="max-w-4xl mx-auto px-4 py-8">
      <header className="mb-8">
        <h1 className="text-3xl font-bold text-slate-900 mb-4">Privacy Policy</h1>
        <p className="text-slate-600 mb-4">
          Last updated: {lastUpdated}
        </p>
        <div className="bg-green-50 border border-green-200 rounded-lg p-6">
          <h2 className="font-semibold text-green-900 mb-2">🔒 Privacy as a First Principle</h2>
          <p className="text-green-800 text-sm mb-3">
            At Nuclear AO3, privacy isn't an afterthought—it's the foundation of everything we build. 
            We believe what you read and write is deeply personal, and we've gone to extraordinary 
            lengths to protect that privacy.
          </p>
          <div className="grid md:grid-cols-3 gap-3 text-xs">
            <div>
              <strong>✓ End-to-end encryption</strong><br/>
              Your data is encrypted in transit and at rest
            </div>
            <div>
              <strong>✓ Zero tracking</strong><br/>
              No analytics, no behavior monitoring
            </div>
            <div>
              <strong>✓ Minimal data collection</strong><br/>
              We only collect what's absolutely necessary
            </div>
          </div>
        </div>
      </header>

      <div className="prose prose-lg max-w-none space-y-8">
        
        {/* Core Principles */}
        <section>
          <h2 className="text-2xl font-semibold text-slate-900 mb-4">Our Privacy Principles</h2>
          <div className="bg-blue-50 border border-blue-200 rounded-lg p-6">
            <div className="grid md:grid-cols-2 gap-6 text-sm">
              <div>
                <h3 className="font-semibold text-blue-900 mb-2">🛡️ Data Minimization</h3>
                <p className="text-blue-800">We collect only the absolute minimum data required for Nuclear AO3 to function. If we don't need it, we don't collect it.</p>
              </div>
              <div>
                <h3 className="font-semibold text-blue-900 mb-2">🔐 Encryption Everything</h3>
                <p className="text-blue-800">All data is encrypted in transit (HTTPS) and at rest (AES-256). Even our staff cannot read your private data.</p>
              </div>
              <div>
                <h3 className="font-semibold text-blue-900 mb-2">🚫 No Surveillance</h3>
                <p className="text-blue-800">We don't track your reading habits, browsing patterns, or behavior. No analytics, no fingerprinting, no surveillance.</p>
              </div>
              <div>
                <h3 className="font-semibold text-blue-900 mb-2">🗑️ Aggressive Deletion</h3>
                <p className="text-blue-800">When you delete something, it's actually deleted. No "soft deletes," no backups, no recovery—gone means gone.</p>
              </div>
              <div>
                <h3 className="font-semibold text-blue-900 mb-2">🌍 International Privacy</h3>
                <p className="text-blue-800">We comply with GDPR, CCPA, and other privacy regulations by design, regardless of where you're located.</p>
              </div>
              <div>
                <h3 className="font-semibold text-blue-900 mb-2">🤝 Transparency</h3>
                <p className="text-blue-800">This policy tells you exactly what we do and don't do. Our code is open source—you can verify our claims.</p>
              </div>
            </div>
          </div>
        </section>

        {/* What We Collect */}
        <section>
          <h2 className="text-2xl font-semibold text-slate-900 mb-4">What We Collect (And Why)</h2>
          <div className="text-slate-700 space-y-6">
            
            <div className="border border-green-200 rounded-lg p-4 bg-green-50">
              <h3 className="font-semibold text-green-900 mb-3">✅ What We DO Collect:</h3>
              <div className="space-y-3 text-sm">
                <div>
                  <strong>Account Information:</strong>
                  <ul className="list-disc pl-5 mt-1 space-y-1">
                    <li><strong>Email address</strong> - For account recovery only (encrypted, never shared)</li>
                    <li><strong>Username</strong> - Your chosen public identifier</li>
                    <li><strong>Encrypted password</strong> - Hashed with bcrypt, salt rounds 12+</li>
                    <li><strong>Account creation date</strong> - For security and spam prevention</li>
                  </ul>
                </div>
                
                <div>
                  <strong>Content You Post:</strong>
                  <ul className="list-disc pl-5 mt-1 space-y-1">
                    <li><strong>Works, comments, bookmarks</strong> - The content you choose to share</li>
                    <li><strong>Tags and metadata</strong> - For organization and discovery</li>
                    <li><strong>Publication dates</strong> - When you posted content</li>
                  </ul>
                </div>
                
                <div>
                  <strong>Essential Technical Data:</strong>
                  <ul className="list-disc pl-5 mt-1 space-y-1">
                    <li><strong>Last login time</strong> - For security purposes only</li>
                    <li><strong>Session tokens</strong> - Encrypted, expire after 24 hours</li>
                    <li><strong>Rate limiting counters</strong> - To prevent abuse (reset hourly)</li>
                  </ul>
                </div>
              </div>
            </div>

            <div className="border border-red-200 rounded-lg p-4 bg-red-50">
              <h3 className="font-semibold text-red-900 mb-3">❌ What We DON'T Collect:</h3>
              <div className="text-sm text-red-800 space-y-2">
                <ul className="list-disc pl-5 space-y-1">
                  <li><strong>IP addresses</strong> - Not logged, not stored, not tracked</li>
                  <li><strong>Browser fingerprints</strong> - No device identification</li>
                  <li><strong>Reading history</strong> - We don't track what you read</li>
                  <li><strong>Search queries</strong> - Searches are processed but not stored</li>
                  <li><strong>Analytics data</strong> - No Google Analytics, no tracking pixels</li>
                  <li><strong>Location data</strong> - No geolocation tracking</li>
                  <li><strong>Social media connections</strong> - No third-party integrations</li>
                  <li><strong>Advertising identifiers</strong> - We don't serve ads</li>
                  <li><strong>Behavioral patterns</strong> - No user profiling or pattern analysis</li>
                  <li><strong>Personal metadata</strong> - No age, gender, or demographic data</li>
                </ul>
              </div>
            </div>
          </div>
        </section>

        {/* How We Protect Your Data */}
        <section>
          <h2 className="text-2xl font-semibold text-slate-900 mb-4">How We Protect Your Data</h2>
          <div className="text-slate-700 space-y-4">
            <div className="grid md:grid-cols-2 gap-6">
              <div className="bg-slate-50 border border-slate-200 rounded-lg p-4">
                <h3 className="font-semibold text-slate-900 mb-3">🔐 Encryption</h3>
                <ul className="text-sm space-y-1">
                  <li><strong>In Transit:</strong> TLS 1.3 with perfect forward secrecy</li>
                  <li><strong>At Rest:</strong> AES-256 encryption for all stored data</li>
                  <li><strong>Passwords:</strong> bcrypt with 12+ salt rounds</li>
                  <li><strong>Sessions:</strong> Encrypted tokens, 24-hour expiry</li>
                </ul>
              </div>

              <div className="bg-slate-50 border border-slate-200 rounded-lg p-4">
                <h3 className="font-semibold text-slate-900 mb-3">🏗️ Infrastructure</h3>
                <ul className="text-sm space-y-1">
                  <li><strong>Server hardening:</strong> Minimal services, regular updates</li>
                  <li><strong>Network isolation:</strong> Segmented environments</li>
                  <li><strong>Access control:</strong> Principle of least privilege</li>
                  <li><strong>Monitoring:</strong> Security-focused, privacy-preserving</li>
                </ul>
              </div>

              <div className="bg-slate-50 border border-slate-200 rounded-lg p-4">
                <h3 className="font-semibold text-slate-900 mb-3">👨‍💻 Access Controls</h3>
                <ul className="text-sm space-y-1">
                  <li><strong>Staff access:</strong> Limited to essential functions only</li>
                  <li><strong>Data access:</strong> Logged and audited</li>
                  <li><strong>Admin privileges:</strong> Time-limited, purpose-specific</li>
                  <li><strong>Third parties:</strong> Zero access to user data</li>
                </ul>
              </div>

              <div className="bg-slate-50 border border-slate-200 rounded-lg p-4">
                <h3 className="font-semibold text-slate-900 mb-3">🔄 Data Lifecycle</h3>
                <ul className="text-sm space-y-1">
                  <li><strong>Retention:</strong> Only as long as necessary</li>
                  <li><strong>Deletion:</strong> Immediate, permanent, irreversible</li>
                  <li><strong>Backups:</strong> Encrypted, time-limited</li>
                  <li><strong>Data portability:</strong> Export your data anytime</li>
                </ul>
              </div>
            </div>
          </div>
        </section>

        {/* Your Privacy Rights */}
        <section>
          <h2 className="text-2xl font-semibold text-slate-900 mb-4">Your Privacy Rights</h2>
          <div className="text-slate-700 space-y-4">
            <p>
              You have complete control over your data. These rights are built into Nuclear AO3, 
              not just promised in policy:
            </p>
            
            <div className="grid md:grid-cols-2 gap-4">
              <div className="bg-orange-50 border border-orange-200 rounded-lg p-4">
                <h3 className="font-semibold text-orange-900 mb-2">🗂️ Right to Access</h3>
                <p className="text-orange-800 text-sm">
                  View all data we have about you through your account dashboard. 
                  Export everything with one click.
                </p>
              </div>

              <div className="bg-orange-50 border border-orange-200 rounded-lg p-4">
                <h3 className="font-semibold text-orange-900 mb-2">✏️ Right to Rectification</h3>
                <p className="text-orange-800 text-sm">
                  Correct or update any information directly through your account 
                  settings. Changes are immediate.
                </p>
              </div>

              <div className="bg-orange-50 border border-orange-200 rounded-lg p-4">
                <h3 className="font-semibold text-orange-900 mb-2">🗑️ Right to Deletion</h3>
                <p className="text-orange-800 text-sm">
                  Delete individual items or your entire account. Deletion is 
                  permanent and immediate—we can't recover deleted data.
                </p>
              </div>

              <div className="bg-orange-50 border border-orange-200 rounded-lg p-4">
                <h3 className="font-semibold text-orange-900 mb-2">📦 Right to Portability</h3>
                <p className="text-orange-800 text-sm">
                  Export all your data in standard formats (JSON, HTML, ePub). 
                  Take your content anywhere.
                </p>
              </div>

              <div className="bg-orange-50 border border-orange-200 rounded-lg p-4">
                <h3 className="font-semibold text-orange-900 mb-2">🛑 Right to Object</h3>
                <p className="text-orange-800 text-sm">
                  Object to any data processing. Since we do minimal processing, 
                  this mainly means account deletion.
                </p>
              </div>

              <div className="bg-orange-50 border border-orange-200 rounded-lg p-4">
                <h3 className="font-semibold text-orange-900 mb-2">⏸️ Right to Restriction</h3>
                <p className="text-orange-800 text-sm">
                  Temporarily restrict processing while we resolve any concerns 
                  about your data.
                </p>
              </div>
            </div>
          </div>
        </section>

        {/* Data Sharing */}
        <section>
          <h2 className="text-2xl font-semibold text-slate-900 mb-4">Data Sharing (Spoiler: We Don't)</h2>
          <div className="text-slate-700 space-y-4">
            <div className="bg-green-50 border border-green-200 rounded-lg p-4">
              <h3 className="font-semibold text-green-900 mb-2">Our Promise:</h3>
              <p className="text-green-800 text-sm mb-3">
                <strong>We do not sell, rent, share, or otherwise distribute your personal data 
                to any third parties, ever, for any reason.</strong>
              </p>
              <p className="text-green-800 text-sm">
                This isn't a business decision—it's a fundamental principle. Nuclear AO3 
                exists to serve the community, not to monetize user data.
              </p>
            </div>

            <p><strong>Limited Exceptions (Legally Required Only):</strong></p>
            <ul className="list-disc pl-6 space-y-2 text-sm">
              <li>
                <strong>Legal compliance:</strong> If required by valid legal process (subpoena, court order). 
                We will challenge overly broad requests and notify users when legally permitted.
              </li>
              <li>
                <strong>Safety emergencies:</strong> Only in immediate threats to human life or safety, 
                and only the minimum data necessary.
              </li>
              <li>
                <strong>Technical service providers:</strong> Our hosting and infrastructure providers 
                may have technical access to encrypted data but are contractually bound to our privacy standards.
              </li>
            </ul>

            <p className="text-sm bg-yellow-50 border border-yellow-200 rounded p-3">
              <strong>Transparency Report:</strong> We commit to publishing an annual transparency 
              report detailing any legal requests for user data and how we responded.
            </p>
          </div>
        </section>

        {/* International Users */}
        <section>
          <h2 className="text-2xl font-semibold text-slate-900 mb-4">International Privacy</h2>
          <div className="text-slate-700 space-y-4">
            <p>
              Nuclear AO3 serves a global community, and we respect international privacy laws:
            </p>
            
            <div className="grid md:grid-cols-3 gap-4 text-sm">
              <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                <h3 className="font-semibold text-blue-900 mb-2">🇪🇺 GDPR Compliance</h3>
                <p className="text-blue-800">
                  Full compliance with EU General Data Protection Regulation, 
                  including data minimization and privacy by design.
                </p>
              </div>

              <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                <h3 className="font-semibold text-blue-900 mb-2">🇺🇸 CCPA Compliance</h3>
                <p className="text-blue-800">
                  California Consumer Privacy Act rights are built into our 
                  platform for all users worldwide.
                </p>
              </div>

              <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                <h3 className="font-semibold text-blue-900 mb-2">🌍 Global Standards</h3>
                <p className="text-blue-800">
                  We apply the highest privacy standards globally, regardless 
                  of local legal requirements.
                </p>
              </div>
            </div>
          </div>
        </section>

        {/* Children's Privacy */}
        <section>
          <h2 className="text-2xl font-semibold text-slate-900 mb-4">Children's Privacy</h2>
          <div className="text-slate-700 space-y-4">
            <div className="bg-purple-50 border border-purple-200 rounded-lg p-4">
              <p className="text-purple-800 text-sm">
                <strong>Age Requirement:</strong> Nuclear AO3's Terms of Service require users to be at least 13 years old. 
                We do not knowingly collect personal information from children under 13.
              </p>
            </div>
            
            <p>
              <strong>No Age Verification:</strong> We do not collect birthdates or implement age verification 
              systems. We rely on users to truthfully represent that they meet our age requirement when creating accounts.
            </p>

            <p>
              <strong>COPPA Compliance:</strong> Because we do not collect birthdates or other age-identifying 
              information, we generally cannot determine if a user is under 13. If we become aware through other 
              means (such as parental contact) that a user is under 13, we will disable the account and delete 
              associated personal information in accordance with COPPA requirements.
            </p>

            <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
              <p className="text-yellow-800 text-sm">
                <strong>Parental Notice:</strong> Nuclear AO3 hosts user-generated content that may include 
                mature themes. We encourage parents and guardians to discuss internet safety and appropriate 
                content consumption with their children and teens. If you believe your child under 13 has 
                created an account, please <a href="/contact" className="text-yellow-900 underline">contact us</a> immediately.
              </p>
            </div>
          </div>
        </section>

        {/* Security Incidents */}
        <section>
          <h2 className="text-2xl font-semibold text-slate-900 mb-4">Security Incident Response</h2>
          <div className="text-slate-700 space-y-4">
            <p>
              Despite our best efforts, security incidents can happen. Here's our commitment:
            </p>
            
            <div className="bg-red-50 border border-red-200 rounded-lg p-4">
              <h3 className="font-semibold text-red-900 mb-2">🚨 Incident Response Promise:</h3>
              <div className="text-red-800 text-sm space-y-2">
                <p><strong>Immediate response:</strong> Security team notified within 1 hour of detection</p>
                <p><strong>User notification:</strong> Affected users notified within 24-72 hours</p>
                <p><strong>Public disclosure:</strong> Full incident report published within 30 days</p>
                <p><strong>Remediation:</strong> All vulnerabilities patched before resuming normal operations</p>
                <p><strong>Transparency:</strong> No hiding behind legal language—clear, honest communication</p>
              </div>
            </div>

            <p className="text-sm">
              We maintain incident response procedures, conduct regular security audits, 
              and participate in responsible disclosure programs.
            </p>
          </div>
        </section>

        {/* Policy Updates */}
        <section>
          <h2 className="text-2xl font-semibold text-slate-900 mb-4">Policy Updates</h2>
          <div className="text-slate-700 space-y-4">
            <p>
              This privacy policy may be updated to reflect changes in our practices 
              or legal requirements. However, our core privacy principles will never change.
            </p>
            
            <div className="bg-slate-50 border border-slate-200 rounded-lg p-4">
              <h3 className="font-semibold text-slate-900 mb-2">Update Process:</h3>
              <ul className="text-sm space-y-1">
                <li><strong>Advance notice:</strong> 30 days minimum for material changes</li>
                <li><strong>Clear communication:</strong> Plain language explanation of what changed and why</li>
                <li><strong>Community input:</strong> Open discussion period for significant changes</li>
                <li><strong>Opt-out option:</strong> Ability to delete account if you disagree with changes</li>
                <li><strong>Version control:</strong> All previous versions available for review</li>
              </ul>
            </div>
          </div>
        </section>

        {/* Contact */}
        <section>
          <h2 className="text-2xl font-semibold text-slate-900 mb-4">Privacy Questions & Concerns</h2>
          <div className="text-slate-700 space-y-4">
            <p>
              We're committed to transparency and accountability. If you have any questions 
              about this privacy policy or our practices:
            </p>
            
            <div className="grid md:grid-cols-2 gap-4 text-sm">
              <div className="bg-orange-50 border border-orange-200 rounded-lg p-4">
                <h3 className="font-semibold text-orange-900 mb-2">📧 Contact Options</h3>
                <ul className="text-orange-800 space-y-1">
                  <li>• <Link href="/contact" className="underline">Contact form</Link> (fastest response)</li>
                  <li>• <Link href="/feedback" className="underline">Privacy feedback</Link> (anonymous option)</li>
                  <li>• Community discussions on policy changes</li>
                </ul>
              </div>

              <div className="bg-orange-50 border border-orange-200 rounded-lg p-4">
                <h3 className="font-semibold text-orange-900 mb-2">⚡ Response Commitment</h3>
                <ul className="text-orange-800 space-y-1">
                  <li>• General questions: 48 hours</li>
                  <li>• Privacy concerns: 24 hours</li>
                  <li>• Urgent issues: Same day</li>
                  <li>• Delete requests: Immediate</li>
                </ul>
              </div>
            </div>
          </div>
        </section>

      </div>

      {/* Footer Navigation */}
      <footer className="mt-12 pt-8 border-t border-slate-200">
        <div className="bg-green-50 rounded-lg p-4">
          <h3 className="font-semibold text-green-900 mb-2">Privacy Resources</h3>
          <div className="flex flex-wrap gap-4 text-sm">
            <Link href="/profile/privacy" className="text-orange-600 hover:text-orange-700 underline">
              Privacy Settings
            </Link>
            <Link href="/terms" className="text-orange-600 hover:text-orange-700 underline">
              Terms of Service
            </Link>
            <Link href="/accessibility" className="text-orange-600 hover:text-orange-700 underline">
              Accessibility Statement
            </Link>
            <Link href="/contact" className="text-orange-600 hover:text-orange-700 underline">
              Contact Us
            </Link>
          </div>
        </div>
      </footer>
    </div>
  );
}