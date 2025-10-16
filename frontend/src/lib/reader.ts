// Reader preferences and state management

import { ReaderPreferences, DEFAULT_READER_PREFERENCES, READER_PRESETS, ReadingProgress } from '@/types/reader';

const READER_PREFS_KEY = 'nuclear-ao3-reader-preferences';
const READING_PROGRESS_KEY = 'nuclear-ao3-reading-progress';

export class ReaderManager {
  private preferences: ReaderPreferences;
  private listeners: Set<(preferences: ReaderPreferences) => void> = new Set();

  constructor() {
    this.preferences = this.loadPreferences();
    this.applyAutoNightMode();
  }

  // Preferences management
  getPreferences(): ReaderPreferences {
    return { ...this.preferences };
  }

  updatePreferences(updates: Partial<ReaderPreferences>): void {
    this.preferences = { ...this.preferences, ...updates };
    this.savePreferences();
    this.notifyListeners();
  }

  applyPreset(presetName: string): void {
    const preset = READER_PRESETS[presetName];
    if (preset) {
      this.updatePreferences(preset);
    }
  }

  resetToDefaults(): void {
    this.preferences = { ...DEFAULT_READER_PREFERENCES };
    this.savePreferences();
    this.notifyListeners();
  }

  // Auto night mode
  private applyAutoNightMode(): void {
    if (!this.preferences.autoNightMode) return;

    const now = new Date();
    const currentTime = now.getHours() * 100 + now.getMinutes();
    const startTime = this.timeStringToNumber(this.preferences.nightModeStart);
    const endTime = this.timeStringToNumber(this.preferences.nightModeEnd);

    let shouldBeDark = false;
    if (startTime > endTime) {
      // Night mode crosses midnight (e.g., 20:00 to 07:00)
      shouldBeDark = currentTime >= startTime || currentTime < endTime;
    } else {
      // Night mode within same day
      shouldBeDark = currentTime >= startTime && currentTime < endTime;
    }

    if (shouldBeDark && this.preferences.theme !== 'dark') {
      this.updatePreferences({ theme: 'dark' });
    } else if (!shouldBeDark && this.preferences.theme === 'dark') {
      this.updatePreferences({ theme: 'light' });
    }
  }

  private timeStringToNumber(timeString: string): number {
    const [hours, minutes] = timeString.split(':').map(Number);
    return hours * 100 + minutes;
  }

  // Reading progress tracking
  saveReadingProgress(workId: string, progress: Partial<ReadingProgress>): void {
    const existing = this.getReadingProgress(workId);
    const updated: ReadingProgress = {
      ...existing,
      workId,
      ...progress,
      lastRead: new Date().toISOString(),
    };

    const allProgress = this.getAllReadingProgress();
    allProgress[workId] = updated;
    
    localStorage.setItem(READING_PROGRESS_KEY, JSON.stringify(allProgress));
  }

  getReadingProgress(workId: string): ReadingProgress {
    const allProgress = this.getAllReadingProgress();
    return allProgress[workId] || {
      workId,
      position: 0,
      lastRead: new Date().toISOString(),
      totalTimeRead: 0,
    };
  }

  private getAllReadingProgress(): Record<string, ReadingProgress> {
    try {
      const stored = localStorage.getItem(READING_PROGRESS_KEY);
      return stored ? JSON.parse(stored) : {};
    } catch {
      return {};
    }
  }

  // CSS generation for reader mode
  generateReaderCSS(): string {
    const p = this.preferences;
    
    const fontFamilyMap = {
      'system': '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
      'serif': 'Georgia, "Times New Roman", serif',
      'sans-serif': '"Helvetica Neue", Arial, sans-serif',
      'dyslexic': '"Comic Neue", "Comic Sans MS", "Trebuchet MS", "Lucida Grande", sans-serif',
      'hyperlegible': '"Arial", "Helvetica", sans-serif',
    };

    const themeMap = {
      'light': { bg: '#ffffff', text: '#1a1a1a', link: '#e97316' },
      'dark': { bg: '#0f0f0f', text: '#e5e5e5', link: '#fb923c' },
      'sepia': { bg: '#f7f3e9', text: '#5c4b37', link: '#c2410c' },
      'high-contrast': { bg: '#000000', text: '#ffffff', link: '#ffff00' },
    };

    const theme = themeMap[p.theme as keyof typeof themeMap] || themeMap.light;
    const marginMap = {
      'compact': '1rem',
      'comfortable': '2rem',
      'spacious': '3rem',
    };

    return `
      /* Nuclear Reader Styles */
      body {
        background-color: ${theme.bg} !important;
        transition: background-color 0.3s ease !important;
      }
      
      .reader-content {
        font-family: ${fontFamilyMap[p.fontFamily]} !important;
        font-size: ${p.fontSize}px !important;
        line-height: ${p.lineHeight} !important;
        letter-spacing: ${p.letterSpacing}em !important;
        text-align: ${p.textAlign} !important;
        background-color: ${theme.bg} !important;
        color: ${theme.text} !important;
        max-width: ${p.contentWidth}% !important;
        margin: 0 auto !important;
        padding: ${marginMap[p.margin]} !important;
        min-height: 100vh;
        transition: all 0.3s ease !important;
      }
      
      .reader-content * {
        color: ${theme.text} !important;
        background-color: transparent !important;
      }
      
      .reader-content a {
        color: ${theme.link};
        text-decoration: underline;
        text-decoration-thickness: 2px;
        text-underline-offset: 3px;
      }
      
      .reader-content h1,
      .reader-content h2,
      .reader-content h3,
      .reader-content h4,
      .reader-content h5,
      .reader-content h6 {
        line-height: 1.2;
        margin-top: ${p.paragraphSpacing * 2}em;
        margin-bottom: ${p.paragraphSpacing}em;
      }
      
      .reader-content p {
        margin-bottom: ${p.paragraphSpacing}em;
      }
      
      .reader-content blockquote {
        border-left: 4px solid ${theme.link};
        padding-left: 1em;
        margin: ${p.paragraphSpacing}em 0;
        font-style: italic;
      }
      
      .reader-content code {
        background-color: ${p.theme === 'dark' ? '#2a2a2a' : '#f5f5f5'};
        padding: 0.2em 0.4em;
        border-radius: 4px;
        font-family: "Monaco", "Consolas", monospace;
      }
      
      ${p.highContrast ? `
        .reader-content {
          -webkit-font-smoothing: auto;
          -moz-osx-font-smoothing: auto;
        }
      ` : ''}
      
      ${p.reduceMotion ? `
        .reader-content * {
          animation-duration: 0.01ms !important;
          animation-iteration-count: 1 !important;
          transition-duration: 0.01ms !important;
        }
      ` : ''}
      
      ${p.focusIndicators ? `
        .reader-content *:focus {
          outline: 3px solid ${theme.link};
          outline-offset: 2px;
        }
      ` : ''}
    `;
  }

  // Persistence
  private loadPreferences(): ReaderPreferences {
    if (typeof window === 'undefined') {
      return { ...DEFAULT_READER_PREFERENCES };
    }
    
    try {
      const stored = localStorage.getItem(READER_PREFS_KEY);
      if (stored) {
        const parsed = JSON.parse(stored);
        return { ...DEFAULT_READER_PREFERENCES, ...parsed };
      }
    } catch (error) {
      console.warn('Failed to load reader preferences:', error);
    }
    return { ...DEFAULT_READER_PREFERENCES };
  }

  private savePreferences(): void {
    try {
      localStorage.setItem(READER_PREFS_KEY, JSON.stringify(this.preferences));
    } catch (error) {
      console.warn('Failed to save reader preferences:', error);
    }
  }

  // Event system
  subscribe(callback: (preferences: ReaderPreferences) => void): () => void {
    this.listeners.add(callback);
    return () => this.listeners.delete(callback);
  }

  private notifyListeners(): void {
    this.listeners.forEach(callback => callback(this.preferences));
  }
}

// Global reader manager instance
export const readerManager = new ReaderManager();

// React hook will be exported from a separate file to avoid bundling React in the core reader manager