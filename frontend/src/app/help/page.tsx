import React from 'react';

export default function HelpPage() {
  return (
    <div className="min-h-screen bg-white">
      <div className="max-w-4xl mx-auto px-4 py-8">
        <h1 className="text-3xl font-bold text-gray-900 mb-8">Help & FAQ</h1>
        
        <div className="space-y-8">
          <section>
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">Getting Started</h2>
            <div className="space-y-4">
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-2">How do I create an account?</h3>
                <p className="text-gray-700">
                  Click the "Register" button in the top navigation. You'll need to provide a username, email, and password. 
                  All accounts are manually approved to maintain community quality.
                </p>
              </div>
              
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-2">What makes Nuclear AO3 different?</h3>
                <p className="text-gray-700">
                  Nuclear AO3 is built with privacy-first principles: end-to-end encryption, zero tracking, minimal data collection, 
                  and aggressive data deletion. Unlike traditional platforms, we protect your privacy by design.
                </p>
              </div>
              
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-2">How do I post my first work?</h3>
                <p className="text-gray-700">
                  Once your account is approved, navigate to "Works" and click "New Work". Fill out the form with your 
                  story details, tags, and content. You can save drafts and publish when ready.
                </p>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">Writing & Publishing</h2>
            <div className="space-y-4">
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-2">What file formats can I upload?</h3>
                <p className="text-gray-700">
                  You can paste text directly into the editor or upload HTML, Markdown, or plain text files. 
                  The platform supports rich formatting including italics, bold, and basic HTML tags.
                </p>
              </div>
              
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-2">How do I tag my work effectively?</h3>
                <p className="text-gray-700">
                  Use clear, descriptive tags for characters, relationships, genres, and warnings. Our auto-complete 
                  system will suggest popular tags. Tag warnings appropriately to help readers make informed choices.
                </p>
              </div>
              
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-2">Can I edit my work after publishing?</h3>
                <p className="text-gray-700">
                  Yes! You can edit your works at any time. Navigate to your work and click "Edit". 
                  Changes are saved immediately and visible to readers.
                </p>
              </div>
              
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-2">How do series work?</h3>
                <p className="text-gray-700">
                  Series let you group related works together. Create a series from the "Series" page, 
                  then add works to it when posting or editing. Readers can follow entire series for updates.
                </p>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">Privacy & Security</h2>
            <div className="space-y-4">
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-2">How is my data protected?</h3>
                <p className="text-gray-700">
                  Your data is encrypted end-to-end, we collect minimal information, and we aggressively delete 
                  unnecessary data. See our <a href="/privacy" className="text-blue-600 hover:underline">Privacy Policy</a> for complete details.
                </p>
              </div>
              
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-2">Can I make my profile private?</h3>
                <p className="text-gray-700">
                  Yes! Visit your Profile Privacy settings to control who can see your works, bookmarks, 
                  and profile information. You can make everything private or selectively share with registered users.
                </p>
              </div>
              
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-2">How do I delete my account?</h3>
                <p className="text-gray-700">
                  Contact us through the <a href="/contact" className="text-blue-600 hover:underline">Contact</a> page. 
                  Account deletion is permanent and removes all your data within 30 days, as outlined in our privacy policy.
                </p>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">Reading & Discovery</h2>
            <div className="space-y-4">
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-2">How do I find works to read?</h3>
                <p className="text-gray-700">
                  Use the search function to filter by tags, fandoms, characters, and relationships. 
                  You can also browse by recent updates, kudos, or bookmarks. Our pattern-based suggestions 
                  help you discover works similar to ones you've enjoyed.
                </p>
              </div>
              
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-2">What's the difference between kudos and bookmarks?</h3>
                <p className="text-gray-700">
                  Kudos are like a "like" - a quick way to show appreciation. Bookmarks save works to your 
                  personal reading list and can include private notes. Both help authors know their work is appreciated.
                </p>
              </div>
              
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-2">How do comments work?</h3>
                <p className="text-gray-700">
                  You can comment on works to share feedback with authors. Comments support basic formatting 
                  and can be threaded for conversations. Authors can moderate comments on their works.
                </p>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">Community Guidelines</h2>
            <div className="space-y-4">
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-2">What content is allowed?</h3>
                <p className="text-gray-700">
                  We welcome transformative fanfiction and original works. Content must be properly tagged, 
                  especially for mature themes. We prohibit harassment, spam, and non-transformative commercial content.
                </p>
              </div>
              
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-2">How do I report inappropriate content?</h3>
                <p className="text-gray-700">
                  Use the "Report" link on any work or comment that violates our terms. 
                  Reports are reviewed by our moderation team within 24-48 hours.
                </p>
              </div>
              
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-2">Can I block other users?</h3>
                <p className="text-gray-700">
                  Yes! You can mute users to hide their works and comments from your view. 
                  This is a client-side filter that doesn't affect the blocked user's experience.
                </p>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">Technical Issues</h2>
            <div className="space-y-4">
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-2">The site isn't loading properly. What should I do?</h3>
                <p className="text-gray-700">
                  Try refreshing the page, clearing your browser cache, or using a different browser. 
                  If issues persist, check our status page or contact support.
                </p>
              </div>
              
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-2">I'm having trouble with the editor. Any tips?</h3>
                <p className="text-gray-700">
                  The rich text editor works best in modern browsers. If you experience issues, 
                  try switching to HTML mode or pasting from a plain text editor. Save frequently as drafts.
                </p>
              </div>
              
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-2">Is there a mobile app?</h3>
                <p className="text-gray-700">
                  Not yet, but our website is fully responsive and works well on mobile devices. 
                  You can add it to your home screen for an app-like experience.
                </p>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-2xl font-semibold text-gray-900 mb-4">Still Need Help?</h2>
            <div className="bg-gray-50 rounded-lg p-6">
              <p className="text-gray-700 mb-4">
                Can't find what you're looking for? We're here to help!
              </p>
              <div className="space-y-2">
                <p className="text-gray-700">
                  • <a href="/contact" className="text-blue-600 hover:underline">Contact Support</a> - Get help from our team
                </p>
                <p className="text-gray-700">
                  • <a href="/feedback" className="text-blue-600 hover:underline">Send Feedback</a> - Suggest improvements or report bugs
                </p>
                <p className="text-gray-700">
                  • Check our community forums (coming soon) for user discussions
                </p>
              </div>
            </div>
          </section>
        </div>
      </div>
    </div>
  );
}