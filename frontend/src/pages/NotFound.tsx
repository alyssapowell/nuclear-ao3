import { Link } from 'react-router-dom';

export default function NotFoundPage() {
  return (
    <div className="max-w-2xl mx-auto px-4 py-16 text-center">
      <h1 className="text-3xl font-bold mb-4">Page not found</h1>
      <p className="text-gray-700 mb-6">There's nothing at this address. It may have been moved or deleted.</p>
      <Link to="/" className="text-blue-700 underline">Go to the home page</Link>
    </div>
  );
}
