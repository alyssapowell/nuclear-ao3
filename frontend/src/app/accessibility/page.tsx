import React from 'react';

export default function AccessibilityPage() {
  return (
    <div className="min-h-screen bg-white">
      <div className="max-w-4xl mx-auto px-4 py-8">
        <h1 className="text-3xl font-bold text-gray-900 mb-8">Accessibility Statement</h1>
        
        <div className="space-y-8">
          <section>
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">Our Commitment</h2>
            <p className="text-gray-700 mb-4">
              Nuclear AO3 is committed to ensuring digital accessibility for all users, including those with disabilities. 
              We strive to provide an inclusive experience that allows everyone to access, navigate, and interact with our platform effectively.
            </p>
            <p className="text-gray-700">
              We continuously work to improve the accessibility of our website and maintain compliance with established guidelines 
              to ensure our platform is usable by the widest possible audience.
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">Accessibility Standards</h2>
            <p className="text-gray-700 mb-4">
              Nuclear AO3 aims to conform to the <strong>Web Content Accessibility Guidelines (WCAG) 2.1 Level AA</strong> standards. 
              These guidelines explain how to make web content accessible to users with disabilities and provide a better user experience for everyone.
            </p>
            <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
              <h3 className="text-lg font-medium text-blue-900 mb-2">WCAG 2.1 AA Compliance</h3>
              <p className="text-blue-800 text-sm">
                We follow the four fundamental principles of accessibility: content must be Perceivable, Operable, 
                Understandable, and Robust (POUR). Our development process includes accessibility testing and review 
                to ensure compliance with these standards.
              </p>
            </div>
          </section>

          <section>
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">Accessibility Features</h2>
            <div className="grid md:grid-cols-2 gap-6">
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-3">Keyboard Navigation</h3>
                <ul className="text-gray-700 space-y-2">
                  <li>• Full keyboard accessibility for all interactive elements</li>
                  <li>• Logical tab order throughout the site</li>
                  <li>• Visible focus indicators</li>
                  <li>• Skip navigation links for efficient browsing</li>
                </ul>
              </div>
              
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-3">Screen Reader Support</h3>
                <ul className="text-gray-700 space-y-2">
                  <li>• Semantic HTML structure</li>
                  <li>• Proper heading hierarchy</li>
                  <li>• Alternative text for images and graphics</li>
                  <li>• ARIA labels and descriptions where needed</li>
                </ul>
              </div>
              
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-3">Visual Design</h3>
                <ul className="text-gray-700 space-y-2">
                  <li>• High contrast color combinations</li>
                  <li>• Resizable text up to 200% without loss of functionality</li>
                  <li>• Clear visual hierarchy and layout</li>
                  <li>• No reliance on color alone to convey information</li>
                </ul>
              </div>
              
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-3">Content & Forms</h3>
                <ul className="text-gray-700 space-y-2">
                  <li>• Clear, descriptive form labels</li>
                  <li>• Error messages that are easy to understand</li>
                  <li>• Consistent navigation and interaction patterns</li>
                  <li>• Plain language and clear instructions</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">Testing & Validation</h2>
            <p className="text-gray-700 mb-4">
              We regularly test our website using a variety of methods to ensure accessibility:
            </p>
            <div className="space-y-4">
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-2">Automated Testing</h3>
                <p className="text-gray-700">
                  We use automated accessibility testing tools integrated into our development workflow 
                  to catch common accessibility issues early in the development process.
                </p>
              </div>
              
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-2">Manual Testing</h3>
                <p className="text-gray-700">
                  Our team performs manual testing using screen readers (NVDA, JAWS, VoiceOver) and 
                  keyboard-only navigation to ensure real-world usability.
                </p>
              </div>
              
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-2">User Feedback</h3>
                <p className="text-gray-700">
                  We actively seek feedback from users with disabilities to understand their experiences 
                  and identify areas for improvement.
                </p>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">Assistive Technology Compatibility</h2>
            <p className="text-gray-700 mb-4">
              Nuclear AO3 is designed to work with the following assistive technologies:
            </p>
            <div className="grid md:grid-cols-3 gap-4">
              <div className="bg-gray-50 rounded-lg p-4">
                <h3 className="font-medium text-gray-900 mb-2">Screen Readers</h3>
                <ul className="text-sm text-gray-700 space-y-1">
                  <li>• NVDA (Windows)</li>
                  <li>• JAWS (Windows)</li>
                  <li>• VoiceOver (macOS/iOS)</li>
                  <li>• TalkBack (Android)</li>
                </ul>
              </div>
              
              <div className="bg-gray-50 rounded-lg p-4">
                <h3 className="font-medium text-gray-900 mb-2">Voice Control</h3>
                <ul className="text-sm text-gray-700 space-y-1">
                  <li>• Dragon NaturallySpeaking</li>
                  <li>• Voice Control (macOS)</li>
                  <li>• Voice Access (Android)</li>
                </ul>
              </div>
              
              <div className="bg-gray-50 rounded-lg p-4">
                <h3 className="font-medium text-gray-900 mb-2">Alternative Input</h3>
                <ul className="text-sm text-gray-700 space-y-1">
                  <li>• Switch devices</li>
                  <li>• Head/eye tracking</li>
                  <li>• Alternative keyboards</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">Browser & Device Support</h2>
            <p className="text-gray-700 mb-4">
              Our accessibility features are supported across modern browsers and devices:
            </p>
            <div className="bg-gray-50 rounded-lg p-4">
              <div className="grid md:grid-cols-2 gap-4">
                <div>
                  <h3 className="font-medium text-gray-900 mb-2">Desktop Browsers</h3>
                  <ul className="text-sm text-gray-700 space-y-1">
                    <li>• Chrome (latest 2 versions)</li>
                    <li>• Firefox (latest 2 versions)</li>
                    <li>• Safari (latest 2 versions)</li>
                    <li>• Edge (latest 2 versions)</li>
                  </ul>
                </div>
                
                <div>
                  <h3 className="font-medium text-gray-900 mb-2">Mobile Platforms</h3>
                  <ul className="text-sm text-gray-700 space-y-1">
                    <li>• iOS Safari (latest 2 versions)</li>
                    <li>• Android Chrome (latest 2 versions)</li>
                    <li>• Mobile assistive technologies</li>
                  </ul>
                </div>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">Known Issues & Limitations</h2>
            <p className="text-gray-700 mb-4">
              We are continuously working to improve accessibility. Currently known issues include:
            </p>
            <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
              <ul className="text-yellow-800 space-y-2">
                <li>• Some complex interactive components are being enhanced for better screen reader support</li>
                <li>• File upload components are being improved for better keyboard accessibility</li>
                <li>• Advanced search filters are being optimized for assistive technology compatibility</li>
              </ul>
              <p className="text-yellow-800 text-sm mt-3">
                We are actively working on these improvements and expect to resolve them in upcoming releases.
              </p>
            </div>
          </section>

          <section>
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">Feedback & Support</h2>
            <p className="text-gray-700 mb-4">
              We welcome your feedback on the accessibility of Nuclear AO3. If you encounter any accessibility barriers 
              or have suggestions for improvement, please let us know.
            </p>
            
            <div className="grid md:grid-cols-2 gap-6">
              <div className="border border-gray-200 rounded-lg p-4">
                <h3 className="text-lg font-medium text-gray-900 mb-2">Report Issues</h3>
                <p className="text-gray-700 text-sm mb-3">
                  If you experience difficulty accessing any part of our website, please contact us with details about:
                </p>
                <ul className="text-gray-700 text-sm space-y-1 mb-3">
                  <li>• The specific page or feature you're having trouble with</li>
                  <li>• Your assistive technology (name and version)</li>
                  <li>• Your browser and operating system</li>
                  <li>• A description of the problem</li>
                </ul>
                <a href="/contact" className="text-blue-600 hover:underline font-medium">
                  Contact Support →
                </a>
              </div>
              
              <div className="border border-gray-200 rounded-lg p-4">
                <h3 className="text-lg font-medium text-gray-900 mb-2">Response Commitment</h3>
                <p className="text-gray-700 text-sm mb-3">
                  We are committed to responding to accessibility feedback promptly:
                </p>
                <ul className="text-gray-700 text-sm space-y-1 mb-3">
                  <li>• Initial response within 2 business days</li>
                  <li>• Detailed investigation and response within 5 business days</li>
                  <li>• Critical accessibility issues prioritized for immediate attention</li>
                </ul>
                <a href="/feedback" className="text-blue-600 hover:underline font-medium">
                  Send Feedback →
                </a>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">Continuous Improvement</h2>
            <p className="text-gray-700 mb-4">
              Accessibility is an ongoing commitment. We regularly:
            </p>
            <ul className="text-gray-700 space-y-2 mb-4">
              <li>• Review and update our accessibility practices</li>
              <li>• Conduct regular accessibility audits</li>
              <li>• Train our development team on accessibility best practices</li>
              <li>• Stay informed about new accessibility standards and techniques</li>
              <li>• Incorporate user feedback into our development process</li>
            </ul>
            
            <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
              <p className="text-blue-800 text-sm">
                <strong>Last updated:</strong> {new Date().toLocaleDateString('en-US', { 
                  year: 'numeric', 
                  month: 'long', 
                  day: 'numeric' 
                })}
              </p>
              <p className="text-blue-800 text-sm mt-2">
                This accessibility statement will be reviewed and updated regularly as we continue to improve 
                the accessibility of Nuclear AO3.
              </p>
            </div>
          </section>
        </div>
      </div>
    </div>
  );
}