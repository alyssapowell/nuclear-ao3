/**
 * Database-driven crossover detection using tag relationships
 * Replaces hardcoded universe families with proper tag wrangling
 */

// API configuration
const API_BASE_URL = process.env.NEXT_PUBLIC_API_GATEWAY_URL || 'http://localhost:8084';

interface UniverseFamily {
  id: string;
  name: string;
  member_fandoms: string[];
}

interface CrossoverAnalysis {
  isCrossover: boolean;
  universes: string[];
  confidence: number;
  analysis: string;
}

class DatabaseCrossoverDetector {
  private universeFamilies: UniverseFamily[] | null = null;
  private cacheExpiry: number = 0;
  private readonly CACHE_DURATION = 5 * 60 * 1000; // 5 minutes

  /**
   * Fetch universe families from the database via API
   */
  private async fetchUniverseFamilies(): Promise<UniverseFamily[]> {
    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/tags/universe-families`, {
        method: 'GET',
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
        }
      });

      if (!response.ok) {
        throw new Error(`Failed to fetch universe families: ${response.status}`);
      }

      const data = await response.json();
      return data.universe_families || [];
    } catch (error) {
      console.warn('Failed to fetch universe families from database, falling back to basic detection:', error);
      
      // Fallback: Basic detection without universe families
      return [];
    }
  }

  /**
   * Get universe families with caching
   */
  private async getUniverseFamilies(): Promise<UniverseFamily[]> {
    const now = Date.now();
    
    if (!this.universeFamilies || now > this.cacheExpiry) {
      this.universeFamilies = await this.fetchUniverseFamilies();
      this.cacheExpiry = now + this.CACHE_DURATION;
    }
    
    return this.universeFamilies;
  }

  /**
   * Determine which universe family a fandom belongs to
   */
  private findUniverseForFandom(fandom: string, universeFamilies: UniverseFamily[]): string | null {
    for (const family of universeFamilies) {
      if (family.member_fandoms.some(member => 
        member.toLowerCase().includes(fandom.toLowerCase()) ||
        fandom.toLowerCase().includes(member.toLowerCase())
      )) {
        return family.name;
      }
    }
    return null;
  }

  /**
   * Analyze if fandoms represent a crossover
   */
  async detectCrossover(fandoms: string[]): Promise<CrossoverAnalysis> {
    if (!fandoms || fandoms.length <= 1) {
      return {
        isCrossover: false,
        universes: fandoms || [],
        confidence: 1.0,
        analysis: 'Single or no fandoms - not a crossover'
      };
    }

    try {
      const universeFamilies = await this.getUniverseFamilies();
      
      if (universeFamilies.length === 0) {
        // Fallback: Basic heuristic detection
        return this.basicCrossoverDetection(fandoms);
      }

      // Map fandoms to their universe families
      const universeMapping = new Map<string, string[]>();
      const unmappedFandoms: string[] = [];

      for (const fandom of fandoms) {
        const universe = this.findUniverseForFandom(fandom, universeFamilies);
        
        if (universe) {
          if (!universeMapping.has(universe)) {
            universeMapping.set(universe, []);
          }
          universeMapping.get(universe)!.push(fandom);
        } else {
          unmappedFandoms.push(fandom);
        }
      }

      // Determine crossover status
      const totalUniverses = universeMapping.size + unmappedFandoms.length;
      const isCrossover = totalUniverses > 1;
      
      const universes = [
        ...Array.from(universeMapping.keys()),
        ...unmappedFandoms.map(f => `Independent: ${f}`)
      ];

      const confidence = unmappedFandoms.length === 0 ? 1.0 : 
                        Math.max(0.5, 1.0 - (unmappedFandoms.length / fandoms.length * 0.5));

      let analysis = '';
      if (isCrossover) {
        const knownUniverses = Array.from(universeMapping.keys());
        if (knownUniverses.length > 1) {
          analysis = `Crossover detected: ${knownUniverses.join(' × ')}`;
        } else if (knownUniverses.length === 1 && unmappedFandoms.length > 0) {
          analysis = `Crossover detected: ${knownUniverses[0]} × ${unmappedFandoms.length} independent fandom(s)`;
        } else {
          analysis = `Crossover detected: ${unmappedFandoms.length} independent fandoms`;
        }
      } else {
        analysis = universeMapping.size > 0 ? 
          `Same universe: ${Array.from(universeMapping.keys())[0]}` :
          'Single independent fandom';
      }

      return {
        isCrossover,
        universes,
        confidence,
        analysis
      };

    } catch (error) {
      console.warn('Database crossover detection failed, using fallback:', error);
      return this.basicCrossoverDetection(fandoms);
    }
  }

  /**
   * Fallback basic crossover detection when database is unavailable
   */
  private basicCrossoverDetection(fandoms: string[]): CrossoverAnalysis {
    // Simple heuristic: if fandoms have very different keywords, likely crossover
    const keywords = [
      ['marvel', 'avengers', 'iron man', 'thor', 'spider'],
      ['harry potter', 'hogwarts', 'wizard'],
      ['sherlock', 'holmes', 'baker street'],
      ['star wars', 'jedi', 'sith', 'force'],
      ['batman', 'superman', 'justice league', 'dc']
    ];

    const matchedGroups = new Set<number>();
    
    for (const fandom of fandoms) {
      const lowerFandom = fandom.toLowerCase();
      for (let i = 0; i < keywords.length; i++) {
        if (keywords[i].some(keyword => lowerFandom.includes(keyword))) {
          matchedGroups.add(i);
          break;
        }
      }
    }

    const isCrossover = matchedGroups.size > 1;
    
    return {
      isCrossover,
      universes: fandoms,
      confidence: 0.7, // Lower confidence for heuristic detection
      analysis: isCrossover ? 
        'Crossover detected (heuristic analysis)' : 
        'Same universe or unrecognized fandoms (heuristic analysis)'
    };
  }

  /**
   * Check if works should be filtered as crossovers
   */
  async shouldFilterAsCrossover(fandoms: string[], userWantsCrossovers: boolean = true): Promise<boolean> {
    if (userWantsCrossovers) {
      return false; // User wants crossovers, don't filter
    }

    const analysis = await this.detectCrossover(fandoms);
    return analysis.isCrossover && analysis.confidence >= 0.8;
  }

  /**
   * Get crossover analysis for display
   */
  async analyzeCrossoverForDisplay(fandoms: string[]): Promise<string> {
    const analysis = await this.detectCrossover(fandoms);
    return analysis.analysis;
  }
}

// Export singleton instance
export const crossoverDetector = new DatabaseCrossoverDetector();

// Export types and functions for compatibility
export type { CrossoverAnalysis };
export const { detectCrossover, shouldFilterAsCrossover, analyzeCrossoverForDisplay } = crossoverDetector;