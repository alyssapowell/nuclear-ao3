package demo

import (
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"nuclear-ao3/internal/html"
)

// The plan is the whole demo archive, made in memory from a fixed random
// seed before anything is written. The same seed and the same date always
// give the same archive (apart from nothing: ids come from the seed too).

const (
	workCount    = 150
	draftCount   = 5
	anonCount    = 6 // anonymous works
	anonCollWork = 5 // works in the anonymous collection
	unrevealed   = 4 // works in the unrevealed collection
	coCreated    = 6
	restricted   = 18
	usersOnly    = 4
	commentCount = 600
	bookmarkGoal = 200
	seriesGoal   = 10
)

type account struct {
	n         int // 1..AccountCount
	id        uuid.UUID
	email     string
	joined    time.Time
	lastLogin time.Time
	pseuds    []*pseud // the default pseud first
	writes    bool     // false for accounts that only read
}

type pseud struct {
	id      uuid.UUID
	acct    *account
	name    string
	desc    string
	created time.Time
}

type tagRec struct {
	id      uuid.UUID
	name    string
	kind    string
	created time.Time
}

type chapter struct {
	id                 uuid.UUID
	number             int
	title, summary     string
	notes, endNotes    string
	content            string
	words              int
	draft              bool
	published, created time.Time
}

type work struct {
	idx      int
	id       uuid.UUID
	owner    *account
	creators []*pseud // the first is the owner's
	title    string
	summary  string
	notes    string
	rating   string

	warnings, categories                   []string
	fandoms, characters, rels, freeforms   []string
	maxChapters                            *int
	complete, draft                        bool
	restricted, anon, inAnon, inUnrevealed bool
	commentPolicy                          string
	commentsOff                            bool

	chapters  []*chapter
	published time.Time // first posted
	updated   time.Time
	pop       float64 // how well liked it is, around 1
	hits      int
	spec      *fandomSpec
}

// hidesCreator is what the app calls a work that does not show its creator.
func (w *work) hidesCreator() bool { return w.anon || w.inAnon || w.inUnrevealed }

func (w *work) wordCount() int {
	n := 0
	for _, c := range w.chapters {
		n += c.words
	}
	return n
}

type comment struct {
	id         uuid.UUID
	work       *work
	chapter    *chapter
	parent     *comment
	acct       *account // nil for a guest
	pseud      *pseud
	guestName  string
	guestEmail string
	ip         string
	content    string
	created    time.Time
	edited     *time.Time
	level      int
}

type kudo struct {
	id      uuid.UUID
	work    *work
	acct    *account // nil for a guest
	guest   string
	ip      string
	created time.Time
}

type bookmark struct {
	id      uuid.UUID
	work    *work
	acct    *account
	pseud   *pseud
	notes   string
	tags    []string
	private bool
	rec     bool
	created time.Time
}

type series struct {
	id               uuid.UUID
	owner            *account
	title, desc      string
	notes            string
	complete         bool
	works            []*work
	created, updated time.Time
}

type collection struct {
	id          uuid.UUID
	name, title string
	desc        string
	owner       *account
	pseud       *pseud
	open        bool
	moderated   bool
	anonymous   bool
	works       []*work
	created     time.Time
}

type subscription struct {
	id      uuid.UUID
	acct    *account
	kind    string // author, work, series
	target  uuid.UUID
	name    string
	events  []string
	freq    string
	created time.Time
}

type block struct {
	id               uuid.UUID
	blocker, blocked *account
	pseud            *pseud // the blocked account's pseud
	created          time.Time
}

type mute struct {
	id      uuid.UUID
	muter   *account
	pseud   *pseud
	created time.Time
}

type plan struct {
	r   *rand.Rand
	now time.Time

	accounts []*account
	pseuds   []*pseud
	tags     []*tagRec
	tagByKey map[string]*tagRec
	works    []*work
	comments []*comment
	kudos    []*kudo
	marks    []*bookmark
	series   []*series
	colls    []*collection
	subs     []*subscription
	blocks   []*block
	mutes    []*mute

	// pairs that must not interact: a block or a mute between two accounts
	apart map[[2]int]bool
}

func newPlan(now time.Time) *plan {
	p := &plan{
		r:        rand.New(rand.NewSource(Seed)),
		now:      now,
		tagByKey: map[string]*tagRec{},
		apart:    map[[2]int]bool{},
	}
	p.makeAccounts()
	p.makeBlocksAndMutes()
	p.makeWorks()
	p.makeCollections()
	p.makeKudos()
	p.makeComments()
	p.makeBookmarks()
	p.makeSeries()
	p.makeSubscriptions()
	return p
}

// --- random helpers ---

func (p *plan) id() uuid.UUID {
	u, err := uuid.NewRandomFromReader(p.r)
	if err != nil { // a math/rand source never fails
		panic(err)
	}
	return u
}

func (p *plan) chance(f float64) bool  { return p.r.Float64() < f }
func (p *plan) between(lo, hi int) int { return lo + p.r.Intn(hi-lo+1) }

func pick[T any](p *plan, list []T) T { return list[p.r.Intn(len(list))] }

func (p *plan) when(lo, hi time.Time) time.Time {
	if !hi.After(lo) {
		return lo
	}
	return lo.Add(time.Duration(p.r.Int63n(int64(hi.Sub(lo)))))
}

func later(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func (p *plan) capNow(t time.Time) time.Time {
	if t.After(p.now.Add(-time.Hour)) {
		return p.now.Add(-time.Hour)
	}
	return t
}

func (p *plan) ip() string {
	// Addresses reserved for documentation (RFC 5737): never a real visitor.
	return fmt.Sprintf("%s.%d", pick(p, []string{"192.0.2", "198.51.100", "203.0.113"}), p.between(1, 254))
}

// --- accounts ---

func (p *plan) makeAccounts() {
	// How many pseuds the accounts with more than one have.
	several := map[int]int{2: 2, 4: 3, 7: 2, 9: 2, 12: 3, 15: 2, 18: 2, 23: 3}
	// No two pen names share a word, so that no pseud's name hints at
	// another's (a pseud must never look like another pseud of the same
	// person).
	used := map[string]bool{}
	name := func() string {
		for tries := 0; ; tries++ {
			var parts []string
			var s string
			switch p.r.Intn(5) {
			case 0, 1:
				parts = []string{pick(p, penAdjectives), pick(p, penNouns)}
				s = parts[0] + parts[1]
			case 2:
				parts = []string{pick(p, penFirst), pick(p, penLast)}
				s = parts[0] + " " + parts[1]
			case 3:
				parts = []string{pick(p, penAdjectives), pick(p, penNouns)}
				s = strings.ToLower(parts[0]) + " " + strings.ToLower(parts[1])
			default:
				parts = []string{pick(p, penFirst), pick(p, penSuffixes)}
				s = strings.ToLower(parts[0]) + "_" + parts[1]
			}
			fresh := true
			for _, part := range parts {
				fresh = fresh && !used[strings.ToLower(part)]
			}
			if fresh || tries > 1000 {
				for _, part := range parts {
					used[strings.ToLower(part)] = true
				}
				return s
			}
		}
	}
	for n := 1; n <= AccountCount; n++ {
		a := &account{n: n, id: p.id(), email: AccountEmail(n), writes: n <= 20}
		a.joined = p.now.Add(-time.Duration(735-p.r.Intn(300)) * 24 * time.Hour)
		a.lastLogin = p.when(p.now.Add(-30*24*time.Hour), p.now)
		count := 1
		if k, ok := several[n]; ok {
			count = k
		}
		for i := 0; i < count; i++ {
			ps := &pseud{id: p.id(), acct: a, name: name(), created: a.joined}
			if i > 0 {
				ps.created = p.when(a.joined, p.now.Add(-30*24*time.Hour))
			}
			if p.chance(0.5) {
				ps.desc = html.Clean("<p>" + pick(p, pseudDescriptions) + "</p>")
			}
			a.pseuds = append(a.pseuds, ps)
			p.pseuds = append(p.pseuds, ps)
		}
		p.accounts = append(p.accounts, a)
	}
}

func (p *plan) isApart(a, b *account) bool {
	if a == nil || b == nil {
		return false
	}
	return p.apart[[2]int{a.n, b.n}] || p.apart[[2]int{b.n, a.n}]
}

func (p *plan) makeBlocksAndMutes() {
	acct := func(n int) *account { return p.accounts[n-1] }
	for _, pair := range [][2]int{{6, 11}, {14, 3}} {
		b := &block{id: p.id(), blocker: acct(pair[0]), blocked: acct(pair[1]), created: p.when(p.now.Add(-300*24*time.Hour), p.now.Add(-20*24*time.Hour))}
		b.pseud = b.blocked.pseuds[0]
		p.blocks = append(p.blocks, b)
		p.apart[pair] = true
	}
	for _, pair := range [][2]int{{9, 5}, {18, 2}} {
		m := &mute{id: p.id(), muter: acct(pair[0]), created: p.when(p.now.Add(-300*24*time.Hour), p.now.Add(-20*24*time.Hour))}
		m.pseud = acct(pair[1]).pseuds[0]
		p.mutes = append(p.mutes, m)
		p.apart[pair] = true
	}
}

// --- works ---

// chapterCounts is how many chapters each work has: most are one-shots, a
// few run long. It adds up to about 400 for 150 works.
func (p *plan) chapterCounts() []int {
	var counts []int
	add := func(n, lo, hi int) {
		for i := 0; i < n; i++ {
			counts = append(counts, p.between(lo, hi))
		}
	}
	add(95, 1, 1)
	add(32, 2, 4)
	add(14, 5, 8)
	add(7, 9, 13)
	add(2, 15, 20)
	p.r.Shuffle(len(counts), func(i, j int) { counts[i], counts[j] = counts[j], counts[i] })
	return counts
}

func (p *plan) tag(name, kind string, first time.Time) {
	// Tag names are unique across kinds, as in the database.
	t := p.tagByKey[name]
	if t == nil {
		t = &tagRec{id: p.id(), name: name, kind: kind, created: first}
		p.tagByKey[name] = t
		p.tags = append(p.tags, t)
	} else if t.kind != kind {
		panic("demo: tag " + name + " is both " + t.kind + " and " + kind)
	}
	if first.Before(t.created) {
		t.created = first
	}
}

func (p *plan) pickFandom() *fandomSpec {
	total := 0
	for _, f := range fandomSpecs {
		total += f.Weight
	}
	n := p.r.Intn(total)
	for i := range fandomSpecs {
		if n < fandomSpecs[i].Weight {
			return &fandomSpecs[i]
		}
		n -= fandomSpecs[i].Weight
	}
	return &fandomSpecs[0]
}

func (p *plan) makeWorks() {
	// Authors write at very different rates: a few write a lot.
	var authors []*account
	var weights []float64
	rank := p.r.Perm(20)
	for i, a := range p.accounts {
		if a.writes {
			authors = append(authors, a)
			weights = append(weights, 1/math.Pow(float64(rank[i]+1), 0.8))
		}
	}
	pickAuthor := func() *account {
		sum := 0.0
		for _, w := range weights {
			sum += w
		}
		x := p.r.Float64() * sum
		for i, w := range weights {
			if x < w {
				return authors[i]
			}
			x -= w
		}
		return authors[0]
	}

	// Which works are special, none of them more than one kind.
	perm := p.r.Perm(workCount)
	take := func(n int) map[int]bool {
		m := map[int]bool{}
		for _, i := range perm[:n] {
			m[i] = true
		}
		perm = perm[n:]
		return m
	}
	drafts, anon, anonColl, unrev := take(draftCount), take(anonCount), take(anonCollWork), take(unrevealed)
	coed, limited, usersOnlyW := take(coCreated), take(restricted), take(usersOnly)
	noComments := take(1)

	counts := p.chapterCounts()
	for i := 0; i < workCount; i++ {
		w := &work{idx: i, id: p.id(), owner: pickAuthor(), commentPolicy: "open", pop: math.Exp(p.r.NormFloat64() * 0.9)}
		w.draft = drafts[i]
		w.anon, w.inAnon, w.inUnrevealed = anon[i], anonColl[i], unrev[i]
		w.restricted = limited[i]
		if usersOnlyW[i] {
			w.commentPolicy = "users_only"
		}
		if noComments[i] {
			w.commentPolicy, w.commentsOff = "disabled", true
		}
		pr := w.owner.pseuds[0]
		if len(w.owner.pseuds) > 1 && p.chance(0.4) {
			pr = pick(p, w.owner.pseuds)
		}
		w.creators = []*pseud{pr}
		if coed[i] {
			for {
				other := pickAuthor()
				if other != w.owner && !p.isApart(other, w.owner) {
					w.creators = append(w.creators, other.pseuds[0])
					break
				}
			}
		}
		p.describe(w)
		p.writeChapters(w, counts[i])
		p.dateWork(w)
		for _, c := range w.chapters {
			c.created = c.published
			if w.draft {
				c.created = w.published
			}
		}
		for _, f := range w.fandoms {
			p.tag(f, "fandom", w.published)
		}
		for _, c := range w.characters {
			p.tag(c, "character", w.published)
		}
		for _, r := range w.rels {
			p.tag(r, "relationship", w.published)
		}
		for _, f := range w.freeforms {
			p.tag(f, "freeform", w.published)
		}
		for _, x := range w.warnings {
			p.tag(x, "warning", w.published)
		}
		for _, x := range w.categories {
			p.tag(x, "category", w.published)
		}
		p.works = append(p.works, w)
	}
}

// describe picks the work's rating, tags and wording.
func (p *plan) describe(w *work) {
	spec := p.pickFandom()
	w.spec = spec
	w.fandoms = []string{spec.Name}
	specs := []*fandomSpec{spec}
	if p.chance(0.08) {
		if other := p.pickFandom(); other != spec {
			w.fandoms = append(w.fandoms, other.Name)
			specs = append(specs, other)
		}
	}

	// Relationships bring their characters along, as they do on AO3.
	chars := map[string]bool{}
	var charOrder []string
	addChar := func(c string) {
		if !chars[c] {
			chars[c] = true
			charOrder = append(charOrder, c)
		}
	}
	cats := map[string]bool{}
	for _, s := range specs {
		if p.chance(0.65) {
			for k := p.between(1, 2); k > 0; k-- {
				rel := pick(p, s.Relationships)
				if !contains(w.rels, rel.Name) {
					w.rels = append(w.rels, rel.Name)
					cats[rel.Category] = true
					for _, m := range rel.Members {
						addChar(m)
					}
				}
			}
		}
		for k := p.between(1, 3); k > 0; k-- {
			addChar(pick(p, s.Characters).Name)
		}
	}
	w.characters = charOrder
	if len(w.rels) == 0 && p.chance(0.7) {
		cats["Gen"] = true
	}
	for _, c := range categoryNames {
		if cats[c] {
			w.categories = append(w.categories, c)
		}
	}
	if len(w.categories) >= 3 {
		w.categories = []string{"Multi"}
	}

	for k := p.between(1, 5); k > 0; k-- {
		if f := pick(p, freeformTags); !contains(w.freeforms, f) {
			w.freeforms = append(w.freeforms, f)
		}
	}

	switch x := p.r.Float64(); {
	case x < 0.45:
		w.rating = "general"
	case x < 0.80:
		w.rating = "teen"
	case x < 0.95:
		w.rating = "mature"
	default:
		w.rating = "explicit"
	}
	switch x := p.r.Float64(); {
	case x < 0.55:
		w.warnings = []string{archiveWarnings.None}
	case x < 0.80:
		w.warnings = []string{archiveWarnings.NotChosen}
	case x < 0.92:
		w.warnings = []string{archiveWarnings.Violence}
	case x < 0.97:
		w.warnings = []string{archiveWarnings.Death}
	default:
		w.warnings = []string{archiveWarnings.Violence, archiveWarnings.Death}
	}

	x := p.textCtx(w)
	w.title = p.title(x)
	w.summary = html.Clean("<p>" + p.sentence(pick(p, summaryTemplates), x) + "</p>")
	if p.chance(0.35) {
		w.notes = html.Clean("<p>" + pick(p, authorNotes) + "</p>")
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// textCtx is the cast and the places of one work, for filling templates.
type textCtx struct {
	a, b, c string
	places  []string
	objects []string
}

func (p *plan) textCtx(w *work) *textCtx {
	x := &textCtx{}
	var names []string
	for _, spec := range fandomSpecs {
		if !contains(w.fandoms, spec.Name) {
			continue
		}
		x.places = append(x.places, spec.Places...)
		x.objects = append(x.objects, spec.Objects...)
		for _, ch := range spec.Characters {
			if contains(w.characters, ch.Name) {
				names = append(names, ch.Short)
			}
		}
	}
	for len(names) < 3 {
		names = append(names, pick(p, []string{"Ines", "Wren", "Tobias", "Marlow", "Odette", "Pip"}))
	}
	p.r.Shuffle(len(names), func(i, j int) { names[i], names[j] = names[j], names[i] })
	x.a, x.b, x.c = names[0], names[1], names[2]
	if x.a == x.b {
		x.b = "Marlow"
	}
	return x
}

func (p *plan) token(key string, x *textCtx) string {
	switch key {
	case "a":
		return x.a
	case "b":
		return x.b
	case "c":
		return x.c
	case "place":
		return pick(p, x.places)
	case "obj":
		return pick(p, x.objects)
	case "adj":
		return pick(p, adjectives)
	case "noun":
		return pick(p, nouns)
	case "verbed":
		return pick(p, pasts)
	case "adv":
		return pick(p, adverbs)
	case "weather":
		return pick(p, weathers)
	case "time":
		return pick(p, times)
	case "feeling":
		return pick(p, feelings)
	case "act":
		return pick(p, acts)
	case "event":
		return pick(p, events)
	case "line", "say":
		// A line of speech, ending as speech does: a full stop (or a comma
		// when a "said" follows), unless it is a question or an exclamation.
		l := p.sentence(pick(p, dialogueLines), x)
		switch {
		case strings.HasSuffix(l, "?") || strings.HasSuffix(l, "!"):
			return l
		case key == "say":
			return l + ","
		}
		return l + "."
	}
	return key
}

// anBeforeVowel finds "a ordinary" and the like, which word lists make.
var anBeforeVowel = regexp.MustCompile(`\b([Aa]) ([AEIOUaeiou])`)

// sentence fills a template and starts it with a capital letter.
func (p *plan) sentence(t string, x *textCtx) string {
	var sb strings.Builder
	for i := 0; i < len(t); {
		if t[i] != '{' {
			sb.WriteByte(t[i])
			i++
			continue
		}
		j := strings.IndexByte(t[i:], '}')
		sb.WriteString(p.token(t[i+1:i+j], x))
		i += j + 1
	}
	s := anBeforeVowel.ReplaceAllString(sb.String(), "${1}n ${2}")
	for i, r := range s { // the first letter, which may follow a quote
		if r >= 'a' && r <= 'z' {
			return s[:i] + strings.ToUpper(string(r)) + s[i+1:]
		}
		if r >= 'A' && r <= 'Z' || r > 127 {
			break
		}
	}
	return s
}

var titleTemplates = []string{
	"The {adj} {noun}", "A {adj} {noun} at {place}", "What {a} Knew About {noun}s", "Letters to {place}", "{adj} Hours",
	"The Last {noun} of {season}", "Notes on a {adj} {noun}", "Where the {noun} Sleeps", "{place}, Slowly", "Nine Lanterns",
	"The {noun} Problem", "Tea for {a} and {b}", "Small {noun}, Large {noun}", "Before the {noun}", "After the {noun}",
	"How to {verb} a {noun}", "Something {adj} This Way", "The Quiet Between {a} and {b}", "An Agreement About a {noun}",
	"{a}, Interrupted", "A {season} of {noun}s", "The {adj} Arrangement", "Dear {a}", "Not a {noun}, Exactly",
}

var smallWords = map[string]bool{"a": true, "an": true, "the": true, "of": true, "and": true, "in": true, "at": true, "to": true,
	"for": true, "on": true, "by": true, "with": true, "from": true, "or": true}

// titleCase capitalises the words of a title, leaving the small ones alone.
func titleCase(s string) string {
	words := strings.Split(s, " ")
	for i, w := range words {
		if i > 0 && smallWords[strings.ToLower(w)] {
			words[i] = strings.ToLower(w)
			continue
		}
		if r, n := utf8.DecodeRuneInString(w); n > 0 {
			words[i] = string(unicode.ToUpper(r)) + w[n:]
		}
	}
	return strings.Join(words, " ")
}

func (p *plan) title(x *textCtx) string {
	t := pick(p, titleTemplates)
	t = strings.ReplaceAll(t, "{season}", pick(p, seasons))
	t = strings.ReplaceAll(t, "{verb}", pick(p, verbs))
	if strings.Contains(t, "{place}") && len(x.places) == 0 {
		t = "The {adj} {noun}"
	}
	s := titleCase(p.sentence(t, x))
	if len([]rune(s)) > 200 {
		s = string([]rune(s)[:200])
	}
	return s
}

// writeChapters makes the chapters and says whether the work is finished.
func (p *plan) writeChapters(w *work, n int) {
	x := p.textCtx(w)
	switch {
	case w.draft:
	case n == 1:
		one := 1
		w.maxChapters, w.complete = &one, true
	case p.chance(0.75):
		w.maxChapters, w.complete = &n, true
	case p.chance(0.5):
		planned := n + p.between(1, 6)
		w.maxChapters = &planned
	}
	for i := 1; i <= n; i++ {
		target := p.between(300, 1800)
		if n == 1 {
			target = p.between(600, 3500)
		}
		c := &chapter{id: p.id(), number: i, draft: w.draft}
		if n > 1 || p.chance(0.3) {
			c.title = titleCase(p.sentence(pick(p, []string{"{adj} {noun}", "The {noun} at {place}", "{time}", "A {adj} {noun}", "{a} and the {noun}"}), x))
		}
		if p.chance(0.15) {
			c.notes = html.Clean("<p>" + pick(p, authorNotes) + "</p>")
		}
		if n > 1 && i == n && p.chance(0.3) {
			c.endNotes = html.Clean("<p>" + pick(p, chapterEndNotes) + "</p>")
		}
		c.content, c.words = p.prose(x, target)
		w.chapters = append(w.chapters, c)
	}
}

// prose writes about target words of story, as the HTML the app stores, and
// counts its words the way the app does (strings.Fields on the stored text).
func (p *plan) prose(x *textCtx, target int) (string, int) {
	var paras []string
	words := 0
	for words < target {
		var para string
		switch r := p.r.Float64(); {
		case r < 0.18:
			para = "<p>" + p.sentence("\"{say}\" said {a}.", x) + "</p>\n<p>" + p.sentence("\"{line}\"", x) + "</p>\n<p>" + p.sentence("\"{say}\" {b} answered.", x) + "</p>"
		case r < 0.22:
			para = "<hr>"
		default:
			var sb strings.Builder
			for k := p.between(3, 6); k > 0; k-- {
				if sb.Len() > 0 {
					sb.WriteByte(' ')
				}
				sb.WriteString(p.sentence(pick(p, narrativeSentences), x))
			}
			para = "<p>" + sb.String() + "</p>"
		}
		para = html.Clean(para)
		words += len(strings.Fields(para))
		paras = append(paras, para)
	}
	return strings.Join(paras, "\n"), words
}

// dateWork spreads the work over the last two years.
func (p *plan) dateWork(w *work) {
	lo := p.now.Add(-730 * 24 * time.Hour)
	for _, c := range w.creators {
		lo = later(lo, c.acct.joined.Add(24*time.Hour))
	}
	hi := p.now.Add(-24 * time.Hour)
	n := len(w.chapters)
	if w.draft {
		w.published = p.when(later(lo, p.now.Add(-45*24*time.Hour)), hi)
		w.updated = w.published
		return
	}
	// Chapters come a few days to a few weeks apart, squeezed to fit before
	// today when the work started late.
	gap := time.Duration(p.between(3, 40)) * 24 * time.Hour
	span := time.Duration(n-1) * gap
	if room := hi.Sub(lo) * 8 / 10; span > room {
		span = room
	}
	w.published = p.when(lo, hi.Add(-span))
	at := w.published
	for i, c := range w.chapters {
		if i > 0 {
			at = at.Add(span / time.Duration(n-1))
		}
		c.published = p.capNow(at)
	}
	w.updated = w.chapters[n-1].published
	if n == 1 && p.chance(0.2) { // edited later
		w.updated = p.capNow(p.when(w.published, p.now))
	}

	ageDays := p.now.Sub(w.published).Hours() / 24
	age := 0.3 + math.Min(ageDays, 400)/400*1.2
	w.hits = int((60 + float64(p.r.Intn(500))) * w.pop * age * (1 + float64(n)/8))
}

// --- people who read ---

// readers are the accounts that may interact with a work: not its creators,
// not anybody blocked or muted to or by them.
func (p *plan) mayReach(a *account, w *work) bool {
	for _, c := range w.creators {
		if c.acct == a || p.isApart(c.acct, a) {
			return false
		}
	}
	return true
}

func (p *plan) isCreator(a *account, w *work) bool {
	for _, c := range w.creators {
		if c.acct == a {
			return true
		}
	}
	return false
}

// visible works: posted, so readers can find them.
func (p *plan) visible() []*work {
	var out []*work
	for _, w := range p.works {
		if !w.draft {
			out = append(out, w)
		}
	}
	return out
}

func (p *plan) weighted(works []*work) func() *work {
	cum := make([]float64, len(works))
	sum := 0.0
	for i, w := range works {
		sum += w.pop
		cum[i] = sum
	}
	return func() *work {
		x := p.r.Float64() * sum
		return works[sort.SearchFloat64s(cum, x)]
	}
}

func (p *plan) makeKudos() {
	for _, w := range p.visible() {
		ageDays := p.now.Sub(w.published).Hours() / 24
		age := 0.3 + math.Min(ageDays, 400)/400*1.2
		members := int(math.Round(w.pop * age * 7))
		if members > AccountCount-2 {
			members = AccountCount - 2
		}
		given := map[int]bool{}
		for tries := 0; len(given) < members && tries < 100; tries++ {
			a := pick(p, p.accounts)
			if given[a.n] || !p.mayReach(a, w) {
				continue
			}
			given[a.n] = true
			p.kudos = append(p.kudos, &kudo{id: p.id(), work: w, acct: a, ip: p.ip(),
				created: p.after(later(w.published, a.joined), 2)})
		}
		if w.restricted {
			continue // guests can't see it
		}
		for g := int(math.Round(w.pop * age * 4)); g > 0; g-- {
			p.kudos = append(p.kudos, &kudo{id: p.id(), work: w, ip: p.ip(),
				guest:   fmt.Sprintf("demo-guest-%x", p.r.Int63()),
				created: p.after(w.published, 2)})
		}
	}
	sort.SliceStable(p.kudos, func(i, j int) bool { return p.kudos[i].work.idx < p.kudos[j].work.idx })
}

// after is a time between start and now, leaning towards the start: most
// readers come soon after a work is posted.
func (p *plan) after(start time.Time, lean float64) time.Time {
	u := math.Pow(p.r.Float64(), lean)
	return p.capNow(start.Add(time.Duration(u * float64(p.now.Sub(start)))))
}

func (p *plan) makeComments() {
	works := p.visible()
	var open []*work
	for _, w := range works {
		if !w.commentsOff {
			open = append(open, w)
		}
	}
	draw := p.weighted(open)
	byWork := map[*work]int{}
	for i := 0; i < commentCount; i++ {
		byWork[draw()]++
	}
	for _, w := range open {
		n := byWork[w]
		if n == 0 {
			continue
		}
		times := make([]time.Time, n)
		for i := range times {
			times[i] = p.after(w.published, 1.6)
		}
		sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })
		var made []*comment
		x := p.textCtx(w)
		for _, at := range times {
			var parent *comment
			if len(made) > 0 && p.chance(0.4) {
				for tries := 0; tries < 10; tries++ {
					if c := pick(p, made); c.level < 3 && c.created.Before(at) {
						parent = c
						break
					}
				}
			}
			c := &comment{id: p.id(), work: w, parent: parent, ip: p.ip(), created: at}
			if !p.whoComments(c, w) {
				continue
			}
			if parent != nil {
				c.level = parent.level + 1
				c.created = p.capNow(later(c.created, parent.created.Add(time.Duration(p.between(1, 200))*time.Hour)))
				c.content = p.sentence(pick(p, replyTemplates), x)
				c.chapter = parent.chapter
			} else {
				c.content = p.sentence(pick(p, commentTemplates), x)
				if len(w.chapters) > 1 && p.chance(0.5) {
					ch := pick(p, w.chapters)
					if !ch.draft && ch.published.Before(c.created) {
						c.chapter = ch
					}
				}
			}
			c.content = html.Clean("<p>" + c.content + "</p>")
			if p.chance(0.04) {
				e := p.capNow(c.created.Add(time.Duration(p.between(1, 48)) * time.Hour))
				c.edited = &e
			}
			made = append(made, c)
			p.comments = append(p.comments, c)
		}
	}
}

// whoComments chooses the author of a comment: a creator answering a reply,
// a member, or a guest. It reports false if nobody suitable was found.
func (p *plan) whoComments(c *comment, w *work) bool {
	guestOK := w.commentPolicy == "open" && !w.restricted
	creatorReplies := c.parent != nil && !p.isCreator(c.parent.acct, w) && p.chance(0.5)
	if creatorReplies && !p.isApart(w.owner, c.parent.acct) {
		c.acct, c.pseud = w.owner, w.creators[0]
		return true
	}
	for tries := 0; tries < 30; tries++ {
		if guestOK && p.chance(0.14) {
			c.guestName = pick(p, guestNames)
			c.guestEmail = strings.ReplaceAll(strings.ToLower(c.guestName), " ", ".") + "@" + Domain
			return true
		}
		a := pick(p, p.accounts)
		if !p.mayReach(a, w) {
			continue
		}
		if c.parent != nil && (c.parent.acct == a || p.isApart(c.parent.acct, a)) {
			continue
		}
		c.acct = a
		c.pseud = a.pseuds[0]
		if len(a.pseuds) > 1 && p.chance(0.3) {
			c.pseud = pick(p, a.pseuds)
		}
		if a.joined.After(c.created) {
			c.created = p.capNow(a.joined.Add(time.Hour))
		}
		return true
	}
	return false
}

func (p *plan) makeBookmarks() {
	draw := p.weighted(p.visible())
	seen := map[[2]int]bool{}
	for tries := 0; len(p.marks) < bookmarkGoal && tries < 5000; tries++ {
		w, a := draw(), pick(p, p.accounts)
		key := [2]int{w.idx, a.n}
		if seen[key] || !p.mayReach(a, w) {
			continue
		}
		seen[key] = true
		b := &bookmark{id: p.id(), work: w, acct: a, pseud: a.pseuds[0], private: p.chance(0.2), rec: p.chance(0.1),
			created: p.after(later(w.published, a.joined), 2)}
		if len(a.pseuds) > 1 && p.chance(0.3) {
			b.pseud = pick(p, a.pseuds)
		}
		if p.chance(0.3) {
			b.notes = html.Clean("<p>" + pick(p, bookmarkNotes) + "</p>")
		}
		if p.chance(0.35) {
			for k := p.between(1, 3); k > 0; k-- {
				if t := pick(p, bookmarkTags); !contains(b.tags, t) {
					b.tags = append(b.tags, t)
				}
			}
		}
		p.marks = append(p.marks, b)
	}
	sort.SliceStable(p.marks, func(i, j int) bool { return p.marks[i].created.Before(p.marks[j].created) })
}

func (p *plan) makeSeries() {
	// Works a series may hold: the owner's own, finished posting, and not
	// hiding their creator (a series page names its creator).
	free := map[*account][]*work{}
	for _, w := range p.works {
		if !w.draft && !w.hidesCreator() && len(w.creators) == 1 {
			free[w.owner] = append(free[w.owner], w)
		}
	}
	for tries := 0; len(p.series) < seriesGoal && tries < 200; tries++ {
		a := pick(p, p.accounts[:20])
		avail := free[a]
		if len(avail) < 2 {
			continue
		}
		k := p.between(2, 5)
		if k > len(avail) {
			k = len(avail)
		}
		p.r.Shuffle(len(avail), func(i, j int) { avail[i], avail[j] = avail[j], avail[i] })
		chosen := append([]*work(nil), avail[:k]...)
		free[a] = avail[k:]
		sort.Slice(chosen, func(i, j int) bool { return chosen[i].published.Before(chosen[j].published) })
		s := &series{id: p.id(), owner: a, works: chosen, title: pick(p, seriesTitles), desc: html.Clean("<p>" + pick(p, seriesDescriptions) + "</p>"),
			created: chosen[0].published, updated: chosen[0].updated}
		for _, w := range chosen {
			if w.updated.After(s.updated) {
				s.updated = w.updated
			}
		}
		s.complete = true
		for _, w := range chosen {
			s.complete = s.complete && w.complete
		}
		// Titles stay different, so two series do not look the same.
		for _, o := range p.series {
			if o.title == s.title {
				s.title += " " + pick(p, []string{"II", "Redux", "Revisited", "Again"})
			}
		}
		p.series = append(p.series, s)
	}
}

func (p *plan) makeCollections() {
	// Which works are in which collection is decided here; the flags that
	// hide creators were set above with the works themselves.
	var normal, anon, unrev []*work
	for _, w := range p.works {
		switch {
		case w.draft:
		case w.inAnon:
			anon = append(anon, w)
		case w.inUnrevealed:
			unrev = append(unrev, w)
		case !w.anon && !w.restricted && len(normal) < 14 && p.chance(0.3):
			normal = append(normal, w)
		}
	}
	specs := []struct {
		name, title, desc    string
		owner                int
		open, mod, anonymous bool
		works                []*work
	}{
		{"quiet-hours", "Quiet Hours: Cozy Reading", "Gentle stories for slow evenings: tea, rain and no hurry. Anyone may add their own work.", 4, true, false, false, normal},
		{"midnight-postbox", "The Midnight Postbox (Anonymous)", "Works posted without their creators' names. The names stay private.", 10, false, false, true, anon},
		{"lantern-swap", "Lantern Swap (Unrevealed)", "A swap whose works stay unrevealed for now: titles and creators are hidden from readers.", 16, false, true, false, unrev},
	}
	for _, s := range specs {
		o := p.accounts[s.owner-1]
		c := &collection{id: p.id(), name: s.name, title: s.title, desc: html.Clean("<p>" + s.desc + "</p>"), owner: o, pseud: o.pseuds[0],
			open: s.open, moderated: s.mod, anonymous: s.anonymous, works: s.works,
			created: p.when(later(o.joined, p.now.Add(-600*24*time.Hour)), p.now.Add(-300*24*time.Hour))}
		p.colls = append(p.colls, c)
	}
}

func (p *plan) makeSubscriptions() {
	seen := map[string]bool{}
	add := func(a *account, kind string, target uuid.UUID, name string, events []string) {
		key := fmt.Sprint(a.n, kind, target)
		if seen[key] {
			return
		}
		seen[key] = true
		freq := "immediate"
		if p.chance(0.3) {
			freq = "daily"
		}
		p.subs = append(p.subs, &subscription{id: p.id(), acct: a, kind: kind, target: target, name: name, events: events, freq: freq,
			created: p.when(later(a.joined, p.now.Add(-400*24*time.Hour)), p.now.Add(-5*24*time.Hour))})
	}
	visible := p.visible()
	for len(p.subs) < 6 { // authors, by pseud
		w := pick(p, visible)
		if w.hidesCreator() {
			continue
		}
		a := pick(p, p.accounts)
		if !p.mayReach(a, w) {
			continue
		}
		add(a, "author", w.creators[0].id, w.creators[0].name, []string{"new_work", "work_updated"})
	}
	for len(p.subs) < 10 { // works
		w := pick(p, visible)
		a := pick(p, p.accounts)
		if !p.mayReach(a, w) || w.inUnrevealed {
			continue
		}
		add(a, "work", w.id, w.title, []string{"work_updated", "new_work"})
	}
	for len(p.subs) < 13 && len(p.series) > 0 { // series
		s := pick(p, p.series)
		a := pick(p, p.accounts)
		if a == s.owner || p.isApart(a, s.owner) {
			continue
		}
		add(a, "series", s.id, s.title, []string{"series_updated", "work_updated"})
	}
}
