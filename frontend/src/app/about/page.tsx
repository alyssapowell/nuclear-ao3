'use client';

import Link from 'next/link';

export default function AboutPage() {
  return (
    <div className="max-w-4xl mx-auto px-4 py-8">
      <header className="mb-12">
        <h1 className="text-4xl font-bold text-slate-900 mb-6">About Nuclear AO3</h1>
        <p className="text-xl text-slate-600 leading-relaxed">
          A modern, liberation-licensed fanfiction archive built by and for the community, 
          prioritizing user privacy, accessibility, and creative freedom.
        </p>
      </header>

      <div className="space-y-12">
        {/* Genesis Section */}
        <section className="prose prose-lg max-w-none">
          <h2 className="text-2xl font-semibold text-slate-900 mb-4">Our Genesis</h2>
          <div className="text-slate-700 space-y-4">
            <p>
              Nuclear AO3 emerged from a simple recognition: the fanfiction community deserves 
              modern, accessible technology that respects both creators and readers. While existing 
              archives have served our community well, technological debt and structural limitations 
              have created barriers to innovation and improvement.
            </p>
            <p>
              This project represents a fresh start—an opportunity to build what we've learned 
              the community needs, using modern tools and approaches that prioritize user experience, 
              accessibility, and privacy from the ground up.
            </p>
            <p>
              We started with a core belief: fanfiction archives are community infrastructure, 
              and like all essential infrastructure, they should be built transparently, 
              maintained sustainably, and owned collectively.
            </p>
          </div>
        </section>

        {/* Community Need */}
        <section className="bg-blue-50 border border-blue-200 rounded-lg p-6">
          <h2 className="text-2xl font-semibold text-blue-900 mb-4">Meeting Community Needs</h2>
          <div className="text-blue-800 space-y-4">
            <p className="font-medium">The fanfiction community has spoken clearly about what they need:</p>
            <ul className="space-y-2 text-sm">
              <li><strong>Accessibility:</strong> WCAG 2.1 AA compliance isn't optional—it's fundamental to inclusive community access</li>
              <li><strong>Privacy:</strong> Robust protection of reading habits, personal data, and creative work</li>
              <li><strong>Performance:</strong> Fast, responsive interfaces that work on all devices and connection speeds</li>
              <li><strong>Transparency:</strong> Open development processes and clear governance structures</li>
              <li><strong>Innovation:</strong> Modern features that enhance discovery, organization, and community connection</li>
              <li><strong>Sustainability:</strong> Technical architecture designed for long-term maintainability</li>
            </ul>
            <p>
              Nuclear AO3 is our answer to these needs—not as a replacement, but as an 
              evolution of what fanfiction archives can become.
            </p>
          </div>
        </section>

        {/* Liberation License */}
        <section className="prose prose-lg max-w-none">
          <h2 className="text-2xl font-semibold text-slate-900 mb-4">Liberation License & Social Good</h2>
          <div className="text-slate-700 space-y-4">
            <p>
              Nuclear AO3 is licensed under the{' '}
              <a 
                href="https://github.com/liberationlicense/license" 
                target="_blank" 
                rel="noopener noreferrer"
                className="text-orange-600 hover:text-orange-700 underline"
              >
                Liberation License
              </a>
              —a license specifically designed for liberation technology that serves social good.
            </p>
            <p>
              Unlike traditional open source licenses, the Liberation License ensures that 
              derivative works continue to serve liberation purposes. This means:
            </p>
            <ul className="space-y-2">
              <li><strong>Community ownership:</strong> The code remains available for community benefit</li>
              <li><strong>Anti-exploitation:</strong> Commercial entities cannot simply take and monetize the work</li>
              <li><strong>Social purpose:</strong> Derivative works must maintain focus on social good</li>
              <li><strong>Access preservation:</strong> Ensures the technology remains accessible to marginalized communities</li>
            </ul>
            <p>
              For fanfiction—a practice often marginalized and misunderstood—this licensing 
              approach provides crucial protection while enabling innovation and community ownership.
            </p>
          </div>
        </section>

        {/* Technical Philosophy */}
        <section className="bg-green-50 border border-green-200 rounded-lg p-6">
          <h2 className="text-2xl font-semibold text-green-900 mb-4">Technical Philosophy</h2>
          <div className="text-green-800 space-y-4">
            <p>Every technical decision in Nuclear AO3 reflects our values:</p>
            <div className="grid md:grid-cols-2 gap-4 text-sm">
              <div>
                <h3 className="font-semibold mb-2">Privacy by Design</h3>
                <ul className="space-y-1">
                  <li>• Minimal data collection</li>
                  <li>• End-to-end encryption</li>
                  <li>• No tracking or analytics</li>
                  <li>• User-controlled data retention</li>
                </ul>
              </div>
              <div>
                <h3 className="font-semibold mb-2">Accessibility First</h3>
                <ul className="space-y-1">
                  <li>• WCAG 2.1 AA compliance</li>
                  <li>• Screen reader optimization</li>
                  <li>• Keyboard navigation</li>
                  <li>• High contrast support</li>
                </ul>
              </div>
              <div>
                <h3 className="font-semibold mb-2">Performance Focus</h3>
                <ul className="space-y-1">
                  <li>• Sub-second page loads</li>
                  <li>• Mobile optimization</li>
                  <li>• Offline reading support</li>
                  <li>• Efficient search algorithms</li>
                </ul>
              </div>
              <div>
                <h3 className="font-semibold mb-2">Community-Driven</h3>
                <ul className="space-y-1">
                  <li>• Open development process</li>
                  <li>• Community governance</li>
                  <li>• Transparent roadmap</li>
                  <li>• Collaborative decision-making</li>
                </ul>
              </div>
            </div>
          </div>
        </section>

        {/* Current Status */}
        <section className="prose prose-lg max-w-none">
          <h2 className="text-2xl font-semibold text-slate-900 mb-4">Current Status</h2>
          <div className="text-slate-700 space-y-4">
            <p>
              Nuclear AO3 is currently in active development, with core functionality 
              including work posting, series management, comments, bookmarks, and search 
              already implemented and tested.
            </p>
            <p>
              We're building transparently, with all development happening in the open. 
              The platform features comprehensive accessibility testing, privacy-focused 
              architecture, and modern user experience design.
            </p>
            <div className="bg-orange-50 border border-orange-200 rounded-lg p-4 mt-6">
              <p className="text-orange-800 text-sm mb-0">
                <strong>Want to contribute?</strong> Nuclear AO3 welcomes contributors of all 
                skill levels. Whether you write code, test accessibility, create documentation, 
                or provide community feedback, there's a place for you in shaping the future 
                of fanfiction archives.
              </p>
            </div>
          </div>
        </section>

        {/* Values */}
        <section className="bg-slate-50 border border-slate-200 rounded-lg p-6">
          <h2 className="text-2xl font-semibold text-slate-900 mb-4">Our Values</h2>
          <div className="grid md:grid-cols-3 gap-6 text-sm">
            <div>
              <h3 className="font-semibold text-slate-900 mb-2">🌈 Inclusivity</h3>
              <p className="text-slate-700">
                Fanfiction has always been a space for marginalized voices. We build technology 
                that welcomes everyone.
              </p>
            </div>
            <div>
              <h3 className="font-semibold text-slate-900 mb-2">🔒 Privacy</h3>
              <p className="text-slate-700">
                What you read and write is yours. We implement privacy protection as a 
                fundamental right, not a feature.
              </p>
            </div>
            <div>
              <h3 className="font-semibold text-slate-900 mb-2">📚 Preservation</h3>
              <p className="text-slate-700">
                Fanfiction is literature. We build systems designed to preserve creative 
                works for future generations.
              </p>
            </div>
            <div>
              <h3 className="font-semibold text-slate-900 mb-2">🤝 Community</h3>
              <p className="text-slate-700">
                Archives exist to serve their communities. Every decision prioritizes 
                user needs over technical convenience.
              </p>
            </div>
            <div>
              <h3 className="font-semibold text-slate-900 mb-2">⚡ Innovation</h3>
              <p className="text-slate-700">
                Modern tools enable better experiences. We embrace new technology that 
                serves our community's needs.
              </p>
            </div>
            <div>
              <h3 className="font-semibold text-slate-900 mb-2">🌍 Sustainability</h3>
              <p className="text-slate-700">
                We build for the long term, with architectures and governance structures 
                designed to last decades.
              </p>
            </div>
          </div>
        </section>

        {/* Contact */}
        <section className="prose prose-lg max-w-none">
          <h2 className="text-2xl font-semibold text-slate-900 mb-4">Get Involved</h2>
          <div className="text-slate-700 space-y-4">
            <p>
              Nuclear AO3 is built by and for the fanfiction community. We welcome 
              involvement at every level:
            </p>
            <div className="grid md:grid-cols-2 gap-6 text-sm">
              <div>
                <h3 className="font-semibold mb-2">For Users</h3>
                <ul className="space-y-1">
                  <li>• Test new features and provide feedback</li>
                  <li>• Report accessibility issues</li>
                  <li>• Suggest improvements</li>
                  <li>• Share your experience</li>
                </ul>
              </div>
              <div>
                <h3 className="font-semibold mb-2">For Contributors</h3>
                <ul className="space-y-1">
                  <li>• Development (all skill levels welcome)</li>
                  <li>• Documentation and user guides</li>
                  <li>• Accessibility testing</li>
                  <li>• Community governance</li>
                </ul>
              </div>
            </div>
            <p>
              Together, we're building the archive our community deserves—one that 
              respects our creativity, protects our privacy, and preserves our stories 
              for generations to come.
            </p>
          </div>
        </section>
      </div>

      {/* Navigation */}
      <footer className="mt-12 pt-8 border-t border-slate-200 text-center">
        <div className="flex flex-wrap justify-center gap-6 text-sm">
          <Link href="/terms" className="text-orange-600 hover:text-orange-700 underline">
            Terms of Service
          </Link>
          <Link href="/privacy" className="text-orange-600 hover:text-orange-700 underline">
            Privacy Policy
          </Link>
          <Link href="/attributions" className="text-orange-600 hover:text-orange-700 underline">
            Open Source Attributions
          </Link>
        </div>
      </footer>
    </div>
  );
}