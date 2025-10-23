'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { useParams } from 'next/navigation';
import Comments from '@/components/Comments';
import KudosButton from '@/components/KudosButton';
import { BookmarkButton } from '@/components/BookmarkButton';
import { GiftButton } from '@/components/GiftButton';
import { GiftList } from '@/components/GiftList';
import { SubscriptionButton } from '@/components/SubscriptionButton';
import RespectfulExportButton from '@/components/RespectfulExportButton';
import ChapterNavigation from '@/components/ChapterNavigation';
import ReaderControls from '@/components/ReaderControls';
import { getWork, getWorkChapters } from '@/lib/api';
import { useAuth } from '@/lib/auth';
import { useReaderPreferences } from '@/hooks/useReaderPreferences';
import { useReadingProgress } from '@/hooks/useReadingProgress';

interface Work {
  id: string;
  title: string;
  summary: string;
  language: string;
  rating: string;
  category: string[];
  warnings: string[];
  fandoms: string[];
  characters: string[];
  relationships: string[];
  freeform_tags: string[];
  word_count: number;
  chapter_count: number;
  max_chapters?: number;
  is_complete: boolean;
  status: string;
  published_at: string;
  updated_at: string;
  hits: number;
  kudos: number;
  comments: number;
  bookmarks: number;
  offline_reading_override?: 'files_and_pwa' | 'pwa_only' | 'none' | 'use_default';
  author_default_offline_reading?: 'files_and_pwa' | 'pwa_only' | 'none';
}

interface Author {
  pseud_id: string;
  pseud_name: string;
  user_id: string;
  username: string;
  is_anonymous: boolean;
}

interface Chapter {
  id: string;
  work_id: string;
  number: number;
  title: string;
  summary: string;
  notes: string;
  end_notes: string;
  content: string;
  word_count: number;
  status: string;
  published_at: string;
  created_at: string;
  updated_at: string;
}

export default function WorkPage() {
  const params = useParams();
  const workId = params.id as string;
  const { user, token } = useAuth();
  
  const [work, setWork] = useState<Work | null>(null);
  const [authors, setAuthors] = useState<Author[]>([]);
  const [chapters, setChapters] = useState<Chapter[]>([]);
  const [currentChapter, setCurrentChapter] = useState<number>(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showReaderControls, setShowReaderControls] = useState(false);
  const [gifts, setGifts] = useState([]);

  const { preferences } = useReaderPreferences();
  const { progress, saveProgress } = useReadingProgress(workId, chapters.length);

  // Check if current user is an author of this work
  const isAuthor = user && authors.some(author => author.user_id === user.id);


  useEffect(() => {
    fetchWork();
    fetchChapters();
  }, [workId]);

  const fetchWork = async () => {
    try {
      // Basic UUID validation
      const uuidRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
      if (!uuidRegex.test(workId)) {
        setError('Invalid Work ID. Work IDs must be in UUID format.');
        setLoading(false);
        return;
      }

      const token = localStorage.getItem('auth_token') || localStorage.getItem('token');
      const data = await getWork(workId, token || undefined);
      setWork(data.work);
      setAuthors(data.authors || []);
    } catch (err) {
      if (err instanceof Error && err.message.includes('400')) {
        setError('Work not found. Please check the Work ID.');
      } else {
        setError(err instanceof Error ? err.message : 'Failed to load work');
      }
    } finally {
      setLoading(false);
    }
  };

  const fetchChapters = async () => {
    try {
      const token = localStorage.getItem('token');
      const data = await getWorkChapters(workId, token || undefined);
      setChapters(data.chapters || []);
    } catch (err) {
      console.error('Failed to load chapters:', err);
    }
  };



  const getRatingClass = (rating: string) => {
    const ratingClasses = {
      'general': 'bg-green-100 text-green-800',
      'teen': 'bg-blue-100 text-blue-800',
      'mature': 'bg-yellow-100 text-yellow-800',
      'explicit': 'bg-red-100 text-red-800',
      'not_rated': 'bg-gray-100 text-gray-800',
    };
    return ratingClasses[rating as keyof typeof ratingClasses] || ratingClasses.not_rated;
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    });
  };

  if (loading) {
    return (
      <div className="max-w-4xl mx-auto px-4 py-8">
        <div className="animate-pulse">
          <div className="h-8 bg-slate-200 rounded mb-4"></div>
          <div className="h-4 bg-slate-200 rounded mb-2"></div>
          <div className="h-4 bg-slate-200 rounded mb-8"></div>
          <div className="h-64 bg-slate-200 rounded"></div>
        </div>
      </div>
    );
  }

  if (error || !work) {
    return (
      <div className="max-w-4xl mx-auto px-4 py-8">
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
          {error || 'Work not found'}
        </div>
      </div>
    );
  }

  const currentChapterData = chapters.find(c => c.number === currentChapter) || chapters[0];

  // Apply reader preferences to content styling
  const readerStyles = preferences.readerMode ? {
    fontFamily: preferences.fontFamily === 'system' ? 'system-ui' : 
                preferences.fontFamily === 'serif' ? 'Georgia, serif' :
                preferences.fontFamily === 'sans-serif' ? 'Arial, sans-serif' :
                preferences.fontFamily === 'dyslexic' ? 'OpenDyslexic, sans-serif' : 'system-ui',
    fontSize: `${preferences.fontSize}px`,
    lineHeight: preferences.lineHeight,
    maxWidth: `${preferences.contentWidth}%`,
    textAlign: preferences.textAlign as 'left' | 'justify'
  } : {};

  const themeClasses = preferences.readerMode 
    ? `theme-${preferences.theme} ${preferences.highContrast ? 'high-contrast' : ''} ${preferences.reduceMotion ? 'reduce-motion' : ''}`
    : '';

  return (
    <div className={`max-w-4xl mx-auto px-4 py-8 ${themeClasses}`}>
      {/* Work Header */}
      <div className="bg-white rounded-lg shadow-sm border border-slate-200 p-6 mb-6">
        <div className="flex justify-between items-start mb-4">
          <div className="flex-1">
            <h1 className="text-3xl font-bold text-slate-900 mb-2">{work.title}</h1>
            <div className="flex flex-wrap items-center gap-2 text-sm text-slate-600">
              <span>by</span>
              {authors.map((author, index) => (
                <span key={author.pseud_id || index}>
                  {author.is_anonymous ? (
                    <span className="text-slate-500">Anonymous</span>
                  ) : (
                    <Link 
                      href={`/users/${author.username}`}
                      className="text-orange-600 hover:text-orange-700 font-medium"
                    >
                      {author.pseud_name}
                    </Link>
                  )}
                  {index < authors.length - 1 && ', '}
                </span>
              ))}
            </div>
            
            <GiftList 
              workId={work.id}
              authToken={token}
            />
          </div>
          
          <div className="flex flex-col items-end text-sm text-slate-600">
            <div>Published: {formatDate(work.published_at)}</div>
            {work.updated_at !== work.published_at && (
              <div>Updated: {formatDate(work.updated_at)}</div>
            )}
          </div>
        </div>

        {/* Tags and Metadata */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 mb-4">
          {/* Left Column */}
          <div className="space-y-3">
            <div>
              <span className="text-sm font-medium text-slate-700">Rating:</span>
              <span className={`ml-2 px-2 py-1 text-xs font-medium rounded ${getRatingClass(work.rating)}`}>
                {work.rating.replace('_', ' ').replace(/\b\w/g, l => l.toUpperCase())}
              </span>
            </div>
            
            {work.warnings && work.warnings.length > 0 && (
              <div>
                <span className="text-sm font-medium text-slate-700">Warnings:</span>
                <div className="mt-1 flex flex-wrap gap-1">
                  {work.warnings.map((warning, index) => (
                    <span 
                      key={index}
                      className="px-2 py-1 text-xs bg-yellow-100 text-yellow-800 rounded"
                    >
                      {warning}
                    </span>
                  ))}
                </div>
              </div>
            )}

            {work.fandoms && work.fandoms.length > 0 && (
              <div>
                <span className="text-sm font-medium text-slate-700">Fandoms:</span>
                <div className="mt-1 flex flex-wrap gap-1">
                  {work.fandoms.map((fandom, index) => (
                    <Link
                      key={index}
                      href={`/works?fandom=${encodeURIComponent(fandom)}`}
                      className="px-2 py-1 text-xs bg-blue-100 text-blue-800 rounded hover:bg-blue-200"
                    >
                      {fandom}
                    </Link>
                  ))}
                </div>
              </div>
            )}
          </div>

          {/* Right Column */}
          <div className="space-y-3">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-sm">
              <div>
                <span className="text-slate-700">Words:</span>
                <span className="ml-1 font-medium">{work.word_count.toLocaleString()}</span>
              </div>
              <div>
                <span className="text-slate-700">Chapters:</span>
                <span className="ml-1 font-medium">
                  {work.chapter_count}{work.max_chapters ? `/${work.max_chapters}` : '/?'}
                </span>
              </div>
              <div>
                <span className="text-slate-700">Hits:</span>
                <span className="ml-1 font-medium">{work.hits.toLocaleString()}</span>
              </div>
              <div>
                <span className="text-slate-700">Kudos:</span>
                <span className="ml-1 font-medium">{work.kudos || 0}</span>
              </div>
            </div>

            <div className="flex flex-col sm:flex-row items-start sm:items-center space-y-2 sm:space-y-0 sm:space-x-2">
              <KudosButton 
                workId={work.id}
                initialKudos={work.kudos}
                authToken={token}
              />
              
              <BookmarkButton 
                workId={work.id}
                authToken={token}
                onBookmarkChange={(isBookmarked) => {
                  // Optionally update UI or show notification
                  console.log('Bookmark status changed:', isBookmarked);
                }}
              />

              <GiftButton
                workId={work.id}
                isAuthor={isAuthor || false}
                onGiftCreated={(gift) => {
                  setGifts(prev => [...prev, gift]);
                }}
              />

              <SubscriptionButton
                type="work"
                targetId={work.id}
                targetName={work.title}
              />

              <RespectfulExportButton
                workId={work.id}
                workTitle={work.title}
                authToken={token}
                authorOfflinePreference={
                  work.offline_reading_override === 'use_default' || !work.offline_reading_override
                    ? work.author_default_offline_reading || 'pwa_only'
                    : work.offline_reading_override
                }
                isAuthor={isAuthor}
              />
            </div>
          </div>
        </div>

        {/* Summary */}
        {work.summary && (
          <div className="border-t border-slate-200 pt-4">
            <h3 className="text-sm font-medium text-slate-700 mb-2">Summary</h3>
            <div className="prose prose-sm text-slate-600">
              {work.summary.split('\n').map((line, index) => (
                <p key={index}>{line}</p>
              ))}
            </div>
          </div>
        )}

        {/* Additional Tags */}
        {((work.characters && work.characters.length > 0) || (work.relationships && work.relationships.length > 0) || (work.freeform_tags && work.freeform_tags.length > 0)) && (
          <div className="border-t border-slate-200 pt-4 mt-4">
            {work.relationships && work.relationships.length > 0 && (
              <div className="mb-3">
                <span className="text-sm font-medium text-slate-700">Relationships:</span>
                <div className="mt-1 flex flex-wrap gap-1">
                  {work.relationships.map((rel, index) => (
                    <Link
                      key={index}
                      href={`/works?relationship=${encodeURIComponent(rel)}`}
                      className="px-2 py-1 text-xs bg-pink-100 text-pink-800 rounded hover:bg-pink-200"
                    >
                      {rel}
                    </Link>
                  ))}
                </div>
              </div>
            )}

            {work.characters && work.characters.length > 0 && (
              <div className="mb-3">
                <span className="text-sm font-medium text-slate-700">Characters:</span>
                <div className="mt-1 flex flex-wrap gap-1">
                  {work.characters.map((char, index) => (
                    <Link
                      key={index}
                      href={`/works?character=${encodeURIComponent(char)}`}
                      className="px-2 py-1 text-xs bg-green-100 text-green-800 rounded hover:bg-green-200"
                    >
                      {char}
                    </Link>
                  ))}
                </div>
              </div>
            )}

            {work.freeform_tags && work.freeform_tags.length > 0 && (
              <div>
                <span className="text-sm font-medium text-slate-700">Additional Tags:</span>
                <div className="mt-1 flex flex-wrap gap-1">
                  {work.freeform_tags.map((tag, index) => (
                    <Link
                      key={index}
                      href={`/works?tag=${encodeURIComponent(tag)}`}
                      className="px-2 py-1 text-xs bg-purple-100 text-purple-800 rounded hover:bg-purple-200"
                    >
                      {tag}
                    </Link>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Reader Controls */}
      <div className="bg-white rounded-lg shadow-sm border border-slate-200 p-4 mb-6">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-slate-900">Reading Experience</h3>
          <button
            onClick={() => setShowReaderControls(!showReaderControls)}
            className="px-3 py-1 text-sm bg-orange-100 text-orange-800 rounded hover:bg-orange-200 transition-colors"
          >
            {showReaderControls ? 'Hide Controls' : 'Customize Reading'}
          </button>
        </div>
        
        {showReaderControls && (
          <ReaderControls className="mb-4" />
        )}

        {/* Enhanced Chapter Navigation */}
        {chapters.length > 1 && (
          <ChapterNavigation
            chapters={chapters.map(ch => ({
              id: ch.id,
              title: ch.title || `Chapter ${ch.number}`,
              wordCount: ch.word_count
            }))}
            currentChapter={currentChapter - 1}
            onChapterChange={(index) => {
              setCurrentChapter(index + 1);
              saveProgress(index, 0);
            }}
            className="relative"
          />
        )}
      </div>

      {/* Chapter Content */}
      {currentChapterData && (
        <div className="bg-white rounded-lg shadow-sm border border-slate-200 p-6">
          {currentChapterData.title && (
            <h2 className="text-xl font-semibold text-slate-900 mb-4">
              Chapter {currentChapterData.number}: {currentChapterData.title}
            </h2>
          )}
          
          {currentChapterData.summary && (
            <div className="mb-4 p-3 bg-slate-50 rounded">
              <h4 className="text-sm font-medium text-slate-700 mb-1">Chapter Summary</h4>
              <div className="text-sm text-slate-600">
                {currentChapterData.summary.split('\n').map((line, index) => (
                  <p key={index}>{line}</p>
                ))}
              </div>
            </div>
          )}

          {currentChapterData.notes && (
            <div className="mb-4 p-3 bg-blue-50 rounded">
              <h4 className="text-sm font-medium text-blue-800 mb-1">Notes</h4>
              <div className="text-sm text-blue-700">
                {currentChapterData.notes.split('\n').map((line, index) => (
                  <p key={index}>{line}</p>
                ))}
              </div>
            </div>
          )}

          <div 
            className={`prose prose-slate max-w-none ${preferences.readerMode ? 'reader-content' : ''}`}
            style={readerStyles}
          >
            {currentChapterData.content.split('\n\n').map((paragraph, index) => (
              <p 
                key={index} 
                className={`mb-4 ${preferences.margin === 'compact' ? 'mb-2' : preferences.margin === 'spacious' ? 'mb-6' : 'mb-4'}`}
                style={preferences.readerMode ? {
                  marginBottom: `${preferences.fontSize * 0.25}px`
                } : {}}
              >
                {paragraph}
              </p>
            ))}
          </div>

          {currentChapterData.end_notes && (
            <div className="mt-6 pt-4 border-t border-slate-200">
              <h4 className="text-sm font-medium text-slate-700 mb-2">End Notes</h4>
              <div className="text-sm text-slate-600">
                {currentChapterData.end_notes.split('\n').map((line, index) => (
                  <p key={index}>{line}</p>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {/* Comments Section */}
      <div className="mt-8">
        <Comments 
          workId={work.id}
          chapterId={currentChapterData?.id}
          allowComments={work.status === 'posted' || work.status === 'published'}
        />
      </div>
    </div>
  );
}