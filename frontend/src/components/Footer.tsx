import Link from 'next/link';

export default function Footer() {
  return (
    <footer className="bg-slate-900 text-slate-300 py-12">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-8">
          {/* About Section */}
          <div className="col-span-1 md:col-span-2">
            <h3 className="text-white font-semibold text-lg mb-4">Nuclear AO3</h3>
            <p className="text-slate-400 mb-4 max-w-md">
              A modern, fast, and user-friendly archive of transformative works. 
              Built with enhanced search, accessibility features, and a focus on reader experience.
            </p>
            <div className="flex space-x-4">
              <Link href="/about" className="text-slate-400 hover:text-white transition-colors">
                About
              </Link>
              <Link href="/terms" className="text-slate-400 hover:text-white transition-colors">
                Terms of Service
              </Link>
              <Link href="/privacy" className="text-slate-400 hover:text-white transition-colors">
                Privacy Policy
              </Link>
            </div>
          </div>

          {/* Quick Links */}
          <div>
            <h4 className="text-white font-medium mb-4">Browse</h4>
            <ul className="space-y-2">
              <li>
                <Link href="/works" className="text-slate-400 hover:text-white transition-colors">
                  All Works
                </Link>
              </li>
              <li>
                <Link href="/search" className="text-slate-400 hover:text-white transition-colors">
                  Advanced Search
                </Link>
              </li>
              <li>
                <Link href="/collections" className="text-slate-400 hover:text-white transition-colors">
                  Collections
                </Link>
              </li>
              <li>
                <Link href="/series" className="text-slate-400 hover:text-white transition-colors">
                  Series
                </Link>
              </li>
            </ul>
          </div>

          {/* Help & Support */}
          <div>
            <h4 className="text-white font-medium mb-4">Support</h4>
            <ul className="space-y-2">
              <li>
                <Link href="/help" className="text-slate-400 hover:text-white transition-colors">
                  Help & FAQ
                </Link>
              </li>
              <li>
                <Link href="/accessibility" className="text-slate-400 hover:text-white transition-colors">
                  Accessibility
                </Link>
              </li>
              <li>
                <Link href="/contact" className="text-slate-400 hover:text-white transition-colors">
                  Contact Us
                </Link>
              </li>
              <li>
                <Link href="/feedback" className="text-slate-400 hover:text-white transition-colors">
                  Feedback
                </Link>
              </li>
            </ul>
          </div>
        </div>

        {/* Bottom Section */}
        <div className="border-t border-slate-700 mt-8 pt-8 flex flex-col sm:flex-row justify-between items-center">
          <div className="text-slate-400 text-sm mb-4 sm:mb-0">
            © {new Date().getFullYear()} Nuclear AO3. All rights reserved.
          </div>
          <div className="flex items-center space-x-4 text-sm">
            <span className="text-slate-500">Version 1.0.0</span>
            <Link href="/api-docs" className="text-slate-400 hover:text-white transition-colors">
              API Docs
            </Link>
            <Link href="/attributions" className="text-slate-400 hover:text-white transition-colors">
              Attributions
            </Link>
          </div>
        </div>
      </div>
    </footer>
  );
}