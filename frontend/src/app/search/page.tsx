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



export default function SearchPage() {
  const router = useRouter();
  const [searchResults, setSearchResults] = useState<Work[]>([]);
  const [recommendations, setRecommendations] = useState<SmartRecommendation[]>([]);
  const [loading, setLoading] = useState(false);
  const [hasSearched, setHasSearched] = useState(false);

  const handleSearch = (results: Work[]) => {
    setSearchResults(results);
    setHasSearched(true);
  };

  const handleWorkClick = (workId: string, workTitle?: string) => {
    router.push(`/works/${workId}`);
  };

  return (
    <div className="max-w-7xl mx-auto px-4 py-8">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-gray-900 mb-4">Enhanced Search</h1>
        <p className="text-gray-600">
          Search the archive with powerful filtering and get intelligent recommendations.
        </p>
      </div>

      {/* Search Form */}
      <div className="mb-8">
        <SearchForm 
          onResults={handleSearch}
          onRecommendations={setRecommendations}
          className="bg-white rounded-lg shadow-sm border p-6"
        />
      </div>

      {/* Search Tips */}
      {!hasSearched && (
        <div className="bg-blue-50 border border-blue-200 rounded-lg p-6 mb-8">
          <h2 className="text-lg font-semibold text-blue-900 mb-3">Search Tips</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm text-blue-800">
            <div>
              <h3 className="font-medium mb-2">Basic Search:</h3>
              <ul className="space-y-1">
                <li>• Use quotes for exact phrases: "hello world"</li>
                <li>• Use - to exclude terms: cats -dogs</li>
                <li>• Use * for wildcards: develop*</li>
              </ul>
            </div>
            <div>
              <h3 className="font-medium mb-2">Advanced Features:</h3>
              <ul className="space-y-1">
                <li>• Filter by word count, rating, status</li>
                <li>• Search within specific fandoms</li>
                <li>• Get smart tag consistency suggestions</li>
              </ul>
            </div>
          </div>
        </div>
      )}

      {/* Smart Recommendations */}
      {recommendations.length > 0 && (
        <div className="mb-8">
          <SmartRecommendations recommendations={recommendations} />
        </div>
      )}

      {/* Search Results */}
      {hasSearched && (
        <div>
          {loading ? (
            <div className="text-center py-12">
              <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-orange-600"></div>
              <p className="mt-2 text-gray-600">Searching...</p>
            </div>
          ) : searchResults.length > 0 ? (
            <div>
              <div className="flex justify-between items-center mb-6">
                <h2 className="text-xl font-semibold text-gray-900">
                  Search Results ({searchResults.length} works)
                </h2>
                <div className="flex gap-2">
                  <select className="px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-orange-500">
                    <optgroup label="Smart Sorting (Anti-Gaming)">
                      <option value="quality_score">🏆 Quality Score</option>
                      <option value="engagement_rate">📊 Engagement Rate</option>
                      <option value="comment_quality">💬 Discussion Quality</option>
                      <option value="discovery_boost">🔍 Discovery Boost</option>
                    </optgroup>
                    <optgroup label="Traditional Sorting">
                      <option value="relevance">Best Match</option>
                      <option value="updated_at">Recently Updated</option>
                      <option value="published_at">Recently Published</option>
                      <option value="word_count">Word Count</option>
                    </optgroup>
                    <optgroup label="Raw Numbers (Gaming-Prone)">
                      <option value="kudos">Most Kudos</option>
                      <option value="hits">Most Hits</option>
                      <option value="bookmarks">Most Bookmarks</option>
                      <option value="comments">Most Comments</option>
                    </optgroup>
                  </select>
                </div>
              </div>
              <SearchResults results={searchResults} onWorkClick={handleWorkClick} />
            </div>
          ) : (
            <div className="text-center py-12 bg-white rounded-lg shadow-sm">
              <h3 className="text-lg font-medium text-gray-900 mb-2">No works found</h3>
              <p className="text-gray-600 mb-4">
                Try adjusting your search terms or filters to find what you're looking for.
              </p>
              <div className="space-y-2 text-sm text-gray-500">
                <p>Suggestions:</p>
                <ul className="space-y-1">
                  <li>• Check your spelling</li>
                  <li>• Try broader search terms</li>
                  <li>• Remove some filters</li>
                  <li>• Use fewer tags in your search</li>
                </ul>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
