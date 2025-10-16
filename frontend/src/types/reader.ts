// Reader mode and customization types

export interface ReaderPreferences {
  // Theme settings
  theme: 'light' | 'dark' | 'sepia' | 'high-contrast' | 'custom';
  customTheme?: {
    backgroundColor: string;
    textColor: string;
    linkColor: string;
  };
  
  // Typography settings
  fontFamily: 'system' | 'serif' | 'sans-serif' | 'dyslexic' | 'hyperlegible';
  fontSize: number; // 12-28px
  lineHeight: number; // 1.2-2.5
  letterSpacing: number; // 0-0.15em
  paragraphSpacing: number; // 0.5-2.0em
  
  // Layout settings
  contentWidth: number; // 60-100% of screen
  margin: 'compact' | 'comfortable' | 'spacious';
  textAlign: 'left' | 'justify';
  
  // Reading features
  readerMode: boolean;
  autoNightMode: boolean;
  nightModeStart: string; // "20:00"
  nightModeEnd: string; // "07:00"
  
  // Accessibility
  reduceMotion: boolean;
  highContrast: boolean;
  focusIndicators: boolean;
  screenReaderOptimized: boolean;
}

export interface ReadingProgress {
  workId: string;
  chapterId?: string;
  position: number; // scroll position or paragraph index
  lastRead: string; // ISO timestamp
  totalTimeRead: number; // seconds
}

export interface ReaderState {
  isReaderMode: boolean;
  preferences: ReaderPreferences;
  progress: ReadingProgress | null;
  isCustomizing: boolean;
}

// Default preferences optimized for readability
export const DEFAULT_READER_PREFERENCES: ReaderPreferences = {
  theme: 'light',
  fontFamily: 'system',
  fontSize: 16,
  lineHeight: 1.6,
  letterSpacing: 0,
  paragraphSpacing: 1.0,
  contentWidth: 75,
  margin: 'comfortable',
  textAlign: 'left',
  readerMode: false,
  autoNightMode: true,
  nightModeStart: '20:00',
  nightModeEnd: '07:00',
  reduceMotion: false,
  highContrast: false,
  focusIndicators: true,
  screenReaderOptimized: false,
};

// Preset configurations for quick setup
export const READER_PRESETS: Record<string, Partial<ReaderPreferences>> = {
  'default': DEFAULT_READER_PREFERENCES,
  'large-text': {
    fontSize: 20,
    lineHeight: 1.8,
    paragraphSpacing: 1.2,
    margin: 'spacious',
  },
  'dyslexia-friendly': {
    fontFamily: 'dyslexic',
    fontSize: 18,
    lineHeight: 1.8,
    letterSpacing: 0.05,
    paragraphSpacing: 1.2,
    textAlign: 'left',
  },
  'night-reader': {
    theme: 'dark',
    fontSize: 17,
    lineHeight: 1.7,
    autoNightMode: true,
  },
  'high-contrast': {
    theme: 'high-contrast',
    fontSize: 18,
    lineHeight: 1.8,
    highContrast: true,
    focusIndicators: true,
  },
  'minimal': {
    contentWidth: 65,
    margin: 'compact',
    fontSize: 15,
    lineHeight: 1.5,
  },
};