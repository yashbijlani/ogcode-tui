package indexer

// stopWords is a comprehensive set of English stop words used to filter
// low-information tokens from PDF page text before indexing.
var stopWords = map[string]struct{}{
	"a": {}, "about": {}, "above": {}, "after": {}, "again": {}, "against": {},
	"ago": {}, "all": {}, "also": {}, "although": {}, "am": {}, "an": {},
	"and": {}, "another": {}, "any": {}, "are": {}, "aren't": {}, "around": {},
	"as": {}, "at": {}, "away": {}, "back": {}, "be": {}, "because": {},
	"been": {}, "before": {}, "being": {}, "below": {}, "between": {}, "both": {},
	"but": {}, "by": {}, "can": {}, "cannot": {}, "can't": {}, "could": {},
	"couldn't": {}, "did": {}, "didn't": {}, "do": {}, "does": {}, "doesn't": {},
	"doing": {}, "done": {}, "don't": {}, "down": {}, "during": {}, "each": {},
	"either": {}, "else": {}, "enough": {}, "even": {}, "ever": {}, "every": {},
	"except": {}, "few": {}, "finally": {}, "for": {}, "found": {}, "from": {},
	"get": {}, "give": {}, "given": {}, "go": {}, "going": {}, "got": {},
	"had": {}, "hadn't": {}, "has": {}, "hasn't": {}, "have": {}, "haven't": {},
	"having": {}, "he": {}, "he'd": {}, "he'll": {}, "he's": {}, "hence": {},
	"her": {}, "here": {}, "here's": {}, "hers": {}, "herself": {}, "him": {},
	"himself": {}, "his": {}, "how": {}, "however": {}, "i": {}, "i'd": {},
	"if": {}, "i'll": {}, "i'm": {}, "in": {}, "into": {}, "is": {}, "isn't": {},
	"it": {}, "its": {}, "it's": {}, "itself": {}, "i've": {}, "just": {},
	"keep": {}, "know": {}, "last": {}, "let": {}, "let's": {}, "like": {},
	"likely": {}, "long": {}, "look": {}, "made": {}, "make": {}, "many": {},
	"may": {}, "maybe": {}, "me": {}, "might": {}, "more": {}, "most": {},
	"much": {}, "my": {}, "myself": {}, "need": {}, "new": {}, "next": {},
	"no": {}, "nor": {}, "not": {}, "now": {}, "of": {}, "off": {}, "often": {},
	"on": {}, "once": {}, "only": {}, "onto": {}, "or": {}, "other": {},
	"otherwise": {}, "our": {}, "ours": {}, "ourselves": {}, "out": {}, "over": {},
	"own": {}, "part": {}, "per": {}, "perhaps": {}, "point": {}, "put": {},
	"rather": {}, "really": {}, "right": {}, "said": {}, "same": {}, "say": {},
	"see": {}, "set": {}, "several": {}, "she": {}, "she'd": {}, "she'll": {},
	"she's": {}, "should": {}, "shouldn't": {}, "since": {}, "so": {}, "some": {},
	"still": {}, "such": {}, "take": {}, "than": {}, "that": {}, "that's": {},
	"the": {}, "their": {}, "theirs": {}, "them": {}, "themselves": {}, "then": {},
	"there": {}, "there's": {}, "therefore": {}, "these": {}, "they": {},
	"they'd": {}, "they'll": {}, "they're": {}, "they've": {}, "this": {},
	"those": {}, "though": {}, "through": {}, "thus": {}, "to": {}, "together": {},
	"too": {}, "toward": {}, "under": {}, "until": {}, "up": {}, "upon": {},
	"us": {}, "use": {}, "used": {}, "using": {}, "very": {}, "via": {},
	"was": {}, "wasn't": {}, "we": {}, "we'd": {}, "well": {}, "we'll": {},
	"were": {}, "we're": {}, "weren't": {}, "we've": {}, "what": {}, "what's": {},
	"when": {}, "whenever": {}, "where": {}, "whereas": {}, "wherever": {},
	"whether": {}, "which": {}, "while": {}, "who": {}, "whoever": {}, "whole": {},
	"whom": {}, "whose": {}, "why": {}, "will": {}, "with": {}, "within": {},
	"without": {}, "won't": {}, "would": {}, "wouldn't": {}, "yes": {}, "yet": {},
	"you": {}, "you'd": {}, "you'll": {}, "your": {}, "you're": {}, "yours": {},
	"yourself": {}, "yourselves": {}, "you've": {}, "zero": {},
	// Common punctuation-adjacent noise
	"'s": {}, "'re": {}, "'ve": {}, "'ll": {}, "'d": {}, "'t": {},
}

// isStopWord returns true if the word is a known English stop word.
func isStopWord(w string) bool {
	_, ok := stopWords[w]
	return ok
}
