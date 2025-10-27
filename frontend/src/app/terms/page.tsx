'use client';

import Link from 'next/link';

export default function TermsOfServicePage() {
  const lastUpdated = new Date().toLocaleDateString('en-US', { 
    year: 'numeric', 
    month: 'long', 
    day: 'numeric' 
  });

  return (
    <div className="max-w-4xl mx-auto px-4 py-8">
      <header className="mb-8">
        <h1 className="text-3xl font-bold text-slate-900 mb-4">Terms of Service</h1>
        <p className="text-slate-600 mb-2">
          Last updated: {lastUpdated}
        </p>
        <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
          <p className="text-blue-800 text-sm">
            These Terms of Service govern your use of Nuclear AO3. By using our service, 
            you agree to these terms. Please read them carefully.
          </p>
        </div>
      </header>

      <div className="prose prose-lg max-w-none space-y-8">
        
        {/* Acceptance */}
        <section>
          <h2 className="text-2xl font-semibold text-slate-900 mb-4">1. Acceptance of Terms</h2>
          <div className="text-slate-700 space-y-4">
            <p>
              By accessing or using Nuclear AO3, you agree to be bound by these Terms of Service 
              and all applicable laws and regulations. If you do not agree with any of these terms, 
              you are prohibited from using or accessing this service.
            </p>
            <p>
              These terms may be updated from time to time. Your continued use of the service 
              after any such changes constitutes your acceptance of the new terms.
            </p>
          </div>
        </section>

        {/* Description of Service */}
        <section>
          <h2 className="text-2xl font-semibold text-slate-900 mb-4">2. Description of Service</h2>
          <div className="text-slate-700 space-y-4">
            <p>
              Nuclear AO3 is a non-commercial, non-profit archive for transformative works, 
              including fanfiction, fanart, fan videos, and other fan-created content. 
              The service is provided free of charge to support and preserve fan creativity.
            </p>
            <p>
              We are committed to providing a platform that:
            </p>
            <ul className="list-disc pl-6 space-y-2">
              <li>Supports maximum accessibility for all users</li>
              <li>Preserves fan works for current and future generations</li>
              <li>Protects user privacy and creative freedom</li>
              <li>Operates transparently under the Liberation License</li>
            </ul>
          </div>
        </section>

        {/* User Accounts */}
        <section>
          <h2 className="text-2xl font-semibold text-slate-900 mb-4">3. User Accounts and Responsibilities</h2>
          <div className="text-slate-700 space-y-4">
            <p><strong>Account Creation:</strong></p>
            <ul className="list-disc pl-6 space-y-2">
              <li>You must provide accurate information when creating an account</li>
              <li>You are responsible for maintaining the security of your account</li>
              <li>You must not share your account credentials with others</li>
              <li>You must be at least 13 years old to create an account</li>
            </ul>
            
            <p><strong>Account Conduct:</strong></p>
            <ul className="list-disc pl-6 space-y-2">
              <li>You are responsible for all activity that occurs under your account</li>
              <li>You must not impersonate others or create false identities</li>
              <li>Multiple accounts by the same person are not permitted</li>
              <li>You must notify us immediately of any unauthorized use of your account</li>
            </ul>
          </div>
        </section>

        {/* Content Guidelines */}
        <section>
          <h2 className="text-2xl font-semibold text-slate-900 mb-4">4. Content Guidelines</h2>
          <div className="text-slate-700 space-y-4">
            <p><strong>Permitted Content:</strong></p>
            <ul className="list-disc pl-6 space-y-2">
              <li>Fanfiction and other transformative works</li>
              <li>Original fiction posted in designated areas</li>
              <li>Fanart, podfic, and other fan-created media</li>
              <li>Content that is properly tagged and categorized</li>
            </ul>

            <p><strong>Prohibited Content:</strong></p>
            <ul className="list-disc pl-6 space-y-2">
              <li>Content that violates copyright (except for fair use/transformative works)</li>
              <li>Real person fiction involving minors or posted without consent</li>
              <li>Content promoting illegal activities</li>
              <li>Spam, advertising, or commercial content</li>
              <li>Harassment, doxxing, or content intended to harm others</li>
              <li>Content containing viruses, malware, or other harmful code</li>
            </ul>

            <p><strong>Content Responsibility:</strong></p>
            <ul className="list-disc pl-6 space-y-2">
              <li>You retain ownership of content you post</li>
              <li>You are responsible for accurately tagging your content</li>
              <li>You grant Nuclear AO3 a license to host and display your content</li>
              <li>You warrant that you have the right to post your content</li>
            </ul>
          </div>
        </section>

        {/* Community Standards */}
        <section>
          <h2 className="text-2xl font-semibold text-slate-900 mb-4">5. Community Standards</h2>
          <div className="text-slate-700 space-y-4">
            <p><strong>Respectful Interaction:</strong></p>
            <ul className="list-disc pl-6 space-y-2">
              <li>Treat all users with respect and courtesy</li>
              <li>Engage constructively in comments and discussions</li>
              <li>Respect content creators' boundaries and preferences</li>
              <li>Use content warnings and tags appropriately</li>
            </ul>

            <p><strong>Prohibited Behavior:</strong></p>
            <ul className="list-disc pl-6 space-y-2">
              <li>Harassment, bullying, or threatening other users</li>
              <li>Discrimination based on identity, race, gender, sexuality, or other characteristics</li>
              <li>Deliberately triggering or harmful behavior</li>
              <li>Attempts to circumvent site features or user blocking</li>
              <li>Coordinated attacks or brigading</li>
            </ul>
          </div>
        </section>

        {/* Privacy and Data */}
        <section>
          <h2 className="text-2xl font-semibold text-slate-900 mb-4">6. Privacy and Data Protection</h2>
          <div className="text-slate-700 space-y-4">
            <p>
              Your privacy is fundamental to our mission. We collect minimal data necessary 
              for service operation and implement strong protections for all user information. 
              For complete details, see our{' '}
              <Link href="/privacy" className="text-orange-600 hover:text-orange-700 underline">
                Privacy Policy
              </Link>.
            </p>
            <p><strong>Key Privacy Principles:</strong></p>
            <ul className="list-disc pl-6 space-y-2">
              <li>We never sell or share personal data with third parties</li>
              <li>We use encryption to protect sensitive information</li>
              <li>You control your data and can request deletion at any time</li>
              <li>We are transparent about all data collection and use</li>
            </ul>
          </div>
        </section>

        {/* Intellectual Property */}
        <section>
          <h2 className="text-2xl font-semibent text-slate-900 mb-4">7. Intellectual Property</h2>
          <div className="text-slate-700 space-y-4">
            <p><strong>Your Content:</strong></p>
            <ul className="list-disc pl-6 space-y-2">
              <li>You retain all rights to content you create and post</li>
              <li>You grant Nuclear AO3 a non-exclusive license to host and display your content</li>
              <li>You may delete your content at any time</li>
              <li>You are responsible for respecting others' intellectual property rights</li>
            </ul>

            <p><strong>Platform Rights:</strong></p>
            <ul className="list-disc pl-6 space-y-2">
              <li>Nuclear AO3 software is licensed under the Liberation License</li>
              <li>Our trademarks and branding remain our property</li>
              <li>We respect the DMCA and respond to valid takedown requests</li>
              <li>We support fair use and transformative work rights</li>
            </ul>
          </div>
        </section>

        {/* Moderation and Enforcement */}
        <section>
          <h2 className="text-2xl font-semibold text-slate-900 mb-4">8. Moderation and Enforcement</h2>
          <div className="text-slate-700 space-y-4">
            <p><strong>Content Moderation:</strong></p>
            <ul className="list-disc pl-6 space-y-2">
              <li>We review reported content and may remove violations</li>
              <li>We may restrict access to improperly tagged content</li>
              <li>Repeat violations may result in account suspension</li>
              <li>We aim for transparent and fair enforcement</li>
            </ul>

            <p><strong>Appeals Process:</strong></p>
            <ul className="list-disc pl-6 space-y-2">
              <li>You may appeal moderation decisions</li>
              <li>We will review appeals in good faith</li>
              <li>Community input may be considered in complex cases</li>
              <li>Final decisions rest with platform administrators</li>
            </ul>
          </div>
        </section>

        {/* Service Availability */}
        <section>
          <h2 className="text-2xl font-semibold text-slate-900 mb-4">9. Service Availability</h2>
          <div className="text-slate-700 space-y-4">
            <p>
              We strive to maintain reliable service but cannot guarantee 100% uptime. 
              The service may be temporarily unavailable due to:
            </p>
            <ul className="list-disc pl-6 space-y-2">
              <li>Scheduled maintenance and updates</li>
              <li>Technical difficulties or server issues</li>
              <li>Security incidents requiring immediate response</li>
              <li>Force majeure events beyond our control</li>
            </ul>
            <p>
              We will provide advance notice of planned maintenance when possible and 
              work to minimize service interruptions.
            </p>
          </div>
        </section>

        {/* Disclaimers */}
        <section>
          <h2 className="text-2xl font-semibold text-slate-900 mb-4">10. Disclaimers and Limitations</h2>
          <div className="text-slate-700 space-y-4">
            <p><strong>Service Disclaimer:</strong></p>
            <p>
              Nuclear AO3 is provided "as is" without warranties of any kind. We do not 
              guarantee that the service will meet your requirements or be error-free.
            </p>

            <p><strong>Content Disclaimer:</strong></p>
            <p>
              We do not control or endorse user-generated content. Views expressed in 
              posted works do not represent our opinions or positions.
            </p>

            <p><strong>Limitation of Liability:</strong></p>
            <p>
              To the maximum extent permitted by law, Nuclear AO3 shall not be liable 
              for any indirect, incidental, special, or consequential damages arising 
              from your use of the service.
            </p>
          </div>
        </section>

        {/* Termination */}
        <section>
          <h2 className="text-2xl font-semibold text-slate-900 mb-4">11. Termination</h2>
          <div className="text-slate-700 space-y-4">
            <p><strong>Your Rights:</strong></p>
            <ul className="list-disc pl-6 space-y-2">
              <li>You may close your account at any time</li>
              <li>You may delete your content before account closure</li>
              <li>Account closure is permanent and cannot be undone</li>
            </ul>

            <p><strong>Our Rights:</strong></p>
            <ul className="list-disc pl-6 space-y-2">
              <li>We may suspend accounts for terms violations</li>
              <li>We may terminate service with reasonable notice</li>
              <li>We will preserve user content during platform transitions when possible</li>
            </ul>
          </div>
        </section>

        {/* Governing Law */}
        <section>
          <h2 className="text-2xl font-semibold text-slate-900 mb-4">12. Governing Law and Jurisdiction</h2>
          <div className="text-slate-700 space-y-4">
            <p>
              These Terms of Service are governed by applicable laws regarding online 
              services and user-generated content platforms. Any disputes will be 
              resolved through good-faith discussion and, if necessary, appropriate 
              legal channels.
            </p>
            <p>
              We are committed to fair and transparent dispute resolution that respects 
              the rights of our international user community.
            </p>
          </div>
        </section>

        {/* Contact */}
        <section>
          <h2 className="text-2xl font-semibold text-slate-900 mb-4">13. Contact Information</h2>
          <div className="text-slate-700 space-y-4">
            <p>
              If you have questions about these Terms of Service, please contact us through 
              our{' '}
              <Link href="/contact" className="text-orange-600 hover:text-orange-700 underline">
                Contact page
              </Link>
              {' '}or{' '}
              <Link href="/feedback" className="text-orange-600 hover:text-orange-700 underline">
                Feedback system
              </Link>.
            </p>
            <p>
              We welcome community input on policies and are committed to transparent 
              governance that serves our users' needs.
            </p>
          </div>
        </section>

      </div>

      {/* Footer Navigation */}
      <footer className="mt-12 pt-8 border-t border-slate-200">
        <div className="bg-slate-50 rounded-lg p-4">
          <h3 className="font-semibold text-slate-900 mb-2">Related Policies</h3>
          <div className="flex flex-wrap gap-4 text-sm">
            <Link href="/privacy" className="text-orange-600 hover:text-orange-700 underline">
              Privacy Policy
            </Link>
            <Link href="/about" className="text-orange-600 hover:text-orange-700 underline">
              About Nuclear AO3
            </Link>
            <Link href="/help" className="text-orange-600 hover:text-orange-700 underline">
              Help & FAQ
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