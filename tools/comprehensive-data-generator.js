#!/usr/bin/env node
/**
 * Comprehensive AO3-style Test Data Generator
 * Generates realistic fanfiction metadata for major fandoms
 */

const fs = require('fs');
const path = require('path');

// Fandom configurations with realistic tag distributions
const FANDOMS = {
  "Marvel Cinematic Universe": {
    characters: [
      "Agatha Harkness", "Rio Vidal", "Wanda Maximoff", "Vision", "Stephen Strange",
      "Tony Stark", "Steve Rogers", "Natasha Romanoff", "Clint Barton", "Bruce Banner",
      "Thor", "Loki", "Peter Parker", "Scott Lang", "Hope van Dyne", "Carol Danvers",
      "Kamala Khan", "Sam Wilson", "Bucky Barnes", "Peter Quill", "Gamora", "Rocket",
      "Groot", "Drax", "Mantis", "Nebula", "T'Challa", "Shuri", "Okoye", "Nakia",
      "Wade Wilson", "Logan", "Matt Murdock", "Frank Castle", "Jessica Jones"
    ],
    relationships: [
      "Agatha Harkness/Rio Vidal", "Wanda Maximoff/Vision", "Wanda Maximoff/Agatha Harkness",
      "Steve Rogers/Bucky Barnes", "Tony Stark/Stephen Strange", "Steve Rogers/Tony Stark",
      "Natasha Romanoff/Clint Barton", "Thor/Loki", "Peter Parker/Michelle Jones",
      "Scott Lang/Hope van Dyne", "Carol Danvers/Maria Rambeau", "Peter Quill/Gamora",
      "T'Challa/Nakia", "Sam Wilson/Bucky Barnes", "Loki/Stephen Strange",
      "Wade Wilson/Logan", "Matt Murdock/Frank Castle", "Jessica Jones/Luke Cage"
    ],
    freeform_tags: [
      "Witches", "Magic", "Angst", "Hurt/Comfort", "Slow Burn", "Enemies to Lovers",
      "Found Family", "Time Travel", "Alternate Universe - Modern Setting", "Fluff",
      "Established Relationship", "First Kiss", "Mutual Pining", "Soul Bond",
      "Protective", "PTSD", "Healing", "Domesticity", "Mission Fic", "Identity Reveal",
      "Superhero", "Powers", "Team Bonding", "Civil War", "Infinity War"
    ],
    common_aus: [
      "Coffee Shop", "College/University", "Soulmate", "Vampire", "Werewolf",
      "No Powers", "Historical", "Royalty", "Spy", "Medical", "High School"
    ]
  },
  "Supernatural": {
    characters: [
      "Dean Winchester", "Sam Winchester", "Castiel", "Gabriel", "Bobby Singer",
      "John Winchester", "Mary Winchester", "Crowley", "Lucifer", "Michael",
      "Jack Kline", "Rowena MacLeod", "Charlie Bradbury", "Jody Mills", "Donna Hanscum",
      "Ellen Harvelle", "Jo Harvelle", "Ash", "Rufus Turner", "Kevin Tran",
      "Chuck Shurley", "Amara", "Balthazar", "Anna Milton", "Ruby"
    ],
    relationships: [
      "Dean Winchester/Castiel", "Sam Winchester/Gabriel", "Dean Winchester/Sam Winchester",
      "Crowley/Aziraphale", "Jack Kline/Alex Jones", "Charlie Bradbury/Dorothy Baum",
      "Jody Mills/Donna Hanscum", "Ellen Harvelle/Bobby Singer", "John Winchester/Mary Winchester",
      "Sam Winchester/Jessica Moore", "Dean Winchester/Lisa Braeden", "Castiel/Meg Masters",
      "Gabriel/Sam Winchester", "Lucifer/Michael", "Rowena MacLeod/Crowley"
    ],
    freeform_tags: [
      "Hunter", "Angels", "Demons", "Apocalypse", "Family Business", "Angst", "Hurt/Comfort",
      "Slow Burn", "Enemies to Lovers", "Found Family", "Protective", "PTSD", "Hell",
      "Heaven", "Purgatory", "Immortality", "Soul Bond", "Grace", "First Kiss",
      "Mutual Pining", "Resurrection", "Time Travel", "Alternate Universe", "Fluff"
    ],
    common_aus: [
      "No Supernatural", "Coffee Shop", "College/University", "Soulmate", "Vampire",
      "Werewolf", "Historical", "Royalty", "Modern", "High School", "Medical"
    ]
  },
  "Harry Potter - J. K. Rowling": {
    characters: [
      "Harry Potter", "Hermione Granger", "Ron Weasley", "Draco Malfoy", "Severus Snape",
      "Sirius Black", "Remus Lupin", "James Potter", "Lily Evans Potter", "Ginny Weasley",
      "Luna Lovegood", "Neville Longbottom", "Fred Weasley", "George Weasley",
      "Minerva McGonagall", "Albus Dumbledore", "Tom Riddle", "Voldemort", "Bellatrix Lestrange",
      "Narcissa Malfoy", "Lucius Malfoy", "Regulus Black", "Barty Crouch Jr"
    ],
    relationships: [
      "Hermione Granger/Draco Malfoy", "Harry Potter/Draco Malfoy", "Sirius Black/Remus Lupin",
      "James Potter/Lily Evans Potter", "Harry Potter/Ginny Weasley", "Ron Weasley/Hermione Granger",
      "Hermione Granger/Severus Snape", "Harry Potter/Tom Riddle", "Luna Lovegood/Ginny Weasley",
      "Fred Weasley/George Weasley", "Regulus Black/James Potter", "Barty Crouch Jr/Regulus Black"
    ],
    freeform_tags: [
      "Hogwarts Eighth Year", "Time Travel", "Marauders Era", "Death Eater Draco Malfoy",
      "Redemption", "Dark Harry Potter", "Manipulative Dumbledore", "Slytherin Harry Potter",
      "Veela", "Creature Fic", "Soul Bond", "Marriage Law", "Arranged Marriage",
      "Enemies to Lovers", "Slow Burn", "Angst", "Hurt/Comfort", "Found Family"
    ],
    common_aus: [
      "Muggle", "Soulmate", "Vampire", "Werewolf", "Modern", "Royalty", "Historical"
    ]
  },
  "Doctor Who": {
    characters: [
      "The Doctor", "Rose Tyler", "Martha Jones", "Donna Noble", "Amy Pond",
      "Rory Williams", "Clara Oswald", "Bill Potts", "Ryan Sinclair", "Yasmin Khan",
      "Graham O'Brien", "River Song", "Jack Harkness", "Mickey Smith", "Sarah Jane Smith",
      "The Master", "Missy", "Jenny", "Vastra", "Strax"
    ],
    relationships: [
      "The Doctor/Rose Tyler", "The Doctor/River Song", "Amy Pond/Rory Williams",
      "The Doctor/Clara Oswald", "The Doctor/Martha Jones", "Jack Harkness/The Doctor",
      "Jenny/Vastra", "The Doctor/The Master", "Mickey Smith/Rose Tyler"
    ],
    freeform_tags: [
      "Time Travel", "Regeneration", "Hurt/Comfort", "Angst", "Protective Doctor",
      "Bad Wolf", "Telepathic Bond", "Time Lord Biology", "TARDIS", "Gallifrey",
      "Daleks", "Cybermen", "Weeping Angels", "Reunion", "Time Lock", "Paradox"
    ],
    common_aus: [
      "Human", "Soulmate", "Modern", "Historical", "No Time Travel", "Academy Era"
    ]
  },
  "Sherlock Holmes & Related Fandoms": {
    characters: [
      "Sherlock Holmes", "John Watson", "Mycroft Holmes", "Jim Moriarty",
      "Mrs. Hudson", "Molly Hooper", "Greg Lestrade", "Irene Adler", "Mary Morstan"
    ],
    relationships: [
      "Sherlock Holmes/John Watson", "Sherlock Holmes/Jim Moriarty",
      "John Watson/Mary Morstan", "Mycroft Holmes/Greg Lestrade"
    ],
    freeform_tags: [
      "Case Fic", "Hurt/Comfort", "First Kiss", "Pining", "Domestic",
      "Post-Reichenbach", "Angst", "Fluff", "Jealousy", "Protective"
    ],
    common_aus: [
      "Modern", "Vampire", "Werewolf", "Omegaverse", "Soulmate", "Historical"
    ]
  },
  "Good Omens - Neil Gaiman & Terry Pratchett": {
    characters: [
      "Aziraphale", "Anthony J. Crowley", "Gabriel", "Beelzebub", "Hastur",
      "Ligur", "Anathema Device", "Newton Pulsifer", "Adam Young", "The Them"
    ],
    relationships: [
      "Aziraphale/Anthony J. Crowley", "Anathema Device/Newton Pulsifer",
      "Gabriel/Beelzebub"
    ],
    freeform_tags: [
      "Ineffable Husbands", "Wing Kink", "Post-Apocalypse", "Domestic",
      "Hurt/Comfort", "Angst", "Fluff", "First Kiss", "Slow Burn", "Protective"
    ],
    common_aus: [
      "Human", "Modern", "Historical", "Soulmate", "Coffee Shop"
    ]
  },
  "Teen Wolf (TV)": {
    characters: [
      "Derek Hale", "Stiles Stilinski", "Scott McCall", "Lydia Martin", "Allison Argent",
      "Jackson Whittemore", "Isaac Lahey", "Erica Reyes", "Vernon Boyd", "Kira Yukimura",
      "Malia Tate", "Liam Dunbar", "Theo Raeken", "Chris Argent", "Noah Stilinski",
      "Peter Hale", "Deucalion", "Jennifer Blake", "Kate Argent"
    ],
    relationships: [
      "Derek Hale/Stiles Stilinski", "Scott McCall/Allison Argent", "Lydia Martin/Jackson Whittemore",
      "Scott McCall/Kira Yukimura", "Malia Tate/Stiles Stilinski", "Isaac Lahey/Scott McCall",
      "Derek Hale/Scott McCall", "Chris Argent/Noah Stilinski", "Peter Hale/Stiles Stilinski"
    ],
    freeform_tags: [
      "Werewolves", "Pack Dynamics", "Mates", "Alpha/Beta/Omega", "Hurt/Comfort",
      "Protective", "Angst", "Fluff", "Slow Burn", "First Kiss", "Pack Mom Stiles",
      "BAMF Stiles", "Magic Stiles", "Spark Stiles", "Family", "Found Family"
    ],
    common_aus: [
      "No Supernatural", "Coffee Shop", "College/University", "Soulmate", "Historical",
      "Royalty", "Modern", "High School", "Medical", "Space"
    ]
  },
  "My Hero Academia": {
    characters: [
      "Midoriya Izuku", "Bakugou Katsuki", "Todoroki Shouto", "Iida Tenya", "Uraraka Ochako",
      "Asui Tsuyu", "Kirishima Eijirou", "Kaminari Denki", "Sero Hanta", "Ashido Mina",
      "Yaoyorozu Momo", "Jirou Kyouka", "Tokoyami Fumikage", "Aizawa Shouta", "Yamada Hizashi",
      "All Might", "Yagi Toshinori", "Shigaraki Tomura", "Dabi", "Hawks"
    ],
    relationships: [
      "Bakugou Katsuki/Midoriya Izuku", "Todoroki Shouto/Midoriya Izuku", "Aizawa Shouta/Yamada Hizashi",
      "Kirishima Eijirou/Bakugou Katsuki", "Kaminari Denki/Shinsou Hitoshi", "Dabi/Hawks",
      "Iida Tenya/Todoroki Shouto", "Asui Tsuyu/Uraraka Ochako", "Jirou Kyouka/Yaoyorozu Momo"
    ],
    freeform_tags: [
      "Quirks", "Heroes", "Villains", "U.A. High School", "Training", "Hurt/Comfort",
      "Angst", "Fluff", "Slow Burn", "First Kiss", "Protective", "BAMF", "Vigilante",
      "Quirkless", "Pro Hero", "Teacher-Student Relationship", "Found Family"
    ],
    common_aus: [
      "No Quirks", "Coffee Shop", "College/University", "Soulmate", "Vampire",
      "Historical", "Royalty", "Modern", "Villain", "Vigilante"
    ]
  },
  "Naruto": {
    characters: [
      "Uzumaki Naruto", "Uchiha Sasuke", "Haruno Sakura", "Hatake Kakashi", "Uchiha Itachi",
      "Gaara", "Rock Lee", "Neji Hyuuga", "Tenten", "Shikamaru Nara", "Ino Yamanaka",
      "Choji Akimichi", "Kiba Inuzuka", "Shino Aburame", "Hinata Hyuuga", "Sai",
      "Yamato", "Jiraiya", "Tsunade", "Orochimaru", "Minato Namikaze", "Kushina Uzumaki"
    ],
    relationships: [
      "Uzumaki Naruto/Uchiha Sasuke", "Hatake Kakashi/Umino Iruka", "Uchiha Itachi/Uchiha Sasuke",
      "Gaara/Rock Lee", "Neji Hyuuga/Tenten", "Shikamaru Nara/Temari", "Sai/Uzumaki Naruto",
      "Haruno Sakura/Ino Yamanaka", "Minato Namikaze/Kushina Uzumaki"
    ],
    freeform_tags: [
      "Ninja", "Chakra", "Jutsu", "Hidden Villages", "Team 7", "Hurt/Comfort", "Angst",
      "Fluff", "Slow Burn", "First Kiss", "Protective", "BAMF", "Time Travel",
      "Alternate Universe", "Found Family", "Bloodline Limits", "Jinchuuriki"
    ],
    common_aus: [
      "Modern", "Coffee Shop", "College/University", "Soulmate", "Vampire",
      "Historical", "Royalty", "High School", "Medical", "Space"
    ]
  },
  "Attack on Titan": {
    characters: [
      "Eren Yeager", "Mikasa Ackerman", "Armin Arlert", "Levi Ackerman", "Erwin Smith",
      "Hange Zoë", "Jean Kirstein", "Marco Bott", "Connie Springer", "Sasha Blouse",
      "Historia Reiss", "Ymir", "Reiner Braun", "Bertolt Hoover", "Annie Leonhart",
      "Zeke Yeager", "Porco Galliard", "Pieck Finger"
    ],
    relationships: [
      "Levi Ackerman/Eren Yeager", "Erwin Smith/Levi Ackerman", "Jean Kirstein/Marco Bott",
      "Historia Reiss/Ymir", "Armin Arlert/Eren Yeager", "Mikasa Ackerman/Eren Yeager",
      "Reiner Braun/Bertolt Hoover", "Annie Leonhart/Mikasa Ackerman"
    ],
    freeform_tags: [
      "Titans", "Survey Corps", "Military", "Post-Apocalyptic", "Wall Maria", "Hurt/Comfort",
      "Angst", "Fluff", "Slow Burn", "First Kiss", "Protective", "BAMF", "Time Travel",
      "Alternate Universe", "Found Family", "War", "Survival"
    ],
    common_aus: [
      "Modern", "Coffee Shop", "College/University", "Soulmate", "Vampire",
      "Historical", "Royalty", "High School", "Medical", "No Titans"
    ]
  },
  "The Witcher (TV)": {
    characters: [
      "Geralt of Rivia", "Jaskier", "Yennefer of Vengerberg", "Ciri", "Triss Merigold",
      "Vesemir", "Lambert", "Eskel", "Cahir", "Tissaia de Vries", "Stregobor",
      "Filavandrel", "Mousesack", "Calanthe", "Eist"
    ],
    relationships: [
      "Geralt of Rivia/Jaskier", "Geralt of Rivia/Yennefer of Vengerberg", "Lambert/Keira Metz",
      "Eskel/Geralt of Rivia", "Tissaia de Vries/Yennefer of Vengerberg", "Cahir/Ciri"
    ],
    freeform_tags: [
      "Witchers", "Magic", "Destiny", "Bard", "Sorceress", "Monster Hunting",
      "Hurt/Comfort", "Angst", "Fluff", "Slow Burn", "First Kiss", "Protective",
      "Found Family", "Child Surprise", "Kaer Morhen", "Continent"
    ],
    common_aus: [
      "Modern", "Coffee Shop", "College/University", "Soulmate", "Historical",
      "Royalty", "High School", "Medical", "No Magic", "Space"
    ]
  },
  "Haikyuu!!": {
    characters: [
      "Hinata Shouyou", "Kageyama Tobio", "Tsukishima Kei", "Yamaguchi Tadashi", "Tanaka Ryuunosuke",
      "Nishinoya Yuu", "Asahi Azumane", "Sugawara Koushi", "Daichi Sawamura", "Oikawa Tooru",
      "Iwaizumi Hajime", "Kuroo Tetsurou", "Kenma Kozume", "Bokuto Koutarou", "Akaashi Keiji",
      "Ushijima Wakatoshi", "Tendou Satori", "Semi Eita", "Shirabu Kenjirou"
    ],
    relationships: [
      "Hinata Shouyou/Kageyama Tobio", "Tsukishima Kei/Yamaguchi Tadashi", "Oikawa Tooru/Iwaizumi Hajime",
      "Kuroo Tetsurou/Kenma Kozume", "Bokuto Koutarou/Akaashi Keiji", "Ushijima Wakatoshi/Tendou Satori",
      "Daichi Sawamura/Sugawara Koushi", "Asahi Azumane/Nishinoya Yuu"
    ],
    freeform_tags: [
      "Volleyball", "High School", "Team", "Sports", "Competition", "Training",
      "Hurt/Comfort", "Angst", "Fluff", "Slow Burn", "First Kiss", "Protective",
      "Found Family", "Nationals", "Spring Tournament", "Friendship"
    ],
    common_aus: [
      "Coffee Shop", "College/University", "Soulmate", "Vampire", "Historical",
      "Royalty", "Modern", "Medical", "Space", "No Volleyball"
    ]
  },
  "Jujutsu Kaisen": {
    characters: [
      "Itadori Yuuji", "Fushiguro Megumi", "Kugisaki Nobara", "Gojou Satoru", "Nanami Kento",
      "Ieiri Shoko", "Inumaki Toge", "Zen'in Maki", "Panda", "Okkotsu Yuuta", "Getou Suguru",
      "Ryoumen Sukuna", "Zen'in Naoya", "Zen'in Mai", "Miwa Kasumi", "Todo Aoi"
    ],
    relationships: [
      "Itadori Yuuji/Fushiguro Megumi", "Gojou Satoru/Getou Suguru", "Inumaki Toge/Fushiguro Megumi",
      "Zen'in Maki/Kugisaki Nobara", "Okkotsu Yuuta/Inumaki Toge", "Nanami Kento/Gojou Satoru",
      "Itadori Yuuji/Ryoumen Sukuna", "Todo Aoi/Itadori Yuuji"
    ],
    freeform_tags: [
      "Jujutsu Sorcerers", "Curses", "Cursed Energy", "Tokyo Jujutsu High", "Shibuya Incident",
      "Hurt/Comfort", "Angst", "Fluff", "Slow Burn", "First Kiss", "Protective",
      "Found Family", "Domain Expansion", "Special Grade", "Vessel"
    ],
    common_aus: [
      "No Curses", "Coffee Shop", "College/University", "Soulmate", "Vampire",
      "Historical", "Royalty", "Modern", "High School", "Medical"
    ]
  },
  "Demon Slayer": {
    characters: [
      "Kamado Tanjirou", "Kamado Nezuko", "Agatsuma Zenitsu", "Hashibira Inosuke", "Tomioka Giyuu",
      "Kocho Shinobu", "Rengoku Kyoujurou", "Uzui Tengen", "Tokitou Muichirou", "Kanroji Mitsuri",
      "Iguro Obanai", "Shinazugawa Sanemi", "Himejima Gyoumei", "Kibutsuji Muzan", "Kokushibou"
    ],
    relationships: [
      "Kamado Tanjirou/Tomioka Giyuu", "Agatsuma Zenitsu/Kamado Tanjirou", "Rengoku Kyoujurou/Tomioka Giyuu",
      "Kocho Shinobu/Tomioka Giyuu", "Kanroji Mitsuri/Iguro Obanai", "Uzui Tengen/Rengoku Kyoujurou",
      "Hashibira Inosuke/Kamado Tanjirou", "Tokitou Muichirou/Kamado Tanjirou"
    ],
    freeform_tags: [
      "Demon Slayers", "Demons", "Breathing Techniques", "Hashira", "Demon Slayer Corps",
      "Hurt/Comfort", "Angst", "Fluff", "Slow Burn", "First Kiss", "Protective",
      "Found Family", "Sun Breathing", "Water Breathing", "Final Selection"
    ],
    common_aus: [
      "Modern", "Coffee Shop", "College/University", "Soulmate", "Vampire",
      "Historical", "Royalty", "High School", "Medical", "No Demons"
    ]
  },
  "Star Wars - All Media Types": {
    characters: [
      "Obi-Wan Kenobi", "Anakin Skywalker", "Padmé Amidala", "Luke Skywalker", "Leia Organa",
      "Han Solo", "Chewbacca", "Lando Calrissian", "Rey", "Finn", "Poe Dameron", "Kylo Ren",
      "Ahsoka Tano", "Yoda", "Mace Windu", "Qui-Gon Jinn", "Darth Vader", "Emperor Palpatine",
      "Jyn Erso", "Cassian Andor", "Din Djarin", "Grogu"
    ],
    relationships: [
      "Obi-Wan Kenobi/Anakin Skywalker", "Han Solo/Leia Organa", "Finn/Poe Dameron", "Rey/Kylo Ren",
      "Luke Skywalker/Han Solo", "Padmé Amidala/Anakin Skywalker", "Jyn Erso/Cassian Andor",
      "Obi-Wan Kenobi/Qui-Gon Jinn", "Ahsoka Tano/Barriss Offee", "Din Djarin/Luke Skywalker"
    ],
    freeform_tags: [
      "Jedi", "Sith", "The Force", "Lightsabers", "Rebellion", "Empire", "Republic",
      "Hurt/Comfort", "Angst", "Fluff", "Slow Burn", "First Kiss", "Protective",
      "Found Family", "Padawan", "Master/Padawan", "Clone Wars", "Resistance"
    ],
    common_aus: [
      "Modern", "Coffee Shop", "College/University", "Soulmate", "Vampire",
      "Historical", "Royalty", "High School", "Medical", "No Force"
    ]
  },
  "The Mandalorian (TV)": {
    characters: [
      "Din Djarin", "Grogu", "Cara Dune", "Greef Karga", "Moff Gideon", "Bo-Katan Kryze",
      "Fennec Shand", "Boba Fett", "Cobb Vanth", "Luke Skywalker", "Ahsoka Tano"
    ],
    relationships: [
      "Din Djarin/Luke Skywalker", "Din Djarin/Cobb Vanth", "Cara Dune/Fennec Shand",
      "Bo-Katan Kryze/Din Djarin", "Boba Fett/Din Djarin"
    ],
    freeform_tags: [
      "Mandalorians", "Bounty Hunter", "Found Family", "The Child", "Beskar", "The Way",
      "Hurt/Comfort", "Angst", "Fluff", "Slow Burn", "First Kiss", "Protective",
      "Space Western", "Single Parent", "Clan", "Creed"
    ],
    common_aus: [
      "Modern", "Coffee Shop", "College/University", "Soulmate", "Vampire",
      "Historical", "Royalty", "High School", "Medical", "Western"
    ]
  }
};

// Rating distributions (realistic percentages)
const RATINGS = [
  { name: "General Audiences", weight: 25 },
  { name: "Teen And Up Audiences", weight: 45 },
  { name: "Mature", weight: 20 },
  { name: "Explicit", weight: 10 }
];

const STATUSES = [
  { name: "Complete", weight: 60 },
  { name: "Work in Progress", weight: 35 },
  { name: "Hiatus", weight: 5 }
];

const LANGUAGES = [
  { name: "English", weight: 85 },
  { name: "中文-普通话 國語", weight: 5 },
  { name: "Русский", weight: 3 },
  { name: "Español", weight: 2 },
  { name: "Français", weight: 2 },
  { name: "Deutsch", weight: 1 },
  { name: "Italiano", weight: 1 },
  { name: "日本語", weight: 1 }
];

// Title templates for different types of fics
const TITLE_TEMPLATES = [
  "The {adjective} {noun}",
  "{character}'s {noun}",
  "When {character} {verb}",
  "{noun} and {noun}",
  "A {adjective} {noun}",
  "The {noun} of {character}",
  "{adjective} {noun}s",
  "{character} and the {adjective} {noun}",
  "How {character} {verb}",
  "{noun} in {place}",
  "The {adjective} {adjective} {noun}",
  "{character}'s Guide to {noun}",
  "Five Times {character} {verb}",
  "The {noun} Chronicles",
  "Beyond the {noun}",
  "{adjective} Hearts",
  "Midnight {noun}",
  "Dancing with {noun}s",
  "The Secret of {noun}",
  "{adjective} Promises"
];

const ADJECTIVES = [
  "Dark", "Hidden", "Secret", "Lost", "Forgotten", "Ancient", "Mysterious", "Broken",
  "Golden", "Silver", "Crimson", "Emerald", "Midnight", "Silent", "Whispered",
  "Forbidden", "Stolen", "Endless", "Eternal", "Sacred", "Wild", "Fierce",
  "Gentle", "Tender", "Brave", "Bold", "Quiet", "Soft", "Strong", "Powerful"
];

const NOUNS = [
  "Heart", "Soul", "Mind", "Dream", "Memory", "Shadow", "Light", "Magic",
  "Power", "Truth", "Secret", "Mystery", "Promise", "Wish", "Hope", "Fear",
  "Love", "Kiss", "Touch", "Dance", "Song", "Story", "Tale", "Journey",
  "Path", "Way", "Road", "Bridge", "Door", "Key", "Lock", "Chain"
];

const VERBS = [
  "falls", "rises", "dances", "sings", "whispers", "dreams", "remembers",
  "forgets", "discovers", "finds", "loses", "saves", "protects", "fights",
  "loves", "hates", "fears", "hopes", "wishes", "believes", "trusts"
];

const PLACES = [
  "Darkness", "Light", "the Shadows", "the City", "the Forest", "the Mountains",
  "the Sea", "the Sky", "Time", "Space", "Dreams", "Memory", "the Past",
  "the Future", "Another World", "the Library", "the Garden", "the Tower"
];

// Author name generators
const AUTHOR_PREFIXES = [
  "Dark", "Shadow", "Moon", "Star", "Sun", "Fire", "Ice", "Storm", "Wind",
  "Magic", "Mystic", "Wild", "Free", "Lost", "Found", "Silver", "Golden",
  "Crimson", "Azure", "Raven", "Phoenix", "Dragon", "Wolf", "Rose", "Lily"
];

const AUTHOR_SUFFIXES = [
  "Writer", "Dreamer", "Weaver", "Keeper", "Hunter", "Walker", "Rider",
  "Singer", "Dancer", "Mage", "Witch", "Wizard", "Author", "Scribe",
  "Poet", "Bard", "Storyteller", "Heart", "Soul", "Spirit", "Wings"
];

// Summary templates
const SUMMARY_TEMPLATES = [
  "When {character} {verb}, everything changes. {character2} must {verb2} to {goal}.",
  "{character} has always been {adjective}, but when {event}, {they} {verb}.",
  "In a world where {premise}, {character} and {character2} must {verb}.",
  "{character} thought {they} knew everything about {noun}, until {character2} {verb}.",
  "Five times {character} {verb}, and the one time {they} {verb2}.",
  "After {event}, {character} finds {themselves} {adjective} and {adjective2}.",
  "{character} and {character2} are {adjective}, but when {event}, they must {verb}.",
  "A {adjective} look at how {character} and {character2} {verb}.",
  "{character} never expected to {verb}, especially not with {character2}.",
  "What happens when {character} discovers that {character2} is {adjective}?"
];

class DataGenerator {
  constructor() {
    this.works = [];
    this.workId = 1;
  }

  // Weighted random selection
  weightedRandom(items) {
    const totalWeight = items.reduce((sum, item) => sum + item.weight, 0);
    let random = Math.random() * totalWeight;
    
    for (const item of items) {
      random -= item.weight;
      if (random <= 0) return item.name;
    }
    return items[0].name;
  }

  // Random selection from array
  randomChoice(arr) {
    return arr[Math.floor(Math.random() * arr.length)];
  }

  // Random multiple selections
  randomChoices(arr, min = 1, max = 5) {
    const count = Math.floor(Math.random() * (max - min + 1)) + min;
    const shuffled = [...arr].sort(() => 0.5 - Math.random());
    return shuffled.slice(0, count);
  }

  // Generate realistic word count with more variety
  generateWordCount() {
    // More realistic AO3 distribution: 
    // 45% short (100-3k), 35% medium (3k-25k), 15% long (25k-100k), 5% epic (100k+)
    const rand = Math.random();
    if (rand < 0.45) {
      // Short fics - drabbles to short stories
      return Math.floor(Math.random() * 2900) + 100; // 100-3000
    } else if (rand < 0.80) {
      // Medium fics - most common range
      return Math.floor(Math.random() * 22000) + 3000; // 3k-25k
    } else if (rand < 0.95) {
      // Long fics - substantial stories
      return Math.floor(Math.random() * 75000) + 25000; // 25k-100k
    } else {
      // Epic fics - the big ones
      return Math.floor(Math.random() * 400000) + 100000; // 100k-500k
    }
  }

  // Generate chapter count based on word count
  generateChapterCount(wordCount) {
    const avgWordsPerChapter = 2000 + Math.random() * 3000; // 2k-5k words per chapter
    const estimatedChapters = Math.ceil(wordCount / avgWordsPerChapter);
    return Math.max(1, estimatedChapters + Math.floor(Math.random() * 3) - 1);
  }

  // Generate engagement metrics based on word count and age
  generateEngagement(wordCount, daysOld) {
    const baseHits = Math.max(100, wordCount * (0.5 + Math.random() * 2) + daysOld * (5 + Math.random() * 15));
    const hits = Math.floor(baseHits);
    
    // Kudos are typically 5-15% of hits
    const kudosRate = 0.05 + Math.random() * 0.1;
    const kudos = Math.floor(hits * kudosRate);
    
    // Comments are typically 1-5% of hits
    const commentRate = 0.01 + Math.random() * 0.04;
    const comments = Math.floor(hits * commentRate);
    
    // Bookmarks are typically 2-8% of hits
    const bookmarkRate = 0.02 + Math.random() * 0.06;
    const bookmarks = Math.floor(hits * bookmarkRate);
    
    return { hits, kudos, comments, bookmarks };
  }

  // Generate dates
  generateDates() {
    // Published in last 2 years
    const now = new Date();
    const twoYearsAgo = new Date(now.getTime() - (2 * 365 * 24 * 60 * 60 * 1000));
    const publishedDate = new Date(twoYearsAgo.getTime() + Math.random() * (now.getTime() - twoYearsAgo.getTime()));
    
    // Updated between published date and now
    const updatedDate = new Date(publishedDate.getTime() + Math.random() * (now.getTime() - publishedDate.getTime()));
    
    return {
      published: publishedDate.toISOString(),
      updated: updatedDate.toISOString()
    };
  }

  // Generate title
  generateTitle(fandomConfig) {
    const template = this.randomChoice(TITLE_TEMPLATES);
    const character = this.randomChoice(fandomConfig.characters);
    const character2 = this.randomChoice(fandomConfig.characters.filter(c => c !== character));
    
    return template
      .replace(/{adjective}/g, this.randomChoice(ADJECTIVES))
      .replace(/{noun}/g, this.randomChoice(NOUNS))
      .replace(/{verb}/g, this.randomChoice(VERBS))
      .replace(/{character}/g, character.split(' ')[0]) // Use first name
      .replace(/{place}/g, this.randomChoice(PLACES));
  }

  // Generate author name
  generateAuthor() {
    const styles = [
      () => this.randomChoice(AUTHOR_PREFIXES) + this.randomChoice(AUTHOR_SUFFIXES),
      () => this.randomChoice(AUTHOR_PREFIXES) + "_" + this.randomChoice(AUTHOR_SUFFIXES),
      () => this.randomChoice(AUTHOR_PREFIXES) + this.randomChoice(AUTHOR_SUFFIXES) + (Math.floor(Math.random() * 999) + 1),
      () => this.randomChoice(ADJECTIVES) + this.randomChoice(NOUNS),
      () => this.randomChoice(ADJECTIVES) + "_" + this.randomChoice(NOUNS) + "_" + (Math.floor(Math.random() * 99) + 1),
      () => this.randomChoice(NOUNS) + this.randomChoice(ADJECTIVES) + (Math.floor(Math.random() * 9999) + 1),
      () => "x" + this.randomChoice(ADJECTIVES) + this.randomChoice(NOUNS) + "x",
      () => this.randomChoice(NOUNS).toLowerCase() + "_" + this.randomChoice(VERBS).toLowerCase(),
      () => this.randomChoice(ADJECTIVES).toLowerCase() + this.randomChoice(NOUNS).toLowerCase() + Math.floor(Math.random() * 999),
      () => "thefandom" + this.randomChoice(["writer", "reader", "lover", "fan", "geek"]) + Math.floor(Math.random() * 999)
    ];
    
    return this.randomChoice(styles)();
  }

  // Generate summary
  generateSummary(fandomConfig) {
    const template = this.randomChoice(SUMMARY_TEMPLATES);
    const characters = this.randomChoices(fandomConfig.characters, 1, 2);
    
    return template
      .replace(/{character}/g, characters[0])
      .replace(/{character2}/g, characters[1] || characters[0])
      .replace(/{adjective}/g, this.randomChoice(ADJECTIVES).toLowerCase())
      .replace(/{adjective2}/g, this.randomChoice(ADJECTIVES).toLowerCase())
      .replace(/{noun}/g, this.randomChoice(NOUNS).toLowerCase())
      .replace(/{verb}/g, this.randomChoice(VERBS))
      .replace(/{verb2}/g, this.randomChoice(VERBS))
      .replace(/{they}/g, "they")
      .replace(/{themselves}/g, "themselves")
      .replace(/{goal}/g, "save the day")
      .replace(/{event}/g, "everything changes")
      .replace(/{premise}/g, "magic is real");
  }

  // Generate tags for category
  generateTags(items, minCount = 1, maxCount = 5) {
    const selected = this.randomChoices(items, minCount, maxCount);
    return selected.map(name => ({
      name,
      category: 'auto-detected', // Will be set by caller
      is_canonical: Math.random() > 0.2, // 80% canonical
      quality_score: 0.6 + Math.random() * 0.4 // 0.6-1.0
    }));
  }

  // Generate a single work
  generateWork(fandomName, fandomConfig) {
    const wordCount = this.generateWordCount();
    const chapterCount = this.generateChapterCount(wordCount);
    const dates = this.generateDates();
    const daysOld = Math.floor((new Date() - new Date(dates.published)) / (1000 * 60 * 60 * 24));
    const engagement = this.generateEngagement(wordCount, daysOld);
    const status = this.weightedRandom(STATUSES);
    
    // Determine max chapters based on status
    let maxChapters = chapterCount;
    if (status === "Work in Progress") {
      maxChapters = Math.random() > 0.5 ? null : chapterCount + Math.floor(Math.random() * 10) + 1;
    }

    // Generate tags
    const fandoms = [{ name: fandomName, category: 'fandom', is_canonical: true, quality_score: 1.0 }];
    const characters = this.generateTags(fandomConfig.characters, 1, 4);
    characters.forEach(tag => tag.category = 'character');
    
    const relationships = this.generateTags(fandomConfig.relationships, 0, 2);
    relationships.forEach(tag => tag.category = 'relationship');
    
    const freeformTags = this.generateTags(fandomConfig.freeform_tags, 2, 8);
    freeformTags.forEach(tag => tag.category = 'freeform');
    
    // Add AU tags occasionally
    if (Math.random() < 0.3) {
      const auTag = this.randomChoice(fandomConfig.common_aus);
      freeformTags.push({
        name: `Alternate Universe - ${auTag}`,
        category: 'freeform',
        is_canonical: true,
        quality_score: 0.9
      });
    }

    const work = {
      id: this.workId++,
      title: this.generateTitle(fandomConfig),
      author: this.generateAuthor(),
      summary: this.generateSummary(fandomConfig),
      word_count: wordCount,
      chapter_count: chapterCount,
      max_chapters: maxChapters,
      rating: this.weightedRandom(RATINGS),
      status: status,
      language: this.weightedRandom(LANGUAGES),
      published_date: dates.published,
      updated_date: dates.updated,
      fandoms: fandoms,
      characters: characters,
      relationships: relationships,
      freeform_tags: freeformTags,
      kudos_count: engagement.kudos,
      comment_count: engagement.comments,
      bookmark_count: engagement.bookmarks,
      hit_count: engagement.hits,
      tag_quality_score: 0.7 + Math.random() * 0.3,
      missing_tag_suggestions: Math.random() < 0.2 ? this.randomChoices(['Slow Burn', 'Hurt/Comfort', 'Angst', 'Fluff'], 1, 2) : []
    };

    return work;
  }

  // Generate works for all fandoms with realistic distribution
  generateAllWorks(totalWorks = 15000) {
    console.log(`🎯 Generating ${totalWorks} comprehensive test stories...`);
    
    // Create realistic distribution - bigger fandoms get more works
    const fandomDistribution = {
      "Harry Potter - J. K. Rowling": 0.18,      // 18% - 2700 works
      "Marvel Cinematic Universe": 0.12,         // 12% - 1800 works  
      "Supernatural": 0.11,                      // 11% - 1650 works
      "My Hero Academia": 0.08,                  // 8% - 1200 works
      "Teen Wolf (TV)": 0.07,                    // 7% - 1050 works
      "Naruto": 0.06,                            // 6% - 900 works
      "Star Wars - All Media Types": 0.06,      // 6% - 900 works
      "Sherlock Holmes & Related Fandoms": 0.05, // 5% - 750 works
      "Attack on Titan": 0.04,                  // 4% - 600 works
      "Haikyuu!!": 0.04,                        // 4% - 600 works
      "Doctor Who": 0.04,                       // 4% - 600 works
      "Jujutsu Kaisen": 0.03,                   // 3% - 450 works
      "The Witcher (TV)": 0.03,                 // 3% - 450 works
      "Good Omens - Neil Gaiman & Terry Pratchett": 0.03, // 3% - 450 works
      "Demon Slayer": 0.03,                     // 3% - 450 works
      "The Mandalorian (TV)": 0.03              // 3% - 450 works
    };
    
    for (const [fandomName, percentage] of Object.entries(fandomDistribution)) {
      const worksCount = Math.floor(totalWorks * percentage);
      console.log(`📚 Generating ${worksCount} works for ${fandomName}...`);
      
      const fandomConfig = FANDOMS[fandomName];
      if (!fandomConfig) {
        console.warn(`⚠️  Fandom config not found for: ${fandomName}`);
        continue;
      }
      
      for (let i = 0; i < worksCount; i++) {
        this.works.push(this.generateWork(fandomName, fandomConfig));
      }
    }

    // Add extra Agatha/Rio content since you're into that right now
    console.log('✨ Adding extra Agatha Harkness/Rio Vidal focused content...');
    const mcuConfig = FANDOMS["Marvel Cinematic Universe"];
    
    for (let i = 0; i < 100; i++) {
      const work = this.generateWork("Marvel Cinematic Universe", mcuConfig);
      
      // Ensure Agatha and Rio are main characters
      work.characters = [
        { name: "Agatha Harkness", category: 'character', is_canonical: true, quality_score: 1.0 },
        { name: "Rio Vidal", category: 'character', is_canonical: true, quality_score: 1.0 }
      ];
      
      // Ensure their relationship is tagged
      work.relationships = [
        { name: "Agatha Harkness/Rio Vidal", category: 'relationship', is_canonical: true, quality_score: 1.0 }
      ];
      
      // Add witch-specific tags
      work.freeform_tags.unshift(
        { name: "Witches", category: 'freeform', is_canonical: true, quality_score: 1.0 },
        { name: "Magic", category: 'freeform', is_canonical: true, quality_score: 1.0 }
      );
      
      // Vary some titles to be more Agatha/Rio specific
      if (Math.random() < 0.3) {
        const agathaRioTitles = [
          "Green Magic and Purple Hearts", "Death and the Witch", "Salem's Daughters",
          "The Coven of Two", "Witches' Road", "Dark Magic, Darker Love", "The Salem Seven",
          "Bound by Magic", "Purple and Green", "The Witch's Kiss", "Coven Mothers",
          "Agatha All Along", "Rio's Garden", "The Ballad of Agatha Harkness"
        ];
        work.title = this.randomChoice(agathaRioTitles);
      }
      
      this.works.push(work);
    }

    console.log(`✅ Generated ${this.works.length} total works across ${Object.keys(FANDOMS).length} fandoms`);
    console.log(`📊 Distribution check: ${Object.entries(fandomDistribution).map(([name, pct]) => 
      `${name}: ${Math.floor(totalWorks * pct)}`).join(', ')}`);
    
    return this.works;
  }

  // Save to file
  saveToFile(filename = 'comprehensive-test-data.json') {
    const outputPath = path.join(__dirname, '..', 'frontend', 'src', 'data', filename);
    
    // Ensure directory exists
    const dir = path.dirname(outputPath);
    if (!fs.existsSync(dir)) {
      fs.mkdirSync(dir, { recursive: true });
    }

    const data = {
      metadata: {
        generated_at: new Date().toISOString(),
        total_works: this.works.length,
        fandoms: Object.keys(FANDOMS),
        generator_version: "1.0.0"
      },
      works: this.works
    };

    fs.writeFileSync(outputPath, JSON.stringify(data, null, 2));
    console.log(`📁 Saved ${this.works.length} works to ${outputPath}`);
    
    return outputPath;
  }

  // Generate statistics
  generateStats() {
    const stats = {
      total_works: this.works.length,
      by_fandom: {},
      by_rating: {},
      by_status: {},
      total_words: 0,
      avg_words: 0,
      total_kudos: 0,
      relationships_with_most_works: {},
      characters_with_most_works: {}
    };

    // Calculate statistics
    this.works.forEach(work => {
      // Fandom stats
      const fandom = work.fandoms[0].name;
      stats.by_fandom[fandom] = (stats.by_fandom[fandom] || 0) + 1;
      
      // Rating stats
      stats.by_rating[work.rating] = (stats.by_rating[work.rating] || 0) + 1;
      
      // Status stats
      stats.by_status[work.status] = (stats.by_status[work.status] || 0) + 1;
      
      // Word stats
      stats.total_words += work.word_count;
      stats.total_kudos += work.kudos_count;
      
      // Character stats
      work.characters.forEach(char => {
        stats.characters_with_most_works[char.name] = (stats.characters_with_most_works[char.name] || 0) + 1;
      });
      
      // Relationship stats
      work.relationships.forEach(rel => {
        stats.relationships_with_most_works[rel.name] = (stats.relationships_with_most_works[rel.name] || 0) + 1;
      });
    });

    stats.avg_words = Math.round(stats.total_words / this.works.length);

    // Sort top characters and relationships
    stats.top_characters = Object.entries(stats.characters_with_most_works)
      .sort(([,a], [,b]) => b - a)
      .slice(0, 20);
      
    stats.top_relationships = Object.entries(stats.relationships_with_most_works)
      .sort(([,a], [,b]) => b - a)
      .slice(0, 15);

    return stats;
  }
}

// Main execution
if (require.main === module) {
  const generator = new DataGenerator();
  
  // Generate 15,000 works with realistic distribution
  console.log('🚀 Starting generation of 15,000 fanfiction works...');
  const startTime = Date.now();
  
  generator.generateAllWorks(15000);
  
  const generationTime = ((Date.now() - startTime) / 1000).toFixed(2);
  console.log(`⏱️  Generation completed in ${generationTime} seconds`);
  
  // Save to file
  const outputPath = generator.saveToFile('massive-ao3-dataset.json');
  
  // Display statistics
  const stats = generator.generateStats();
  console.log('\n📊 Generation Statistics:');
  console.log(`Total Works: ${stats.total_works.toLocaleString()}`);
  console.log(`Total Words: ${stats.total_words.toLocaleString()}`);
  console.log(`Average Words: ${stats.avg_words.toLocaleString()}`);
  console.log(`Total Kudos: ${stats.total_kudos.toLocaleString()}`);
  
  console.log('\n📚 Works by Fandom:');
  Object.entries(stats.by_fandom)
    .sort(([,a], [,b]) => b - a)
    .slice(0, 15)
    .forEach(([fandom, count]) => {
      console.log(`  ${fandom}: ${count.toLocaleString()} works`);
    });
  
  console.log('\n⭐ Rating Distribution:');
  Object.entries(stats.by_rating).forEach(([rating, count]) => {
    const percentage = ((count / stats.total_works) * 100).toFixed(1);
    console.log(`  ${rating}: ${count.toLocaleString()} works (${percentage}%)`);
  });
  
  console.log('\n📝 Status Distribution:');
  Object.entries(stats.by_status).forEach(([status, count]) => {
    const percentage = ((count / stats.total_works) * 100).toFixed(1);
    console.log(`  ${status}: ${count.toLocaleString()} works (${percentage}%)`);
  });
  
  console.log('\n💕 Top Relationships:');
  stats.top_relationships.slice(0, 15).forEach(([rel, count]) => {
    console.log(`  ${rel}: ${count} works`);
  });
  
  console.log('\n👤 Top Characters:');
  stats.top_characters.slice(0, 20).forEach(([char, count]) => {
    console.log(`  ${char}: ${count} works`);
  });
  
  // Word count analysis
  const wordCounts = generator.works.map(w => w.word_count).sort((a, b) => a - b);
  const median = wordCounts[Math.floor(wordCounts.length / 2)];
  const p90 = wordCounts[Math.floor(wordCounts.length * 0.9)];
  const p99 = wordCounts[Math.floor(wordCounts.length * 0.99)];
  
  console.log('\n📖 Word Count Analysis:');
  console.log(`  Shortest: ${Math.min(...wordCounts).toLocaleString()} words`);
  console.log(`  Median: ${median.toLocaleString()} words`);
  console.log(`  Average: ${stats.avg_words.toLocaleString()} words`);
  console.log(`  90th percentile: ${p90.toLocaleString()} words`);
  console.log(`  99th percentile: ${p99.toLocaleString()} words`);
  console.log(`  Longest: ${Math.max(...wordCounts).toLocaleString()} words`);
  
  console.log(`\n🎉 Complete! Dataset saved to: ${outputPath}`);
  console.log(`📁 File size: ~${(JSON.stringify(generator.works).length / 1024 / 1024).toFixed(1)} MB`);
}

module.exports = DataGenerator;