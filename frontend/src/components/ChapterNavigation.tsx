import { useState } from 'react';
import { useSwipeGestures } from '@/hooks/useSwipeGestures';

interface Chapter {
  id: string;
  title: string;
  wordCount: number;
}

interface ChapterNavigationProps {
  chapters: Chapter[];
  currentChapter: number;
  onChapterChange: (chapterIndex: number) => void;
  className?: string;
}

export default function ChapterNavigation({
  chapters,
  currentChapter,
  onChapterChange,
  className = ''
}: ChapterNavigationProps) {
  const [showChapterList, setShowChapterList] = useState(false);

  // Swipe gestures for chapter navigation
  useSwipeGestures({
    onSwipeLeft: () => {
      // Next chapter (swipe left to go forward)
      if (currentChapter < chapters.length - 1) {
        onChapterChange(currentChapter + 1);
      }
    },
    onSwipeRight: () => {
      // Previous chapter (swipe right to go back)
      if (currentChapter > 0) {
        onChapterChange(currentChapter - 1);
      }
    },
    threshold: 75 // Require more deliberate swipe for chapter change
  });

  const handlePrevious = () => {
    if (currentChapter > 0) {
      onChapterChange(currentChapter - 1);
    }
  };

  const handleNext = () => {
    if (currentChapter < chapters.length - 1) {
      onChapterChange(currentChapter + 1);
    }
  };

  const hasPrevious = currentChapter > 0;
  const hasNext = currentChapter < chapters.length - 1;

  return (
    <nav 
      className={`chapter-navigation ${className}`}
      aria-label="Chapter navigation"
    >
      {/* Main Navigation Controls */}
      <div className="flex justify-between items-center gap-4">
        {/* Previous Chapter */}
        <button
          onClick={handlePrevious}
          disabled={!hasPrevious}
          className={`btn btn-outline btn-sm ${!hasPrevious ? 'opacity-50 cursor-not-allowed' : ''}`}
          title={hasPrevious ? `Previous: ${chapters[currentChapter - 1]?.title}` : 'No previous chapter'}
        >
          <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
          <span className="hidden sm:inline">Previous</span>
        </button>

        {/* Chapter Selector */}
        <div className="flex items-center gap-2">
          <button
            onClick={() => setShowChapterList(!showChapterList)}
            className="btn btn-outline btn-sm"
            title="Select chapter"
          >
            <span className="text-sm">
              Chapter {currentChapter + 1} of {chapters.length}
            </span>
            <svg className="w-4 h-4 ml-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
            </svg>
          </button>

          {/* Mobile swipe hint */}
          <div className="hidden sm:block text-xs text-slate-500">
            📱 Swipe to navigate
          </div>
        </div>

        {/* Next Chapter */}
        <button
          onClick={handleNext}
          disabled={!hasNext}
          className={`btn btn-primary btn-sm ${!hasNext ? 'opacity-50 cursor-not-allowed' : ''}`}
          title={hasNext ? `Next: ${chapters[currentChapter + 1]?.title}` : 'No next chapter'}
        >
          <span className="hidden sm:inline">Next</span>
          <svg className="w-4 h-4 ml-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
          </svg>
        </button>
      </div>

      {/* Chapter List Dropdown */}
      {showChapterList && (
        <div className="absolute top-full left-0 right-0 mt-2 bg-white border border-slate-200 rounded-lg shadow-lg z-50 max-h-80 overflow-y-auto">
          <div className="p-2">
            <h3 className="font-semibold text-slate-900 text-sm mb-2 px-2">
              All Chapters
            </h3>
            <div className="space-y-1">
              {chapters.map((chapter, index) => (
                <button
                  key={chapter.id}
                  onClick={() => {
                    onChapterChange(index);
                    setShowChapterList(false);
                  }}
                  className={`w-full text-left px-3 py-2 rounded text-sm transition-colors ${
                    index === currentChapter
                      ? 'bg-orange-100 text-orange-900 border border-orange-200'
                      : 'hover:bg-slate-50 text-slate-700'
                  }`}
                >
                  <div className="flex justify-between items-start">
                    <div>
                      <div className="font-medium">
                        Chapter {index + 1}: {chapter.title}
                      </div>
                      <div className="text-xs text-slate-500 mt-1">
                        {chapter.wordCount.toLocaleString()} words
                      </div>
                    </div>
                    {index === currentChapter && (
                      <div className="text-orange-600 text-xs font-bold">
                        CURRENT
                      </div>
                    )}
                  </div>
                </button>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Progress Indicator */}
      <div className="mt-3">
        <div className="flex justify-between text-xs text-slate-500 mb-1">
          <span>Progress</span>
          <span>{Math.round(((currentChapter + 1) / chapters.length) * 100)}%</span>
        </div>
        <div className="w-full bg-slate-200 rounded-full h-2">
          <div
            className="bg-orange-500 h-2 rounded-full transition-all duration-300"
            style={{ width: `${((currentChapter + 1) / chapters.length) * 100}%` }}
          />
        </div>
      </div>

      {/* Keyboard Navigation Hint */}
      <div className="mt-2 text-xs text-slate-500 text-center">
        Use ← → arrow keys or swipe to navigate chapters
      </div>
    </nav>
  );
}