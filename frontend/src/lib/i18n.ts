// Nuclear AO3: Internationalization Support
// Multi-language support for global accessibility

export interface Translation {
  [key: string]: string | Translation;
}

export interface Translations {
  [lang: string]: Translation;
}

// Supported languages with their display names
export const SUPPORTED_LANGUAGES = {
  'en': 'English',
  'es': 'Español',
  'fr': 'Français', 
  'de': 'Deutsch',
  'it': 'Italiano',
  'pt': 'Português',
  'ru': 'Русский',
  'ja': '日本語',
  'ko': '한국어',
  'zh-CN': '简体中文',
  'zh-TW': '繁體中文',
  'ar': 'العربية'
} as const;

export type SupportedLanguage = keyof typeof SUPPORTED_LANGUAGES;

// Translation keys structure
export const translations: Translations = {
  en: {
    // Navigation
    nav: {
      works: 'Works',
      search: 'Search', 
      bookmarks: 'Bookmarks',
      collections: 'Collections',
      dashboard: 'Dashboard',
      profile: 'Profile',
      settings: 'Settings',
      logout: 'Log Out'
    },

    // Search & Filtering
    search: {
      title: 'Search Works',
      placeholder: 'Search for fanfiction...',
      filters: 'Filters',
      tags: 'Tags',
      rating: 'Rating',
      category: 'Category',
      status: 'Status',
      language: 'Language',
      wordCount: 'Word Count',
      sort: 'Sort by',
      results: {
        found: 'Found {{count}} works',
        none: 'No works found',
        loading: 'Searching...'
      }
    },

    // Smart Recommendations
    recommendations: {
      title: 'Smart Recommendations',
      missingCharacter: 'Missing Characters',
      missingRelationship: 'Missing Relationships', 
      tagQuality: 'Tag Quality Improvements',
      canonicalSuggestion: 'Canonical Tags',
      relatedTags: 'Related Tags',
      confidence: '{{percent}}% confidence',
      apply: 'Apply',
      howItWorks: 'How Smart Recommendations Work',
      description: {
        missingCharacter: 'Detected from relationship tags (e.g., "Agatha/Reader" suggests adding "Reader" character)',
        missingRelationship: 'Inferred from character combinations and common patterns',
        tagQuality: 'Suggestions to improve tag accuracy and discoverability', 
        canonicalSuggestion: 'Recommendations to use official, canonical tag versions',
        relatedTags: 'Popular tags often used together with your current tags'
      }
    },

    // Works
    works: {
      title: 'Browse Works',
      subtitle: 'Discover fanfiction with enhanced filtering and smart recommendations',
      byAuthor: 'by {{author}}',
      chapters: '{{current}}/{{total}} chapters',
      words: '{{count}} words',
      complete: 'Complete',
      inProgress: 'In Progress',
      lastUpdated: 'Updated {{date}}',
      published: 'Published {{date}}',
      rating: {
        'General Audiences': 'General Audiences',
        'Teen And Up Audiences': 'Teen And Up Audiences', 
        'Mature': 'Mature',
        'Explicit': 'Explicit',
        'Not Rated': 'Not Rated'
      },
      category: {
        'Gen': 'General',
        'M/M': 'M/M',
        'F/F': 'F/F', 
        'F/M': 'F/M',
        'Multi': 'Multi',
        'Other': 'Other'
      }
    },

    // Bookmarks
    bookmarks: {
      title: 'My Bookmarks',
      add: 'Bookmark',
      remove: 'Remove Bookmark',
      notes: 'Notes',
      private: 'Private',
      recommend: 'Recommend',
      tags: 'Bookmark Tags',
      collections: 'Collections'
    },

    // Reading Experience
    reader: {
      fontSize: 'Font Size',
      fontFamily: 'Font Family',
      lineHeight: 'Line Height', 
      theme: 'Theme',
      progress: 'Reading Progress',
      nextChapter: 'Next Chapter',
      prevChapter: 'Previous Chapter',
      toc: 'Table of Contents',
      fullscreen: 'Fullscreen',
      settings: 'Reading Settings'
    },

    // PWA & Offline
    offline: {
      title: 'Offline Mode',
      subtitle: 'Continue reading without internet',
      downloadWork: 'Download for Offline',
      downloadProgress: 'Downloading... {{percent}}%',
      downloaded: 'Available Offline',
      storage: 'Storage Used: {{size}}',
      manage: 'Manage Downloads'
    },

    // Common UI
    common: {
      loading: 'Loading...',
      error: 'An error occurred',
      retry: 'Try Again', 
      save: 'Save',
      cancel: 'Cancel',
      close: 'Close',
      next: 'Next',
      previous: 'Previous',
      more: 'Show More',
      less: 'Show Less',
      clear: 'Clear',
      apply: 'Apply',
      reset: 'Reset',
      delete: 'Delete',
      edit: 'Edit',
      share: 'Share'
    },

    // Time & Dates
    time: {
      justNow: 'just now',
      minutesAgo: '{{count}} minutes ago',
      hoursAgo: '{{count}} hours ago', 
      daysAgo: '{{count}} days ago',
      weeksAgo: '{{count}} weeks ago',
      monthsAgo: '{{count}} months ago',
      yearsAgo: '{{count}} years ago'
    },

    // Errors
    errors: {
      networkError: 'Network connection error',
      serverError: 'Server error, please try again',
      notFound: 'Page not found',
      unauthorized: 'You need to log in',
      forbidden: 'Access denied',
      validationError: 'Please check your input'
    }
  },

  es: {
    nav: {
      works: 'Obras',
      search: 'Buscar',
      bookmarks: 'Marcadores',
      collections: 'Colecciones',
      dashboard: 'Panel',
      profile: 'Perfil',
      settings: 'Configuración',
      logout: 'Cerrar Sesión'
    },
    search: {
      title: 'Buscar Obras',
      placeholder: 'Buscar fanfiction...',
      filters: 'Filtros',
      tags: 'Etiquetas',
      rating: 'Clasificación',
      category: 'Categoría',
      status: 'Estado',
      language: 'Idioma',
      wordCount: 'Número de Palabras',
      sort: 'Ordenar por',
      results: {
        found: 'Se encontraron {{count}} obras',
        none: 'No se encontraron obras',
        loading: 'Buscando...'
      }
    }
    // ... More Spanish translations
  },

  fr: {
    nav: {
      works: 'Œuvres',
      search: 'Rechercher',
      bookmarks: 'Signets',
      collections: 'Collections',
      dashboard: 'Tableau de bord',
      profile: 'Profil',
      settings: 'Paramètres',
      logout: 'Se déconnecter'
    },
    search: {
      title: 'Rechercher des Œuvres',
      placeholder: 'Rechercher de la fanfiction...',
      filters: 'Filtres',
      tags: 'Étiquettes',
      rating: 'Classification',
      category: 'Catégorie',
      status: 'Statut',
      language: 'Langue',
      wordCount: 'Nombre de mots',
      sort: 'Trier par',
      results: {
        found: '{{count}} œuvres trouvées',
        none: 'Aucune œuvre trouvée',
        loading: 'Recherche...'
      }
    }
    // ... More French translations
  }

  // Additional languages would be added here...
};

// i18n utility class
export class I18n {
  private currentLanguage: SupportedLanguage = 'en';
  private fallbackLanguage: SupportedLanguage = 'en';

  constructor(initialLanguage?: SupportedLanguage) {
    if (initialLanguage && initialLanguage in SUPPORTED_LANGUAGES) {
      this.currentLanguage = initialLanguage;
    } else {
      // Auto-detect browser language
      this.currentLanguage = this.detectBrowserLanguage();
    }
  }

  private detectBrowserLanguage(): SupportedLanguage {
    if (typeof window === 'undefined') return 'en';
    
    const browserLang = navigator.language.toLowerCase();
    
    // Check for exact match first
    if (browserLang in SUPPORTED_LANGUAGES) {
      return browserLang as SupportedLanguage;
    }
    
    // Check for language prefix match (e.g., 'en-US' -> 'en')
    const langPrefix = browserLang.split('-')[0];
    for (const supportedLang of Object.keys(SUPPORTED_LANGUAGES)) {
      if (supportedLang.startsWith(langPrefix)) {
        return supportedLang as SupportedLanguage;
      }
    }
    
    return 'en'; // Fallback to English
  }

  setLanguage(language: SupportedLanguage): void {
    if (language in SUPPORTED_LANGUAGES) {
      this.currentLanguage = language;
      
      // Save to localStorage for persistence
      if (typeof window !== 'undefined') {
        localStorage.setItem('nuclear-ao3-language', language);
      }
    }
  }

  getCurrentLanguage(): SupportedLanguage {
    return this.currentLanguage;
  }

  getSupportedLanguages(): typeof SUPPORTED_LANGUAGES {
    return SUPPORTED_LANGUAGES;
  }

  // Get translation with parameter interpolation
  t(key: string, params?: Record<string, any>): string {
    let translation = this.getTranslation(key, this.currentLanguage);
    
    // Fallback to English if translation not found
    if (!translation && this.currentLanguage !== this.fallbackLanguage) {
      translation = this.getTranslation(key, this.fallbackLanguage);
    }
    
    // Return key if no translation found
    if (!translation) {
      console.warn(`Translation missing: ${key} for language ${this.currentLanguage}`);
      return key;
    }

    // Interpolate parameters
    if (params) {
      return this.interpolate(translation, params);
    }

    return translation;
  }

  private getTranslation(key: string, language: SupportedLanguage): string | null {
    const keys = key.split('.');
    let current: any = translations[language];

    for (const k of keys) {
      if (current && typeof current === 'object' && k in current) {
        current = current[k];
      } else {
        return null;
      }
    }

    return typeof current === 'string' ? current : null;
  }

  private interpolate(template: string, params: Record<string, any>): string {
    return template.replace(/\{\{(\w+)\}\}/g, (match, key) => {
      return params[key]?.toString() || match;
    });
  }

  // Format numbers according to locale
  formatNumber(num: number): string {
    if (typeof window === 'undefined') return num.toString();
    
    try {
      return new Intl.NumberFormat(this.currentLanguage).format(num);
    } catch (error) {
      return num.toString();
    }
  }

  // Format dates according to locale
  formatDate(date: Date, options?: Intl.DateTimeFormatOptions): string {
    if (typeof window === 'undefined') return date.toISOString();
    
    try {
      return new Intl.DateTimeFormat(this.currentLanguage, options).format(date);
    } catch (error) {
      return date.toLocaleDateString();
    }
  }

  // Format relative time (e.g., "2 hours ago")
  formatRelativeTime(date: Date): string {
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMinutes = Math.floor(diffMs / (1000 * 60));
    const diffHours = Math.floor(diffMinutes / 60);
    const diffDays = Math.floor(diffHours / 24);
    const diffWeeks = Math.floor(diffDays / 7);
    const diffMonths = Math.floor(diffDays / 30);
    const diffYears = Math.floor(diffDays / 365);

    if (diffMinutes < 1) {
      return this.t('time.justNow');
    } else if (diffMinutes < 60) {
      return this.t('time.minutesAgo', { count: diffMinutes });
    } else if (diffHours < 24) {
      return this.t('time.hoursAgo', { count: diffHours });
    } else if (diffDays < 7) {
      return this.t('time.daysAgo', { count: diffDays });
    } else if (diffWeeks < 4) {
      return this.t('time.weeksAgo', { count: diffWeeks });
    } else if (diffMonths < 12) {
      return this.t('time.monthsAgo', { count: diffMonths });
    } else {
      return this.t('time.yearsAgo', { count: diffYears });
    }
  }

  // Check if current language is RTL (Right-to-Left)
  isRTL(): boolean {
    const rtlLanguages = ['ar', 'he', 'fa', 'ur'];
    return rtlLanguages.includes(this.currentLanguage);
  }
}

// Global i18n instance
export const i18n = new I18n();

// React hook for using translations
import { useState, useEffect } from 'react';

export function useTranslation() {
  const [language, setLanguage] = useState(i18n.getCurrentLanguage());

  useEffect(() => {
    // Load saved language from localStorage
    if (typeof window !== 'undefined') {
      const savedLang = localStorage.getItem('nuclear-ao3-language') as SupportedLanguage;
      if (savedLang && savedLang in SUPPORTED_LANGUAGES) {
        i18n.setLanguage(savedLang);
        setLanguage(savedLang);
      }
    }
  }, []);

  const changeLanguage = (newLanguage: SupportedLanguage) => {
    i18n.setLanguage(newLanguage);
    setLanguage(newLanguage);
  };

  return {
    t: i18n.t.bind(i18n),
    language,
    changeLanguage,
    formatNumber: i18n.formatNumber.bind(i18n),
    formatDate: i18n.formatDate.bind(i18n),
    formatRelativeTime: i18n.formatRelativeTime.bind(i18n),
    isRTL: i18n.isRTL.bind(i18n),
    supportedLanguages: SUPPORTED_LANGUAGES
  };
}