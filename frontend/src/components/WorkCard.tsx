'use client';

import Link from 'next/link';

interface Tag {
  name: string;
  category: string;
  quality_score?: number;
  is_canonical?: boolean;
}

interface WorkCardProps {
  work: {
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
    relationships: Tag[];
    characters: Tag[];
    freeform_tags: Tag[];
    fandoms: Tag[];
    kudos_count?: number;
    bookmark_count?: number;
    hit_count?: number;
    comment_count?: number;
    tag_quality_score?: number;
    missing_tag_suggestions?: string[];
  };
  showEnhancedFeatures?: boolean;
}

export default function WorkCard({ work, showEnhancedFeatures = true }: WorkCardProps) {
  const formatDate = (dateString: string) => {
    if (!dateString) {
      return 'Unknown date';
    }
    
    const date = new Date(dateString);
    if (isNaN(date.getTime())) {
      return 'Invalid date';
    }
    
    return date.toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric'
    });
  };

  const formatNumber = (num: number) => {
    if (num >= 1000000) return `${(num / 1000000).toFixed(1)}M`;
    if (num >= 1000) return `${(num / 1000).toFixed(1)}K`;
    return num.toString();
  };

  const getQualityColor = (score: number) => {
    if (score >= 0.8) return 'text-green-600';
    if (score >= 0.6) return 'text-yellow-600';
    return 'text-red-600';
  };

  const getStatusColor = (status: string) => {
    if (!status) return 'bg-gray-100 text-gray-800';
    switch (status.toLowerCase()) {
      case 'complete': return 'bg-green-100 text-green-800';
      case 'work in progress': return 'bg-blue-100 text-blue-800';
      case 'hiatus': return 'bg-yellow-100 text-yellow-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  };

  const getRatingColor = (rating: string) => {
    if (!rating) return 'bg-gray-100 text-gray-800';
    switch (rating.toLowerCase()) {
      case 'general audiences': return 'bg-green-100 text-green-800';
      case 'teen and up audiences': return 'bg-blue-100 text-blue-800';
      case 'mature': return 'bg-orange-100 text-orange-800';
      case 'explicit': return 'bg-red-100 text-red-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  };

  const TagGroup = ({ tags, category }: { tags: Tag[]; category: string }) => {
    if (!tags || tags.length === 0) return null;

    const getCategoryIcon = (cat: string) => {
      switch (cat) {
        case 'relationship': return '💕';
        case 'character': return '👤';
        case 'freeform': return '🏷️';
        case 'fandom': return '📚';
        default: return '🔖';
      }
    };

    return (
      <div className="flex flex-wrap gap-1">
        {tags.map((tag, index) => (
          <Link
            key={index}
            href={`/works?${category}=${encodeURIComponent(tag.name)}`}
            className="inline-flex items-center px-2 py-1 bg-gray-100 hover:bg-gray-200 text-gray-700 text-xs rounded-full transition-colors"
          >
            <span className="mr-1">{getCategoryIcon(tag.category)}</span>
            <span>{tag.name}</span>
            {showEnhancedFeatures && tag.quality_score !== undefined && (
              <span className={`ml-1 text-xs ${getQualityColor(tag.quality_score)}`}>
                {Math.round(tag.quality_score * 100)}%
              </span>
            )}
            {showEnhancedFeatures && tag.is_canonical && (
              <span className="ml-1 text-xs text-green-600">✓</span>
            )}
          </Link>
        ))}
      </div>
    );
  };

  return (
    <div className="bg-white border border-gray-200 rounded-lg p-6 hover:shadow-md transition-shadow">
      <div className="flex justify-between items-start mb-3">
        <div className="flex-1">
          <Link href={`/works/${work.id}`} className="text-lg font-semibold text-blue-600 hover:text-blue-800">
            {work.title}
          </Link>
          <p className="text-sm text-gray-600">by {work.author}</p>
        </div>
        
        {showEnhancedFeatures && work.tag_quality_score !== undefined && (
          <div className="ml-4 text-right">
            <div className="text-xs text-gray-500">Tag Quality</div>
            <div className={`text-sm font-medium ${getQualityColor(work.tag_quality_score)}`}>
              {Math.round(work.tag_quality_score * 100)}%
            </div>
          </div>
        )}
      </div>

      <div className="flex flex-wrap gap-2 mb-3">
        <span className={`px-2 py-1 text-xs rounded-full ${getRatingColor(work.rating)}`}>
          {work.rating}
        </span>
        <span className={`px-2 py-1 text-xs rounded-full ${getStatusColor(work.status)}`}>
          {work.status}
        </span>
        <span className="px-2 py-1 text-xs bg-gray-100 text-gray-700 rounded-full">
          {work.language}
        </span>
      </div>

      {work.summary && (
        <p className="text-sm text-gray-700 mb-3 line-clamp-3">
          {work.summary}
        </p>
      )}

      <div className="space-y-2 mb-4">
        {work.fandoms && work.fandoms.length > 0 && (
          <div>
            <span className="text-xs font-medium text-gray-500 mb-1 block">Fandoms:</span>
            <TagGroup tags={work.fandoms} category="fandoms" />
          </div>
        )}

        {work.relationships && work.relationships.length > 0 && (
          <div>
            <span className="text-xs font-medium text-gray-500 mb-1 block">Relationships:</span>
            <TagGroup tags={work.relationships} category="relationships" />
          </div>
        )}

        {work.characters && work.characters.length > 0 && (
          <div>
            <span className="text-xs font-medium text-gray-500 mb-1 block">Characters:</span>
            <TagGroup tags={work.characters} category="characters" />
          </div>
        )}

        {work.freeform_tags && work.freeform_tags.length > 0 && (
          <div>
            <span className="text-xs font-medium text-gray-500 mb-1 block">Additional Tags:</span>
            <TagGroup tags={work.freeform_tags} category="freeform_tags" />
          </div>
        )}
      </div>

      {showEnhancedFeatures && work.missing_tag_suggestions && work.missing_tag_suggestions.length > 0 && (
        <div className="mb-4 p-3 bg-blue-50 border border-blue-200 rounded-lg">
          <div className="text-xs font-medium text-blue-800 mb-1">💡 Smart Suggestions:</div>
          <div className="flex flex-wrap gap-1">
            {work.missing_tag_suggestions.map((suggestion, index) => (
              <span key={index} className="inline-flex items-center px-2 py-1 bg-blue-100 text-blue-800 text-xs rounded-full">
                <span className="mr-1">+</span>
                {suggestion}
              </span>
            ))}
          </div>
        </div>
      )}

      <div className="flex justify-between items-center text-sm text-gray-500 border-t border-gray-100 pt-3">
        <div className="flex items-center space-x-4">
          <span>{formatNumber(work.word_count)} words</span>
          <span>
            {work.chapter_count}/{work.max_chapters || '?'} chapters
          </span>
          {work.kudos_count !== undefined && (
            <span>❤️ {formatNumber(work.kudos_count)}</span>
          )}
          {work.bookmark_count !== undefined && (
            <span>🔖 {formatNumber(work.bookmark_count)}</span>
          )}
          {work.comment_count !== undefined && (
            <span>💬 {formatNumber(work.comment_count)}</span>
          )}
        </div>
        
        <div className="text-xs">
          <div>Published: {formatDate(work.published_date)}</div>
          {work.updated_date !== work.published_date && (
            <div>Updated: {formatDate(work.updated_date)}</div>
          )}
        </div>
      </div>
    </div>
  );
}