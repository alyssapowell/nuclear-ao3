/**
 * Universe family detection for crossover filtering
 * This will eventually be replaced with proper tag wrangling from the database
 */

interface UniverseFamily {
  id: string;
  name: string;
  tags: string[];
}

// Common universe families (hardcoded for now, will move to database)
const UNIVERSE_FAMILIES: UniverseFamily[] = [
  {
    id: 'marvel',
    name: 'Marvel Universe',
    tags: [
      'Marvel',
      'Marvel Cinematic Universe',
      'Marvel (Comics)',
      'Thor (Marvel)',
      'Iron Man (Movies)', 
      'Captain America (Movies)',
      'Spider-Man (Tom Holland Movies)',
      'Avengers (Marvel Movies)',
      'X-Men (Movies)',
      'Deadpool (Movieverse)',
      'Guardians of the Galaxy (Movies)',
      'The Avengers (Marvel) - All Media Types'
    ]
  },
  {
    id: 'harrypotter',
    name: 'Harry Potter Universe',
    tags: [
      'Harry Potter - J. K. Rowling',
      'Harry Potter (Movies)',
      'Harry Potter - All Media Types',
      'Fantastic Beasts and Where to Find Them (Movies)',
      'Harry Potter: Hogwarts Mystery (Video Game)',
      'Harry Potter and the Cursed Child - Thorne & Rowling'
    ]
  },
  {
    id: 'sherlock',
    name: 'Sherlock Holmes Universe',
    tags: [
      'Sherlock Holmes & Related Fandoms',
      'Sherlock (TV)',
      'Sherlock Holmes - Arthur Conan Doyle',
      'Sherlock Holmes (2009)',
      'Elementary (TV)',
      'The Great Mouse Detective (1986)'
    ]
  },
  {
    id: 'dc',
    name: 'DC Universe',
    tags: [
      'DC Comics',
      'Batman - All Media Types',
      'Superman - All Media Types',
      'Wonder Woman - All Media Types',
      'Justice League - All Media Types',
      'The Flash (TV 2014)',
      'Arrow (TV 2012)',
      'Supergirl (TV 2015)',
      'Batman (Movies - Nolan)',
      'Man of Steel (2013)'
    ]
  },
  {
    id: 'starwars',
    name: 'Star Wars Universe',
    tags: [
      'Star Wars - All Media Types',
      'Star Wars Sequel Trilogy',
      'Star Wars Prequel Trilogy',
      'Star Wars Original Trilogy',
      'The Mandalorian (TV)',
      'Star Wars: The Clone Wars (2008) - All Media Types',
      'Rogue One: A Star Wars Story (2016)'
    ]
  },
  {
    id: 'startrek',
    name: 'Star Trek Universe',
    tags: [
      'Star Trek',
      'Star Trek: The Original Series',
      'Star Trek: The Next Generation',
      'Star Trek: Deep Space Nine',
      'Star Trek: Voyager',
      'Star Trek: Discovery',
      'Star Trek: Strange New Worlds',
      'Star Trek (2009)'
    ]
  },
  {
    id: 'anime_naruto',
    name: 'Naruto Universe',
    tags: [
      'Naruto',
      'Naruto Shippuuden',
      'Boruto: Naruto Next Generations'
    ]
  },
  {
    id: 'anime_onepiece',
    name: 'One Piece Universe',
    tags: [
      'One Piece',
      'One Piece (Anime & Manga)'
    ]
  }
];

/**
 * Get the universe family for a given fandom tag
 */
export function getUniverseFamily(fandomTag: string): string | null {
  const normalizedTag = fandomTag.toLowerCase().trim();
  
  for (const family of UNIVERSE_FAMILIES) {
    for (const tag of family.tags) {
      if (tag.toLowerCase() === normalizedTag) {
        return family.id;
      }
    }
  }
  
  // Return the tag itself if no family found (treats as unique universe)
  return normalizedTag;
}

/**
 * Determine if a list of fandoms represents a crossover
 * Returns true if fandoms span multiple universe families
 */
export function isCrossover(fandoms: string[]): boolean {
  if (!fandoms || fandoms.length <= 1) {
    return false;
  }
  
  const universeIds = fandoms
    .map(fandom => getUniverseFamily(fandom))
    .filter(Boolean);
  
  const uniqueUniverses = new Set(universeIds);
  return uniqueUniverses.size > 1;
}

/**
 * Get human-readable universe names for debugging
 */
export function getUniverseNames(fandoms: string[]): string[] {
  const universeIds = fandoms
    .map(fandom => getUniverseFamily(fandom))
    .filter((id): id is string => Boolean(id));
    
  const uniqueIds = [...new Set(universeIds)];
  
  return uniqueIds.map(id => {
    const family = UNIVERSE_FAMILIES.find(f => f.id === id);
    return family?.name || id;
  });
}

/**
 * Examples for testing:
 * 
 * isCrossover(['Marvel Cinematic Universe', 'Thor (Marvel)']) 
 * → false (same universe)
 * 
 * isCrossover(['Harry Potter - J. K. Rowling', 'Marvel Cinematic Universe']) 
 * → true (different universes)
 * 
 * isCrossover(['Sherlock (TV)', 'Sherlock Holmes - Arthur Conan Doyle'])
 * → false (same universe family)
 */