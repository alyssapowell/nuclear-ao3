'use client';

import { useState } from 'react';

export interface SmartRecommendation {
  type: 'missing_character' | 'missing_relationship' | 'tag_quality' | 'canonical_suggestion' | 'related_tags';
  title: string;
  description: string;
  suggestions: string[];
  confidence_score: number;
  category: string;
}

interface SmartRecommendationsProps {
  recommendations: SmartRecommendation[];
  onApplyRecommendation?: (recommendation: { tag: string; category: string; type: string }) => void;
  className?: string;
}

export default function SmartRecommendations({ 
  recommendations, 
  onApplyRecommendation,
  className = "" 
}: SmartRecommendationsProps) {
  const [expandedId, setExpandedId] = useState<number | null>(null);

  if (!recommendations || recommendations.length === 0) {
    return null;
  }

  const getRecommendationIcon = (type: string) => {
    switch (type) {
      case 'missing_character': return '👤';
      case 'missing_relationship': return '💕';
      case 'tag_quality': return '⭐';
      case 'canonical_suggestion': return '✅';
      case 'related_tags': return '🔗';
      default: return '💡';
    }
  };

  const getConfidenceColor = (score: number) => {
    if (score >= 0.8) return 'text-green-600 bg-green-100';
    if (score >= 0.6) return 'text-yellow-600 bg-yellow-100';
    return 'text-red-600 bg-red-100';
  };

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'missing_character': return 'bg-blue-50 border-blue-200';
      case 'missing_relationship': return 'bg-pink-50 border-pink-200';
      case 'tag_quality': return 'bg-yellow-50 border-yellow-200';
      case 'canonical_suggestion': return 'bg-green-50 border-green-200';
      case 'related_tags': return 'bg-purple-50 border-purple-200';
      default: return 'bg-gray-50 border-gray-200';
    }
  };

  const handleApplyRecommendation = (recommendation: SmartRecommendation, suggestion: string) => {
    if (onApplyRecommendation) {
      onApplyRecommendation({ 
        tag: suggestion, 
        category: recommendation.category, 
        type: recommendation.type 
      });
    }
  };

  return (
    <div className={`space-y-4 ${className}`}>
      <div className="flex items-center space-x-2">
        <span className="text-lg">🤖</span>
        <h3 className="text-lg font-semibold text-gray-900">Smart Recommendations</h3>
        <span className="text-sm text-gray-500">({recommendations.length})</span>
      </div>

      <div className="space-y-3">
        {recommendations.map((recommendation, index) => (
          <div
            key={index}
            className={`border rounded-lg p-4 ${getTypeColor(recommendation.type)}`}
          >
            <div className="flex items-start justify-between">
              <div className="flex-1">
                <div className="flex items-center space-x-3 mb-2">
                  <span className="text-xl">{getRecommendationIcon(recommendation.type)}</span>
                  <div>
                    <h4 className="font-medium text-gray-900">{recommendation.title}</h4>
                    <p className="text-sm text-gray-600">{recommendation.description}</p>
                  </div>
                </div>

                {recommendation.suggestions.length > 0 && (
                  <div className="mt-3">
                    <div className="flex flex-wrap gap-2">
                      {recommendation.suggestions.slice(0, expandedId === index ? undefined : 3).map((suggestion, suggestionIndex) => (
                        <button
                          key={suggestionIndex}
                          onClick={() => handleApplyRecommendation(recommendation, suggestion)}
                          className="inline-flex items-center px-3 py-1 bg-white border border-gray-300 rounded-full text-sm hover:bg-gray-50 hover:border-gray-400 transition-colors"
                        >
                          <span className="mr-1">+</span>
                          {suggestion}
                        </button>
                      ))}
                    </div>

                    {recommendation.suggestions.length > 3 && (
                      <button
                        onClick={() => setExpandedId(expandedId === index ? null : index)}
                        className="mt-2 text-sm text-blue-600 hover:text-blue-800"
                      >
                        {expandedId === index 
                          ? 'Show less' 
                          : `Show ${recommendation.suggestions.length - 3} more suggestions`
                        }
                      </button>
                    )}
                  </div>
                )}
              </div>

              <div className="ml-4">
                <div className={`px-2 py-1 rounded-full text-xs font-medium ${getConfidenceColor(recommendation.confidence_score)}`}>
                  {Math.round(recommendation.confidence_score * 100)}%
                </div>
              </div>
            </div>
          </div>
        ))}
      </div>

      <div className="text-xs text-gray-500 bg-gray-50 rounded-lg p-3">
        <div className="flex items-start space-x-2">
          <span>ℹ️</span>
          <div>
            <p className="font-medium">How Smart Recommendations Work:</p>
            <ul className="mt-1 space-y-1">
              <li>• <strong>Missing Characters:</strong> Detected from relationship tags (e.g., "Agatha/Reader" suggests adding "Reader" character)</li>
              <li>• <strong>Missing Relationships:</strong> Inferred from character combinations and common patterns</li>
              <li>• <strong>Tag Quality:</strong> Suggestions to improve tag accuracy and discoverability</li>
              <li>• <strong>Canonical Tags:</strong> Recommendations to use official, canonical tag versions</li>
              <li>• <strong>Related Tags:</strong> Popular tags often used together with your current tags</li>
            </ul>
          </div>
        </div>
      </div>
    </div>
  );
}