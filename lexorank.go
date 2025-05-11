package lexorank

import (
	"fmt"
	"slices"
	"strings"
	"unicode"
)

type CharacterSet interface {
	Min() rune
	Max() rune
	// Next should return a next character in the set.
	Next(rune) (rune, bool)
	// Prev should return a previous character in the set.
	Prev(rune) (rune, bool)
	// Mid should return a character at the midpoint between a and b,
	// treating the character set as a circular sequence.
	// If b comes before a, it wraps around the end of the set.
	//
	// Examples of "0123456789":
	// - Mid('2', '5') → '3' or '4' (2→3→4→5)
	// - Mid('8', '2') → '0' (8→9→0→1→2)
	Mid(rune, rune) rune
}

type characterSet struct {
	runes       []rune
	runeToIndex [128]int
}

func NewASCIICharacterSet(set string) (CharacterSet, error) {
	runes := []rune(set)
	slices.Sort(runes)
	var runeToIndex [128]int
	for i, r := range runes {
		if !isASCII(r) {
			return nil, fmt.Errorf("invalid character set: %c is not an ASCII character", r)
		}
		if runeToIndex[r] != 0 {
			return nil, fmt.Errorf("invalid character set: %c is duplicated", r)
		}
		runeToIndex[r] = i
	}
	return &characterSet{
		runes,
		runeToIndex,
	}, nil
}

func (c *characterSet) Min() rune {
	return c.runes[0]
}

func (c *characterSet) Max() rune {
	return c.runes[len(c.runes)-1]
}

func (c *characterSet) Next(r rune) (rune, bool) {
	index := c.runeToIndex[r]
	if index == len(c.runes)-1 {
		return 0, false
	}
	next := c.runes[index+1]
	return next, true
}

func (c *characterSet) Prev(r rune) (rune, bool) {
	index := c.runeToIndex[r]
	if index == 0 {
		return 0, false
	}
	prev := c.runes[index-1]
	return prev, true
}

func (c *characterSet) Mid(a, b rune) rune {
	indexA := c.runeToIndex[a]
	indexB := c.runeToIndex[b]
	if indexB < indexA {
		indexB += len(c.runes)
	}
	midIndex := (indexA + indexB) / 2
	return c.runes[midIndex]
}

func isASCII(r rune) bool {
	return r >= 0 && r <= unicode.MaxASCII
}

func ValidateCharacterSet(set CharacterSet) error {
	r := set.Min()
	for {
		next, ok := set.Next(r)
		if !ok {
			break
		}
		if r >= next {
			return fmt.Errorf("invalid character set: %c >= %c", r, next)
		}
		r = next
	}
	r = set.Max()
	for {
		prev, ok := set.Prev(r)
		if !ok {
			break
		}
		if r <= prev {
			return fmt.Errorf("invalid character set: %c <= %c", r, prev)
		}
		r = prev
	}
	return nil
}

type Key string

func (k Key) String() string {
	return string(k)
}

func (k Key) WithBucket(bucket string) BucketKey {
	return BucketKey{
		bucket,
		k,
	}
}

type BucketKey struct {
	bucket string
	key    Key
}

func (k BucketKey) String() string {
	return fmt.Sprintf("%s|%s", k.bucket, k.key)
}

func (k BucketKey) Key() Key {
	return k.key
}

type Generator struct {
	characterSet CharacterSet
	initial      string
}

var (
	DefaultCharacterSet = mustCharacterSet(NewASCIICharacterSet("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"))
)

func defaultInitial(cs CharacterSet) string {
	return strings.Repeat(string(cs.Mid(cs.Min(), cs.Max())), 6)
}

func mustCharacterSet(set CharacterSet, err error) CharacterSet {
	if err != nil {
		panic(err)
	}
	return set
}

func NewGenerator(opts ...GeneratorOption) (*Generator, error) {
	g := &Generator{
		DefaultCharacterSet,
		"",
	}
	for _, opt := range opts {
		opt(g)
	}
	if g.initial == "" {
		g.initial = defaultInitial(g.characterSet)
	}
	// No need to check if characters are in the character set
	return g, nil
}

func (g *Generator) Between(prevKey, nextKey Key) (Key, error) {
	if prevKey == "" && nextKey == "" {
		return Key(g.initial), nil
	}

	if nextKey == "" {
		runes := []rune(prevKey)
		n := len(runes)
		for i := n - 1; i >= 0; i-- {
			charToIncrement := runes[i]
			incrementedChar, ok := g.characterSet.Next(charToIncrement)
			if ok {
				runes[i] = incrementedChar
				for j := i + 1; j < n; j++ {
					runes[j] = g.characterSet.Min()
				}
				return Key(runes), nil
			}
		}
		// If the min character is used here, generating a key between prevKey and generated key will be impossible.
		// For example, if prevKey was "000" and generated key was "0000", no key can be generated between them.
		// If the generated key is "0001", a key between "000" and "0001" can be "00004".
		nextToMin, ok := g.characterSet.Next(g.characterSet.Min())
		if !ok {
			return "", fmt.Errorf("next character of min character '%c' not found", g.characterSet.Min())
		}
		return Key(string(prevKey) + string(nextToMin)), nil
	}

	if prevKey == "" {
		runes := []rune(nextKey)
		n := len(runes)
		for i := n - 1; i >= 0; i-- {
			charToDecrement := runes[i]
			decrementedChar, ok := g.characterSet.Prev(charToDecrement)
			if ok {
				runes[i] = decrementedChar
				for j := i + 1; j < n; j++ {
					runes[j] = g.characterSet.Max()
				}
				return Key(runes), nil
			}
		}
		return "", fmt.Errorf("cannot generate key strictly before '%s' as it (or its prefix) consists of all min characters from the set", nextKey)
	}

	if prevKey > nextKey {
		return "", fmt.Errorf("prevKey ('%s') must be strictly less than nextKey ('%s')", prevKey, nextKey)
	}

	prevRunes := []rune(string(prevKey))
	nextRunes := []rune(string(nextKey))
	switch n := len(prevRunes) - len(nextRunes); {
	case n > 0:
		for i := 0; i < n; i++ {
			nextRunes = append(nextRunes, g.characterSet.Min())
		}
	case n < 0:
		for i := 0; i < -n; i++ {
			prevRunes = append(prevRunes, g.characterSet.Min())
		}
	}

	var commonPrefix []rune
	i := 0
	for i < len(prevRunes) && i < len(nextRunes) && prevRunes[i] == nextRunes[i] {
		commonPrefix = append(commonPrefix, prevRunes[i])
		i++
	}

	if i == len(prevRunes) {
		return prevKey + Key(g.characterSet.Mid(g.characterSet.Min(), g.characterSet.Max())), nil
	}

	prefix := commonPrefix
	for ; i < len(prevRunes); i++ {
		prevChar := prevRunes[i]
		nextChar := nextRunes[i]
		next := g.characterSet.Mid(prevChar, nextChar)

		if next > prevChar {
			result := append(prefix[:i], next)
			for j := i + 1; j < len(prevRunes); j++ {
				result = append(result, g.characterSet.Min())
			}
			return Key(result), nil
		}

		prefix = append(prefix, prevChar)
	}

	return Key(prevRunes) + Key(g.characterSet.Mid(g.characterSet.Min(), g.characterSet.Max())), nil
}

// Next generates a key that comes after the given key
func (g *Generator) Next(key Key) (Key, error) {
	return g.Between(key, "")
}

// Prev generates a key that comes before the given key
func (g *Generator) Prev(key Key) (Key, error) {
	return g.Between("", key)
}

type option func(*Generator)
type GeneratorOption option

func WithCharacterSet(set CharacterSet) GeneratorOption {
	return func(g *Generator) {
		g.characterSet = set
	}
}

func WithInitial(initial string) GeneratorOption {
	return func(r *Generator) {
		r.initial = initial
	}
}

type Bucket struct {
	name      string
	generator *Generator
}

func NewBucket(name string, g *Generator) *Bucket {
	return &Bucket{
		name,
		g,
	}
}

func (b *Bucket) Between(prev, next BucketKey) (BucketKey, error) {
	k, err := b.generator.Between(prev.key, next.key)
	if err != nil {
		return BucketKey{}, err
	}
	return BucketKey{
		b.name,
		k,
	}, nil
}
