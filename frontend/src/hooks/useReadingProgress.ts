import { useState, useEffect, useCallback } from 'react';

interface ReadingProgress {
  workId: string;
  chapterIndex: number;
  scrollPosition: number;
  totalProgress: number; // 0-100
  lastRead: number; // timestamp
}

export function useReadingProgress(workId: string, totalChapters: number) {
  const [progress, setProgress] = useState<ReadingProgress | null>(null);
  const [scrollProgress, setScrollProgress] = useState(0);

  // Load progress from localStorage
  useEffect(() => {
    if (typeof window === 'undefined') return;
    
    const key = `reading-progress-${workId}`;
    const stored = localStorage.getItem(key);
    if (stored) {
      try {
        setProgress(JSON.parse(stored));
      } catch (e) {
        console.warn('Failed to parse reading progress:', e);
      }
    }
  }, [workId]);

  // Track scroll position for current chapter
  useEffect(() => {
    const handleScroll = () => {
      const scrollTop = window.scrollY;
      const docHeight = document.documentElement.scrollHeight - window.innerHeight;
      const scrollPercent = docHeight > 0 ? (scrollTop / docHeight) * 100 : 0;
      setScrollProgress(Math.min(100, Math.max(0, scrollPercent)));
    };

    window.addEventListener('scroll', handleScroll, { passive: true });
    handleScroll(); // Initial calculation
    
    return () => window.removeEventListener('scroll', handleScroll);
  }, []);

  // Save progress
  const saveProgress = useCallback((chapterIndex: number, scrollPosition?: number) => {
    const newProgress: ReadingProgress = {
      workId,
      chapterIndex,
      scrollPosition: scrollPosition ?? window.scrollY,
      totalProgress: ((chapterIndex + 1) / totalChapters) * 100,
      lastRead: Date.now()
    };

    setProgress(newProgress);
    
    const key = `reading-progress-${workId}`;
    if (typeof window !== 'undefined') {
      localStorage.setItem(key, JSON.stringify(newProgress));
    }
  }, [workId, totalChapters]);

  // Auto-save progress when chapter changes or scroll position changes significantly
  useEffect(() => {
    const saveTimer = setTimeout(() => {
      if (progress) {
        saveProgress(progress.chapterIndex, window.scrollY);
      }
    }, 2000); // Save after 2 seconds of no scrolling

    return () => clearTimeout(saveTimer);
  }, [scrollProgress, saveProgress, progress]);

  // Mark chapter as complete
  const markChapterComplete = useCallback((chapterIndex: number) => {
    saveProgress(chapterIndex, 0);
  }, [saveProgress]);

  // Get reading statistics
  const getReadingStats = useCallback(() => {
    if (!progress) return null;

    const chaptersRead = progress.chapterIndex + 1;
    const remainingChapters = totalChapters - chaptersRead;
    
    return {
      chaptersRead,
      remainingChapters,
      overallProgress: progress.totalProgress,
      currentChapterProgress: scrollProgress,
      lastReadDate: new Date(progress.lastRead),
    };
  }, [progress, totalChapters, scrollProgress]);

  return {
    progress,
    scrollProgress,
    saveProgress,
    markChapterComplete,
    getReadingStats,
  };
}