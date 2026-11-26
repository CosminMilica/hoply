package cvdoc

import (
	"strings"
)

// StockWords are the filler words a CV avoids: the CV prompt forbids them and the style check flags them, as whole
// words.
var StockWords = []string{"leverage", "leveraged", "leveraging", "spearheaded", "spearheading", "orchestrated", "robust",
	"seamless", "seamlessly", "cutting-edge", "state-of-the-art", "end-to-end", "holistic", "synergy", "synergies",
	"utilize", "utilized", "utilizing", "facilitated", "empowered", "empowering", "streamlined", "streamlining",
	"ensure", "ensured", "ensuring", "passionate", "dynamic", "innovative", "best-in-class", "world-class",
	"results-driven", "proven track record", "comprehensive", "successfully", "effectively", "efficiently",
	"pivotal", "instrumental", "foster", "fostered", "elevate", "elevated", "showcase", "showcased", "various", "full-stack observability"}

// LetterCliches are the phrases a cover letter or a message never says: the letter prompt forbids them, the letter
// check refuses a sentence that holds one and the style check flags it.
var LetterCliches = []string{"passionate", "team player", "hard-working", "hardworking", "results-driven", "detail-oriented",
	"perfect fit", "perfect candidate", "dream job", "i am writing to", "i'm writing to", "i believe i would be",
	"to whom it may concern", "just checking in", "look forward to hearing", "please find attached",
	"would love the opportunity", "excited to", "thrilled", "exceptional addition", "uniquely positions",
	"hit the ground running", "add value", "make an impact", "touch base", "reach out", "one more reason i fit",
	"robust", "leverage"}

// Quoted lists phrases for a prompt: "a", "b" and "c".
func Quoted(phrases []string) string {
	quoted := make([]string, len(phrases))
	for i, phrase := range phrases {
		quoted[i] = `"` + phrase + `"`
	}
	return strings.Join(quoted, ", ")
}
