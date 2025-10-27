'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import SearchForm from '@/components/SearchForm';
import SearchResults from '@/components/SearchResults';
import SmartRecommendations, { SmartRecommendation } from '@/components/SmartRecommendations';

interface Work {
  id: string;
  title: string;
  author: string;
  summary?: string;
  word_count: number;
  chapter_count: number;
  max_chapters?: number;
  rating: string;
  status: string;
  language: string;
  published_date: string;
  updated_date: string;
  relationships: Array<{ name: string; category: string }>;
  characters: Array<{ name: string; category: string }>;
  freeform_tags: Array<{ name: string; category: string }>;
  fandoms: Array<{ name: string; category: string }>;
  kudos_count?: number;
  bookmark_count?: number;
  hit_count?: number;
  comment_count?: number;
  tag_quality_score?: number;
  missing_tag_suggestions?: string[];
}



export default function HomePage() {
  const router = useRouter();
  const [searchResults, setSearchResults] = useState<Work[]>([]);
  const [recommendations, setRecommendations] = useState<SmartRecommendation[]>([]);

  const handleWorkClick = (workId: string, workTitle?: string) => {
    router.push(`/works/${workId}`);
  };

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      {/* Hero Section */}
      <div className="text-center mb-12">
        <h1 className="text-4xl font-bold text-gray-900 mb-4">
          Welcome to Nuclear AO3
        </h1>
        <p className="text-xl text-gray-600 max-w-2xl mx-auto">
          A fast, modern archive of our own. Discover amazing fanworks with enhanced search, 
          smart recommendations, and powerful filtering.
        </p>
      </div>

      {/* Search Section */}
      <section className="mb-8">
        <SearchForm 
          onResults={setSearchResults} 
          onRecommendations={setRecommendations}
          className="max-w-4xl mx-auto"
        />
      </section>

      {/* Recommendations */}
      {recommendations.length > 0 && (
        <section className="mb-8">
          <SmartRecommendations recommendations={recommendations} />
        </section>
      )}

      {/* Results */}
      {searchResults.length > 0 && (
        <section>
          <SearchResults results={searchResults} onWorkClick={handleWorkClick} />
        </section>
      )}

      {/* Recent Works or Featured Content */}
      {searchResults.length === 0 && (
        <section className="bg-white rounded-lg shadow-sm p-8 text-center">
          <h2 className="text-2xl font-semibold text-gray-900 mb-4">
            Start Your Search
          </h2>
          <p className="text-gray-600 mb-6">
            Use the search form above to find works by title, author, tags, or browse by fandom.
          </p>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6 text-left">
            <div className="bg-orange-50 p-6 rounded-lg">
              <h3 className="font-semibold text-orange-900 mb-2">🔍 Enhanced Search</h3>
              <p className="text-orange-800 text-sm">
                Search with intelligent tag suggestions and filtering options.
              </p>
            </div>
            <div className="bg-blue-50 p-6 rounded-lg">
              <h3 className="font-semibold text-blue-900 mb-2">⚡ Fast Performance</h3>
              <p className="text-blue-800 text-sm">
                Optimized for speed with advanced caching and search algorithms.
              </p>
            </div>
            <div className="bg-green-50 p-6 rounded-lg">
              <h3 className="font-semibold text-green-900 mb-2">🔍 Smart Tag Suggestions</h3>
              <p className="text-green-800 text-sm">
                Pattern-based suggestions to help improve your tagging consistency and completeness.
              </p>
            </div>
          </div>
        </section>
      )}
    </div>
  );
}