package demo

// The word banks everything is made from. All of the wording is original and
// deliberately plain: the demo archive only needs text that reads like a
// story, not stories. The fandoms are works in the public domain, and
// "Original Work".

// character is a canonical character: the tag name, and what the stories
// call them.
type character struct{ Name, Short string }

// relationship is a canonical relationship tag, the characters in it (by tag
// name) and its AO3 category.
type relationship struct {
	Name     string
	Members  []string
	Category string
}

type fandomSpec struct {
	Name          string
	Weight        int
	Characters    []character
	Relationships []relationship
	Places        []string
	Objects       []string
}

var fandomSpecs = []fandomSpec{
	{
		Name: "Sherlock Holmes - Arthur Conan Doyle", Weight: 22,
		Characters: []character{
			{"Sherlock Holmes", "Holmes"}, {"John Watson", "Watson"}, {"Mrs. Hudson", "Mrs. Hudson"},
			{"Mycroft Holmes", "Mycroft"}, {"Irene Adler", "Irene"}, {"James Moriarty", "Moriarty"},
			{"Inspector Lestrade", "Lestrade"},
		},
		Relationships: []relationship{
			{"Sherlock Holmes/John Watson", []string{"Sherlock Holmes", "John Watson"}, "M/M"},
			{"Sherlock Holmes & John Watson", []string{"Sherlock Holmes", "John Watson"}, "Gen"},
			{"Sherlock Holmes & Mycroft Holmes", []string{"Sherlock Holmes", "Mycroft Holmes"}, "Gen"},
			{"Sherlock Holmes/Irene Adler", []string{"Sherlock Holmes", "Irene Adler"}, "F/M"},
		},
		Places:  []string{"Baker Street", "the sitting room", "a hansom cab", "the Strand", "the Embankment", "a chemist's shop", "the Diogenes Club", "a foggy platform", "Regent's Park"},
		Objects: []string{"magnifying glass", "violin", "case file", "pipe", "telegram", "cipher", "mantel clock", "tin of tobacco", "chemistry set"},
	},
	{
		Name: "Pride and Prejudice - Jane Austen", Weight: 18,
		Characters: []character{
			{"Elizabeth Bennet", "Elizabeth"}, {"Fitzwilliam Darcy", "Darcy"}, {"Jane Bennet", "Jane"},
			{"Charles Bingley", "Bingley"}, {"Lydia Bennet", "Lydia"}, {"Charlotte Lucas", "Charlotte"},
			{"Mr. Collins", "Mr. Collins"}, {"George Wickham", "Wickham"},
		},
		Relationships: []relationship{
			{"Elizabeth Bennet/Fitzwilliam Darcy", []string{"Elizabeth Bennet", "Fitzwilliam Darcy"}, "F/M"},
			{"Jane Bennet/Charles Bingley", []string{"Jane Bennet", "Charles Bingley"}, "F/M"},
			{"Elizabeth Bennet & Jane Bennet", []string{"Elizabeth Bennet", "Jane Bennet"}, "Gen"},
			{"Elizabeth Bennet & Charlotte Lucas", []string{"Elizabeth Bennet", "Charlotte Lucas"}, "Gen"},
		},
		Places:  []string{"Longbourn", "Pemberley", "Netherfield", "Meryton", "the assembly rooms", "a country lane", "the drawing room", "the shrubbery", "Rosings"},
		Objects: []string{"calling card", "sampler", "pianoforte", "letter", "fan", "bonnet", "tea service", "volume of sermons", "pair of gloves"},
	},
	{
		Name: "Les Misérables - Victor Hugo", Weight: 12,
		Characters: []character{
			{"Jean Valjean", "Valjean"}, {"Javert", "Javert"}, {"Cosette", "Cosette"},
			{"Marius Pontmercy", "Marius"}, {"Éponine Thénardier", "Éponine"}, {"Enjolras", "Enjolras"},
			{"Grantaire", "Grantaire"}, {"Fantine", "Fantine"},
		},
		Relationships: []relationship{
			{"Enjolras/Grantaire", []string{"Enjolras", "Grantaire"}, "M/M"},
			{"Marius Pontmercy/Cosette", []string{"Marius Pontmercy", "Cosette"}, "F/M"},
			{"Jean Valjean & Cosette", []string{"Jean Valjean", "Cosette"}, "Gen"},
			{"Jean Valjean & Javert", []string{"Jean Valjean", "Javert"}, "Gen"},
			{"Éponine Thénardier/Marius Pontmercy", []string{"Éponine Thénardier", "Marius Pontmercy"}, "F/M"},
		},
		Places:  []string{"a Paris café", "the Rue Plumet", "the convent garden", "Montfermeil", "a cramped garret", "the riverbank", "the market square", "a cobbled alley"},
		Objects: []string{"silver candlestick", "loaf of bread", "pocket watch", "street map", "red scarf", "ledger", "folded pamphlet", "wooden doll"},
	},
	{
		Name: "Dracula - Bram Stoker", Weight: 8,
		Characters: []character{
			{"Count Dracula", "the Count"}, {"Jonathan Harker", "Jonathan"}, {"Mina Harker", "Mina"},
			{"Lucy Westenra", "Lucy"}, {"Abraham Van Helsing", "Van Helsing"}, {"John Seward", "Seward"},
			{"R. M. Renfield", "Renfield"},
		},
		Relationships: []relationship{
			{"Jonathan Harker/Mina Harker", []string{"Jonathan Harker", "Mina Harker"}, "F/M"},
			{"Lucy Westenra & Mina Harker", []string{"Lucy Westenra", "Mina Harker"}, "Gen"},
			{"Lucy Westenra/Mina Harker", []string{"Lucy Westenra", "Mina Harker"}, "F/F"},
			{"Abraham Van Helsing & John Seward", []string{"Abraham Van Helsing", "John Seward"}, "Gen"},
		},
		Places:  []string{"Whitby", "Carfax Abbey", "the asylum at Purfleet", "a mountain pass", "the harbour wall", "the churchyard", "a shuttered study", "the night train"},
		Objects: []string{"travel journal", "wooden stake", "garlic wreath", "phonograph cylinder", "brass crucifix", "diary", "oil lamp", "typewritten page"},
	},
	{
		Name: "Frankenstein - Mary Shelley", Weight: 7,
		Characters: []character{
			{"Victor Frankenstein", "Victor"}, {"The Creature", "the Creature"}, {"Elizabeth Lavenza", "Elizabeth"},
			{"Henry Clerval", "Henry"}, {"Robert Walton", "Walton"},
		},
		Relationships: []relationship{
			{"Victor Frankenstein & The Creature", []string{"Victor Frankenstein", "The Creature"}, "Gen"},
			{"Victor Frankenstein/Elizabeth Lavenza", []string{"Victor Frankenstein", "Elizabeth Lavenza"}, "F/M"},
			{"Victor Frankenstein & Henry Clerval", []string{"Victor Frankenstein", "Henry Clerval"}, "Gen"},
			{"Robert Walton & Victor Frankenstein", []string{"Robert Walton", "Victor Frankenstein"}, "Gen"},
		},
		Places:  []string{"Ingolstadt", "the lake at Geneva", "the laboratory", "the Arctic ice", "a mountain hut", "the university library", "a storm-lit road", "the ship's cabin"},
		Objects: []string{"bundle of letters", "glass retort", "lantern", "borrowed coat", "log book", "shuttered window", "bottle of ink", "iron key"},
	},
	{
		Name: "Little Women - Louisa May Alcott", Weight: 10,
		Characters: []character{
			{"Jo March", "Jo"}, {"Meg March", "Meg"}, {"Beth March", "Beth"}, {"Amy March", "Amy"},
			{"Theodore Laurence", "Laurie"}, {"Marmee March", "Marmee"}, {"Friedrich Bhaer", "Professor Bhaer"},
			{"Aunt March", "Aunt March"},
		},
		Relationships: []relationship{
			{"Jo March/Theodore Laurence", []string{"Jo March", "Theodore Laurence"}, "F/M"},
			{"Amy March/Theodore Laurence", []string{"Amy March", "Theodore Laurence"}, "F/M"},
			{"Jo March/Friedrich Bhaer", []string{"Jo March", "Friedrich Bhaer"}, "F/M"},
			{"Jo March & Beth March", []string{"Jo March", "Beth March"}, "Gen"},
			{"March Sisters", []string{"Jo March", "Meg March", "Beth March", "Amy March"}, "Gen"},
		},
		Places:  []string{"the March house", "the attic", "Concord", "the garden gate", "a sleigh ride", "the boarding house", "the neighbour's library", "the front parlour"},
		Objects: []string{"stack of manuscripts", "knitting basket", "pair of slippers", "play script", "pickled limes", "piano", "inkwell", "patched shawl"},
	},
	{
		Name: "Arthurian Mythology", Weight: 10,
		Characters: []character{
			{"King Arthur", "Arthur"}, {"Guinevere", "Guinevere"}, {"Lancelot", "Lancelot"}, {"Merlin", "Merlin"},
			{"Morgan le Fay", "Morgan"}, {"Gawain", "Gawain"}, {"Mordred", "Mordred"}, {"Percival", "Percival"},
		},
		Relationships: []relationship{
			{"King Arthur/Guinevere", []string{"King Arthur", "Guinevere"}, "F/M"},
			{"Guinevere/Lancelot", []string{"Guinevere", "Lancelot"}, "F/M"},
			{"Lancelot/King Arthur", []string{"Lancelot", "King Arthur"}, "M/M"},
			{"Merlin & King Arthur", []string{"Merlin", "King Arthur"}, "Gen"},
			{"Gawain & Percival", []string{"Gawain", "Percival"}, "Gen"},
		},
		Places:  []string{"Camelot", "the Round Table", "Avalon", "Tintagel", "a forest road", "the great hall", "the lake shore", "a ruined chapel"},
		Objects: []string{"cracked shield", "sword belt", "silver goblet", "weathered map", "carved chess piece", "banner", "harp", "bowl of apples"},
	},
	{
		Name: "Original Work", Weight: 13,
		Characters: []character{
			{"Original Characters", "Ines"}, {"Original Female Character(s)", "Wren"}, {"Original Male Character(s)", "Tobias"},
		},
		Relationships: []relationship{
			{"Original Character/Original Character", []string{"Original Characters"}, "Other"},
			{"Original Female Character(s) & Original Male Character(s)", []string{"Original Female Character(s)", "Original Male Character(s)"}, "Gen"},
		},
		Places:  []string{"the harbour town", "a corner café", "the laundromat", "the night bus", "an empty library", "the roof of the building", "a rented flat", "the allotments"},
		Objects: []string{"thermos", "spare key", "mixtape", "bicycle lamp", "cracked phone", "paperback", "umbrella", "shoebox of photographs"},
	},
}

// Tags that need no fandom.
var freeformTags = []string{
	"Fluff", "Angst", "Hurt/Comfort", "Alternate Universe - Coffee Shops & Cafés", "Slow Burn",
	"Alternate Universe - Modern Setting", "Friends to Lovers", "Enemies to Lovers", "Found Family",
	"Humor", "Crack", "Pining", "Mutual Pining", "Established Relationship", "Fix-It", "Missing Scene",
	"Canon Compliant", "Post-Canon", "Character Study", "Domestic Fluff", "Banter", "Letters",
	"First Kiss", "Alternate Universe - Canon Divergence", "Bed Sharing", "Rain", "Tea", "Snowed In",
	"Slice of Life", "Whump", "Epistolary", "Not Beta Read", "Alternate Universe - Bookstore",
	"Comfort", "Miscommunication", "One Shot", "Winter",
}

var archiveWarnings = struct {
	None, NotChosen, Violence, Death string
}{
	"No Archive Warnings Apply", "Creator Chose Not To Use Archive Warnings",
	"Graphic Depictions Of Violence", "Major Character Death",
}

var categoryNames = []string{"F/F", "F/M", "Gen", "M/M", "Multi", "Other"}

// Names for pen names and guests.
var (
	penAdjectives = []string{"Quiet", "Amber", "Hollow", "Paper", "Velvet", "Midnight", "Copper", "Wintering", "Gilded", "Lunar",
		"Salt", "Moss", "Dusk", "Ember", "Thistle", "Fable", "Marigold", "Ivory", "Cinder", "Linen", "Slate", "Wild", "Honeyed", "Faded"}
	penNouns = []string{"Quill", "Lantern", "Fox", "Harbor", "Ledger", "Kettle", "Moth", "Orchard", "Compass", "Sparrow", "Teacup",
		"Atlas", "Parlor", "Tide", "Thimble", "Cobweb", "Almanac", "Bramble", "Echo", "Meadow", "Pocketwatch", "Inkwell", "Willow", "Postbox"}
	penFirst = []string{"Odalys", "Perrin", "Thessaly", "Lucan", "Wynne", "Corvin", "Isolde", "Maren", "Tamsin", "Ansel", "Briony",
		"Caspian", "Delphine", "Evander", "Fenella", "Gideon", "Halcyon", "Ione", "Jasper", "Kestrel", "Linnea", "Merrick", "Nerys", "Orrin"}
	penLast = []string{"Marsh", "Fenwick", "Ashdown", "Pell", "Quill", "Rooke", "Sable", "Thorne", "Underhill", "Vance", "Wick",
		"Yarrow", "Ames", "Birch", "Crane", "Dunmore", "Ellery", "Foxe", "Gale", "Hollis"}
	penSuffixes = []string{"writes", "reads", "scribbles", "in_ink", "by_lamplight", "after_dark", "with_tea", "afk"}
	guestNames  = []string{"A Reader", "Tea Drinker", "Night Owl", "Bookworm", "someone passing through", "Wandering Reader", "Lurker No More",
		"Anon Friend", "Cozy Corner", "Marginalia", "Lantern Lit", "Page Turner", "Reader from Afar", "Tiny Fan", "Evening Visitor"}
)

var pseudDescriptions = []string{
	"Writes slow burns and tea-related crimes.",
	"Mostly short stories, sometimes long ones by accident.",
	"Here for the quiet scenes between the plot.",
	"Updates when the kettle boils. Comments feed me.",
	"Letters, lanterns and the occasional disaster.",
	"Writing for the joy of it. Be kind in the comments.",
	"Fond of epistolary, rainy afternoons and found family.",
	"New here, trying every tag at least once.",
	"Short, soft, and mostly finished.",
	"Collects first lines. Finishes some of them.",
}

// Words for titles, summaries and text.
var (
	adjectives = []string{"quiet", "borrowed", "last", "unexpected", "gentle", "crooked", "silver", "patient", "wandering", "improbable",
		"careful", "ordinary", "restless", "golden", "unfinished", "faithful", "winter", "curious", "stubborn", "midnight", "kind", "tangled"}
	nouns = []string{"letter", "lantern", "bargain", "garden", "promise", "umbrella", "ledger", "melody", "staircase", "supper", "winter",
		"detour", "compass", "kettle", "harvest", "secret", "doorway", "season", "errand", "memory", "morning", "journey"}
	seasons = []string{"spring", "summer", "autumn", "winter"}
	verbs   = []string{"mend", "borrow", "forget", "answer", "bake", "carry", "keep", "return", "write", "wait for", "find", "lose"}
	pasts   = []string{"studied", "straightened", "turned over", "set down", "carried", "folded", "lifted", "polished", "weighed", "traced",
		"considered", "tucked away", "rearranged", "passed across", "examined"}
	adverbs  = []string{"slowly", "carefully", "without a word", "with great patience", "almost fondly", "briskly", "absently", "quietly", "at last", "all at once"}
	weathers = []string{"Rain tapped the glass", "A thin fog lay over everything", "The afternoon light turned amber", "Snow had begun to fall",
		"The wind rattled the shutters", "It was the sort of grey morning that asks nothing of anyone", "Frost silvered the window",
		"A warm evening settled over the street", "Thunder muttered somewhere far off"}
	times    = []string{"before breakfast", "just after dusk", "long past midnight", "in the middle of the afternoon", "at first light", "as the clocks struck six", "the following Tuesday", "a week later"}
	feelings = []string{"relieved", "a little embarrassed", "unreasonably pleased", "wary", "hopeful", "tired in a good way", "unsettled", "quietly delighted", "stubbornly calm"}
	acts     = []string{"counted the steps again", "made a list and then ignored it", "put the kettle on", "wrote three drafts and kept none", "walked the long way round",
		"read the same page twice", "rehearsed an apology", "hid a smile in the collar", "tidied the desk that was already tidy", "looked for an excuse to stay"}
	events = []string{"the long storm", "a very strange week", "the grand disagreement", "the move", "the party that went wrong", "an unexpected visit"}
)

// Sentence templates. {a}, {b} and {c} are the characters of the chapter;
// every other token is picked at random each time it appears.
var narrativeSentences = []string{
	"{a} {verbed} the {obj} {adv}, as though it might have something to say.",
	"{weather}, and {a} was {feeling}.",
	"By the time {b} arrived at {place}, {a} had already {act}.",
	"\"{say}\" said {a}, and then, because it needed saying, \"{line}\"",
	"Nobody at {place} remarked on the {obj}, which was exactly how {a} preferred it.",
	"It was {time} when {b} first noticed the {adj} {noun} on the table.",
	"{a} had never been good at {noun}s, and {b} was not helping.",
	"There is a kind of silence that settles {time}, and {a} knew it well.",
	"{b} {verbed} the {obj} and handed it over without ceremony.",
	"If anyone had asked, {a} would have said the {noun} was {b}'s idea.",
	"The {adj} {noun} sat between them, unmentioned.",
	"{a} {act}, then {act}, and finally gave up pretending to be busy.",
	"At {place}, the {obj} had been waiting since {event}.",
	"{c} looked from {a} to {b} and wisely said nothing at all.",
	"A {adj} sort of warmth settled in the room, and {a} did not question it.",
	"{b} laughed, which was not what {a} had expected, and not unwelcome either.",
	"There was a {noun} to be dealt with, and it was {time}, and {a} had run out of reasons to wait.",
	"{a} stood at {place} for a long moment, one hand on the {obj}.",
	"\"{say}\" {b} said {adv}. {a} did not answer at once.",
	"Things had been simpler before {event}, but {a} could not honestly say they had been better.",
	"{b} set the {obj} on the shelf beside the {obj}, and the shelf did not complain.",
	"Somewhere beyond {place} a clock was striking, and neither of them counted.",
	"{a} was {feeling}, and said so, which surprised them both.",
	"The {adj} {noun} had been {b}'s for years, and giving it up was no small thing.",
	"They went on to {place} together, talking of nothing in particular and meaning most of it.",
	"{weather}; {a} {verbed} the {obj} once more and put it away.",
	"Of all the {noun}s in the world, {b} had chosen the most {adj} one, and {a} could only admire it.",
	"{a} had practised the speech {time}, and now every word of it had gone.",
	"{c} brought news that changed very little and mattered a great deal.",
	"The {obj} was {adj}, the {obj} was cold, and {a} was nonetheless content.",
}

var dialogueLines = []string{
	"I did not say that", "You said something very like it", "It is only a small thing", "Then let us make it a smaller one",
	"I had not thought of it that way", "Stay a little longer", "That is not what I meant, and you know it", "Well, someone had to ask",
	"We could simply try", "I will think about it", "It was always going to rain", "Do you never tire of being right?",
	"There is tea, if that helps", "Nothing is quite as bad as it looks by lamplight", "I would rather know", "Say it plainly, then",
	"You are impossible", "And yet you are still here", "It seems I owe you an apology", "Let it be a surprise",
	"Not today", "Perhaps tomorrow", "I kept it for you", "How long have you been standing there?",
	"It is quite all right", "Look at that!", "That is a fine plan, and a terrible one", "Shall we begin?",
}

var summaryTemplates = []string{
	"{a} has never been good at {noun}s, and {b} is not helping.",
	"In which {a} and {b} are snowed in at {place} with only one {obj}.",
	"The {adj} {noun} arrives {time}, and {a} finds out what {b} has been hiding.",
	"{a} keeps a list of things that cannot possibly go wrong. {b} reads it.",
	"After {event}, {a} and {b} try to go back to normal. It does not go well, and then it does.",
	"{a} receives a {adj} {noun} by post and decides, against advice, to follow it to {place}.",
	"Three small mistakes, one {obj}, and {b} at the door.",
	"What {a} meant to say and what {a} said are two different things. {b} notices.",
	"Nothing happens at {place} for eleven days. Then {a} turns up.",
	"{a} and {b} agree to share a {noun} for one week. It lasts longer.",
	"Letters between {a} and {b}, written {time} and mailed in the morning.",
	"A {adj} case of mistaken {noun}s, with {c} caught in the middle.",
	"{b} was supposed to be gone by now. {a} was not supposed to mind.",
	"Short, soft, and mostly about a {obj}.",
	"If {a} had known the {noun} was {adj}, none of this would have started.",
}

var authorNotes = []string{
	"Thank you for reading! This one wandered off in the middle and I let it.",
	"Written in one sitting and edited in several. Any rough edges are mine.",
	"Not beta read, so please be gentle with the typos.",
	"This started as a prompt and grew a plot of its own.",
	"For everyone who asked for more of this pair.",
	"A small thing to say thank you to the people who left comments last month.",
	"Set vaguely after the main events. Details bend where they had to.",
	"Chapter titles are a private joke. You are welcome to guess.",
	"Posting as I finish each chapter, so updates may be uneven.",
	"Comments make my whole week, so thank you in advance.",
}

var chapterEndNotes = []string{
	"Thank you for reading. Next chapter soon.",
	"The next part is half-written. Tell me what you hope happens.",
	"A reminder that tea solves most things in this story.",
}

var commentTemplates = []string{
	"I loved this so much. The bit with the {obj} made me smile.",
	"This was exactly what I needed today, thank you for writing it.",
	"The way you wrote {a} is perfect. I could hear every line.",
	"Slow burns are the best and this one is no exception.",
	"I read this twice already. The ending is so gentle.",
	"The little details, like the {obj}, make it feel lived in.",
	"Thank you for sharing this! Please keep writing.",
	"I did not expect to laugh at the {noun}, but here we are.",
	"This made my morning. Bookmarked to reread when I need comfort.",
	"What a lovely, quiet story. {a} and {b} feel so real together.",
	"I am so glad I found this. Please update soon!",
	"The pacing here is wonderful. Every scene earns its place.",
	"Your take on {b} is my new favorite. More, please!",
	"That last line! I will be thinking about it all week.",
	"Cozy, funny and a little bit sad. Perfect.",
	"I came for the {noun} and stayed for the dialogue.",
	"You have a real gift for small moments. Thank you.",
	"I would read an entire novel about {a} and the {obj}.",
	"Such a warm story. I felt better after reading it.",
	"The {adj} {noun} was a clever touch.",
}

var replyTemplates = []string{
	"Thank you so much, this made my day!",
	"I am so glad it worked for you. Thank you for telling me.",
	"Ha, the {obj} was a late addition, so I am happy it landed.",
	"Thank you! More is on the way.",
	"That means a lot. I was nervous about posting this one.",
	"Thanks for reading all the way through!",
	"I agree, {a} deserved a gentler ending.",
	"You made me smile. Thank you!",
	"Exactly what I was hoping to get across. Thank you.",
	"Glad you enjoyed it, and welcome to the comments!",
}

var bookmarkNotes = []string{
	"Perfect for a rainy afternoon.", "Rereading this whenever I need comfort.", "The dialogue in this is so good.",
	"Such a quiet, lovely story.", "Wish there were more like this.", "Saving for later, the summary got me.",
	"Gentle and funny. Recommended.", "One of my favorites in this fandom.", "Great pacing and a satisfying ending.",
	"For when I want something short and kind.",
}

var bookmarkTags = []string{
	"to read", "comfort", "favorites", "rec", "reread", "cozy", "slow burn", "short and sweet", "rainy day", "beautiful prose",
}

var seriesTitles = []string{
	"Quiet Hours", "The Lantern Cycle", "Small Mercies", "Letters Left Out", "Ordinary Days", "After the Storm",
	"The Kettle Chronicles", "Winter Sketches", "A Season of Errands", "Borrowed Time", "Notes from the Margins", "The Long Way Round",
}

var seriesDescriptions = []string{
	"Stories set in the same quiet corner, in no strict order.",
	"A handful of related one-shots and one longer piece.",
	"Companion pieces. Each can be read alone.",
	"A slow sequence that grew one chapter at a time.",
}
