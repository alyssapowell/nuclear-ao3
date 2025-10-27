import React from 'react';

export default function ContactPage() {
  return (
    <div className="min-h-screen bg-white">
      <div className="max-w-4xl mx-auto px-4 py-8">
        <h1 className="text-3xl font-bold text-gray-900 mb-8">Contact Us</h1>
        
        <div className="space-y-8">
          <section>
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">Get in Touch</h2>
            <p className="text-gray-700 mb-6">
              We're here to help! Whether you have questions, need technical support, want to report an issue, 
              or have suggestions for improvement, we want to hear from you. Choose the contact method that 
              works best for your situation.
            </p>
          </section>

          <div className="grid md:grid-cols-2 gap-8">
            <section className="bg-blue-50 border border-blue-200 rounded-lg p-6">
              <h2 className="text-xl font-semibold text-blue-900 mb-4">
                📧 General Support
              </h2>
              <p className="text-blue-800 mb-4">
                For general questions, account issues, or technical support:
              </p>
              <div className="space-y-2">
                <p className="text-blue-800 font-medium">support@nuclear-ao3.org</p>
                <p className="text-blue-700 text-sm">
                  • Response time: 1-2 business days<br/>
                  • We respond to all inquiries<br/>
                  • Include as much detail as possible
                </p>
              </div>
            </section>

            <section className="bg-red-50 border border-red-200 rounded-lg p-6">
              <h2 className="text-xl font-semibold text-red-900 mb-4">
                🚨 Report Issues
              </h2>
              <p className="text-red-800 mb-4">
                For content violations, harassment, or urgent safety concerns:
              </p>
              <div className="space-y-2">
                <p className="text-red-800 font-medium">abuse@nuclear-ao3.org</p>
                <p className="text-red-700 text-sm">
                  • Priority response: Within 24 hours<br/>
                  • All reports reviewed confidentially<br/>
                  • Include links and context when possible
                </p>
              </div>
            </section>

            <section className="bg-green-50 border border-green-200 rounded-lg p-6">
              <h2 className="text-xl font-semibold text-green-900 mb-4">
                ♿ Accessibility Support
              </h2>
              <p className="text-green-800 mb-4">
                For accessibility issues or accommodation requests:
              </p>
              <div className="space-y-2">
                <p className="text-green-800 font-medium">accessibility@nuclear-ao3.org</p>
                <p className="text-green-700 text-sm">
                  • Response time: 2 business days<br/>
                  • High priority for critical access issues<br/>
                  • Include your assistive technology details
                </p>
              </div>
            </section>

            <section className="bg-purple-50 border border-purple-200 rounded-lg p-6">
              <h2 className="text-xl font-semibold text-purple-900 mb-4">
                💡 Feature Requests
              </h2>
              <p className="text-purple-800 mb-4">
                For suggestions, feature ideas, or community feedback:
              </p>
              <div className="space-y-2">
                <p className="text-purple-800 font-medium">features@nuclear-ao3.org</p>
                <p className="text-purple-700 text-sm">
                  • Response time: 3-5 business days<br/>
                  • All suggestions considered<br/>
                  • Community input helps guide development
                </p>
              </div>
            </section>
          </div>

          <section>
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">Alternative Contact Methods</h2>
            <div className="space-y-6">
              
              <div className="border border-gray-200 rounded-lg p-6">
                <h3 className="text-lg font-medium text-gray-900 mb-3">📝 Anonymous Feedback</h3>
                <p className="text-gray-700 mb-3">
                  Prefer to share feedback without providing your contact information?
                </p>
                <a 
                  href="/feedback" 
                  className="inline-block bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700 transition-colors"
                >
                  Submit Anonymous Feedback
                </a>
              </div>

              <div className="border border-gray-200 rounded-lg p-6">
                <h3 className="text-lg font-medium text-gray-900 mb-3">🐛 Bug Reports</h3>
                <p className="text-gray-700 mb-3">
                  Found a technical issue? Help us improve by reporting bugs with detailed information:
                </p>
                <ul className="text-gray-600 text-sm space-y-1 mb-3">
                  <li>• What you were trying to do</li>
                  <li>• What happened vs. what you expected</li>
                  <li>• Your browser and operating system</li>
                  <li>• Steps to reproduce the issue</li>
                  <li>• Screenshots if applicable</li>
                </ul>
                <p className="text-gray-700">
                  Send bug reports to: <span className="font-medium">support@nuclear-ao3.org</span>
                </p>
              </div>

              <div className="border border-gray-200 rounded-lg p-6">
                <h3 className="text-lg font-medium text-gray-900 mb-3">📰 Media & Press</h3>
                <p className="text-gray-700 mb-3">
                  Journalists, researchers, or media professionals seeking information about Nuclear AO3:
                </p>
                <div className="space-y-2">
                  <p className="text-gray-700">
                    Email: <span className="font-medium">press@nuclear-ao3.org</span>
                  </p>
                  <p className="text-gray-600 text-sm">
                    Please include your media outlet, deadline, and specific information needed.
                  </p>
                </div>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">Response Expectations</h2>
            <div className="bg-gray-50 rounded-lg p-6">
              <div className="grid md:grid-cols-2 gap-6">
                <div>
                  <h3 className="text-lg font-medium text-gray-900 mb-3">What to Expect</h3>
                  <ul className="text-gray-700 space-y-2">
                    <li>• Personal response from our team</li>
                    <li>• Acknowledgment within stated timeframes</li>
                    <li>• Follow-up if additional information is needed</li>
                    <li>• Respect for your privacy and confidentiality</li>
                  </ul>
                </div>
                
                <div>
                  <h3 className="text-lg font-medium text-gray-900 mb-3">Response Times</h3>
                  <ul className="text-gray-700 space-y-2">
                    <li>• <strong>Urgent safety issues:</strong> Within 24 hours</li>
                    <li>• <strong>General support:</strong> 1-2 business days</li>
                    <li>• <strong>Accessibility issues:</strong> 2 business days</li>
                    <li>• <strong>Feature requests:</strong> 3-5 business days</li>
                  </ul>
                </div>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">Before You Contact Us</h2>
            <p className="text-gray-700 mb-4">
              You might find quick answers to common questions in our resources:
            </p>
            <div className="grid md:grid-cols-3 gap-4">
              <a 
                href="/help" 
                className="block border border-gray-200 rounded-lg p-4 hover:bg-gray-50 transition-colors"
              >
                <h3 className="font-medium text-gray-900 mb-2">📚 Help & FAQ</h3>
                <p className="text-gray-600 text-sm">
                  Common questions and step-by-step guides
                </p>
              </a>
              
              <a 
                href="/privacy" 
                className="block border border-gray-200 rounded-lg p-4 hover:bg-gray-50 transition-colors"
              >
                <h3 className="font-medium text-gray-900 mb-2">🔒 Privacy Policy</h3>
                <p className="text-gray-600 text-sm">
                  How we protect and handle your data
                </p>
              </a>
              
              <a 
                href="/terms" 
                className="block border border-gray-200 rounded-lg p-4 hover:bg-gray-50 transition-colors"
              >
                <h3 className="font-medium text-gray-900 mb-2">📋 Terms of Service</h3>
                <p className="text-gray-600 text-sm">
                  Community guidelines and platform rules
                </p>
              </a>
            </div>
          </section>

          <section>
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">Community Guidelines for Contact</h2>
            <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-6">
              <p className="text-yellow-800 mb-4">
                To help us assist you effectively, please:
              </p>
              <ul className="text-yellow-800 space-y-2">
                <li>• Be respectful and patient with our team</li>
                <li>• Provide clear, detailed information about your issue</li>
                <li>• Use the appropriate contact method for your inquiry type</li>
                <li>• Allow sufficient time for response per our stated timelines</li>
                <li>• Follow up politely if you don't receive a response within expected timeframes</li>
              </ul>
            </div>
          </section>

          <section>
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">Privacy & Confidentiality</h2>
            <p className="text-gray-700 mb-4">
              We take your privacy seriously in all communications:
            </p>
            <ul className="text-gray-700 space-y-2">
              <li>• Your contact information is never shared with third parties</li>
              <li>• Support conversations remain confidential</li>
              <li>• We only use your information to respond to your inquiry</li>
              <li>• Anonymous feedback options are available for sensitive topics</li>
              <li>• All staff are trained on privacy and confidentiality practices</li>
            </ul>
          </section>

          <section className="border-t border-gray-200 pt-8">
            <div className="bg-blue-50 border border-blue-200 rounded-lg p-6">
              <h2 className="text-xl font-semibold text-blue-900 mb-3">
                Thank You for Reaching Out
              </h2>
              <p className="text-blue-800">
                Your feedback, questions, and reports help us build a better platform for everyone. 
                We're committed to responding thoughtfully and working together to address your needs 
                and improve the Nuclear AO3 experience.
              </p>
            </div>
          </section>
        </div>
      </div>
    </div>
  );
}