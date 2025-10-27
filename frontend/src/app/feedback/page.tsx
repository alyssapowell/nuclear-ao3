'use client';

import React, { useState } from 'react';

export default function FeedbackPage() {
  const [feedbackType, setFeedbackType] = useState('general');
  const [feedback, setFeedback] = useState('');
  const [email, setEmail] = useState('');
  const [includeEmail, setIncludeEmail] = useState(false);
  const [submitted, setSubmitted] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    // In a real implementation, this would send to an API endpoint
    // For now, we'll just simulate the submission
    console.log('Feedback submitted:', {
      type: feedbackType,
      feedback,
      email: includeEmail ? email : null,
      timestamp: new Date().toISOString()
    });
    
    setSubmitted(true);
    
    // Reset form after showing success message
    setTimeout(() => {
      setSubmitted(false);
      setFeedback('');
      setEmail('');
      setIncludeEmail(false);
      setFeedbackType('general');
    }, 3000);
  };

  if (submitted) {
    return (
      <div className="min-h-screen bg-white">
        <div className="max-w-2xl mx-auto px-4 py-8">
          <div className="text-center">
            <div className="bg-green-50 border border-green-200 rounded-lg p-8">
              <div className="text-4xl mb-4">✅</div>
              <h1 className="text-2xl font-bold text-green-900 mb-4">Thank You!</h1>
              <p className="text-green-800 mb-4">
                Your feedback has been submitted successfully. We appreciate you taking the time to help us improve Nuclear AO3.
              </p>
              <p className="text-green-700 text-sm">
                {includeEmail ? 
                  "We'll review your feedback and may follow up if we need additional information." :
                  "Your anonymous feedback will be reviewed by our team."
                }
              </p>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-white">
      <div className="max-w-4xl mx-auto px-4 py-8">
        <h1 className="text-3xl font-bold text-gray-900 mb-8">Send Feedback</h1>
        
        <div className="space-y-8">
          <section>
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">Help Us Improve</h2>
            <p className="text-gray-700 mb-6">
              Your feedback is invaluable in making Nuclear AO3 better for everyone. Whether you have suggestions, 
              bug reports, feature requests, or general comments, we want to hear from you. You can submit feedback 
              anonymously or include your contact information if you'd like a response.
            </p>
          </section>

          <div className="grid md:grid-cols-3 gap-6 mb-8">
            <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
              <h3 className="font-medium text-blue-900 mb-2">💡 Feature Requests</h3>
              <p className="text-blue-800 text-sm">
                Suggest new features or improvements to existing functionality
              </p>
            </div>
            
            <div className="bg-red-50 border border-red-200 rounded-lg p-4">
              <h3 className="font-medium text-red-900 mb-2">🐛 Bug Reports</h3>
              <p className="text-red-800 text-sm">
                Report issues, errors, or unexpected behavior you've encountered
              </p>
            </div>
            
            <div className="bg-green-50 border border-green-200 rounded-lg p-4">
              <h3 className="font-medium text-green-900 mb-2">📝 General Feedback</h3>
              <p className="text-green-800 text-sm">
                Share your thoughts, suggestions, or general comments about the platform
              </p>
            </div>
          </div>

          <form onSubmit={handleSubmit} className="space-y-6">
            <div>
              <label htmlFor="feedbackType" className="block text-sm font-medium text-gray-900 mb-2">
                What type of feedback are you sharing?
              </label>
              <select
                id="feedbackType"
                value={feedbackType}
                onChange={(e) => setFeedbackType(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              >
                <option value="general">General Feedback</option>
                <option value="feature">Feature Request</option>
                <option value="bug">Bug Report</option>
                <option value="usability">Usability Issue</option>
                <option value="accessibility">Accessibility Issue</option>
                <option value="performance">Performance Issue</option>
                <option value="content">Content or Community Issue</option>
                <option value="other">Other</option>
              </select>
            </div>

            <div>
              <label htmlFor="feedback" className="block text-sm font-medium text-gray-900 mb-2">
                Your Feedback <span className="text-red-500">*</span>
              </label>
              <textarea
                id="feedback"
                value={feedback}
                onChange={(e) => setFeedback(e.target.value)}
                rows={8}
                required
                placeholder="Please share your feedback in as much detail as possible. For bug reports, include steps to reproduce the issue and any error messages you saw."
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              />
              <p className="text-gray-500 text-sm mt-2">
                The more specific you can be, the better we can address your feedback.
              </p>
            </div>

            <div className="border-t border-gray-200 pt-6">
              <div className="flex items-start space-x-3">
                <input
                  type="checkbox"
                  id="includeEmail"
                  checked={includeEmail}
                  onChange={(e) => setIncludeEmail(e.target.checked)}
                  className="mt-1 h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                />
                <div className="flex-1">
                  <label htmlFor="includeEmail" className="text-sm font-medium text-gray-900">
                    I'd like to be contacted about this feedback
                  </label>
                  <p className="text-gray-500 text-sm">
                    Check this if you want us to be able to follow up with questions or updates. 
                    Otherwise, your feedback will be submitted anonymously.
                  </p>
                </div>
              </div>
            </div>

            {includeEmail && (
              <div>
                <label htmlFor="email" className="block text-sm font-medium text-gray-900 mb-2">
                  Your Email Address
                </label>
                <input
                  type="email"
                  id="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required={includeEmail}
                  placeholder="your.email@example.com"
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
                <p className="text-gray-500 text-sm mt-2">
                  We'll only use this email to respond to your feedback. We never share your contact information.
                </p>
              </div>
            )}

            <div className="bg-gray-50 rounded-lg p-4">
              <h3 className="text-sm font-medium text-gray-900 mb-2">Privacy Notice</h3>
              <ul className="text-gray-700 text-sm space-y-1">
                <li>• Your feedback helps us improve Nuclear AO3 for everyone</li>
                <li>• Anonymous feedback is completely anonymous - we can't trace it back to you</li>
                <li>• If you provide your email, we may contact you only about this specific feedback</li>
                <li>• We never share feedback or contact information with third parties</li>
                <li>• All feedback is reviewed by our team and treated confidentially</li>
              </ul>
            </div>

            <div className="flex justify-end">
              <button
                type="submit"
                disabled={!feedback.trim()}
                className="bg-blue-600 text-white px-6 py-2 rounded-lg hover:bg-blue-700 focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors"
              >
                Submit Feedback
              </button>
            </div>
          </form>

          <section className="border-t border-gray-200 pt-8">
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">Other Ways to Get Help</h2>
            <div className="grid md:grid-cols-2 gap-6">
              <div className="border border-gray-200 rounded-lg p-4">
                <h3 className="font-medium text-gray-900 mb-2">📞 Direct Contact</h3>
                <p className="text-gray-700 text-sm mb-3">
                  Need immediate help or want to discuss something directly?
                </p>
                <a 
                  href="/contact" 
                  className="text-blue-600 hover:underline text-sm font-medium"
                >
                  View Contact Options →
                </a>
              </div>
              
              <div className="border border-gray-200 rounded-lg p-4">
                <h3 className="font-medium text-gray-900 mb-2">❓ Help & FAQ</h3>
                <p className="text-gray-700 text-sm mb-3">
                  Check our help documentation for answers to common questions.
                </p>
                <a 
                  href="/help" 
                  className="text-blue-600 hover:underline text-sm font-medium"
                >
                  Browse Help Topics →
                </a>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">Feedback Guidelines</h2>
            <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-6">
              <p className="text-yellow-800 mb-4">
                To help us process your feedback effectively:
              </p>
              <ul className="text-yellow-800 space-y-2">
                <li>• Be as specific as possible about the issue or suggestion</li>
                <li>• For bugs, include steps to reproduce and what you expected to happen</li>
                <li>• For feature requests, explain the problem you're trying to solve</li>
                <li>• Include relevant details like browser, device, or account information when applicable</li>
                <li>• Be respectful and constructive in your feedback</li>
                <li>• One topic per feedback submission works best for our review process</li>
              </ul>
            </div>
          </section>
        </div>
      </div>
    </div>
  );
}