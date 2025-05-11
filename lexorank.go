package lexorank

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"
)

// CharacterSet defines a set of characters that can be used for key generation.
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

// NewASCIICharacterSet creates a new CharacterSet from a string of ASCII characters.
func NewASCIICharacterSet(set string) (CharacterSet, error) {
	runes := []rune(set)
	slices.Sort(runes)
	var runeToIndex [128]int
	for i, r := range runes {
		if !isASCII(r) {
			return nil, fmt.Errorf("invalid character set: '%c' is not an ASCII character", r)
		}
		if runeToIndex[r] != 0 {
			return nil, fmt.Errorf("invalid character set: '%c' is duplicated", r)
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
	index := midIndex % len(c.runes)
	return c.runes[index]
}

func isASCII(r rune) bool {
	return r >= 0 && r <= unicode.MaxASCII
}

// ValidateCharacterSet checks if the character set is valid by ensuring that
// the characters are in ascending order and that there are no duplicates.
func ValidateCharacterSet(set CharacterSet) error {
	r := set.Min()
	for {
		next, ok := set.Next(r)
		if !ok {
			break
		}
		if r >= next {
			return fmt.Errorf("invalid character set: '%c' >= '%c'", r, next)
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
			return fmt.Errorf("invalid character set: '%c' <= '%c'", r, prev)
		}
		r = prev
	}
	return nil
}

// Key represents a lexicographically sortable string key.
type Key string

func (k Key) String() string {
	return string(k)
}

// BucketKey represents a Key within a specific bucket namespace.
type BucketKey string

// String returns a string representation of the BucketKey in the format "bucket|key".
func (k BucketKey) String() string {
	return string(k)
}

// Generator is responsible for creating and managing lexicographically sortable keys.
type Generator struct {
	characterSet CharacterSet
	initial      string
}

var (
	// DefaultCharacterSet is the standard character set used for key generation.
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

// NewGenerator creates a new Generator with the specified options.
func NewGenerator(opts ...GeneratorOption) *Generator {
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
	return g
}

// Between generates a key that comes between the prevKey and nextKey keys.
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
			return "", fmt.Errorf("next character of min character '%c' not found: %q - %q", g.characterSet.Min(), prevKey, nextKey)
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
		return "", fmt.Errorf("cannot generate key strictly before %q as it (or its prefix) consists of all min characters from the set: %q - %q", nextKey, prevKey, nextKey)
	}

	if prevKey > nextKey {
		return "", fmt.Errorf("prevKey (%q) must be strictly less than nextKey (%q)", prevKey, nextKey)
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

	mid := g.characterSet.Mid(g.characterSet.Min(), g.characterSet.Max())
	for i, prevChar := range prevRunes {
		nextChar := nextRunes[i]
		if prevChar == nextChar {
			continue
		}
		next := g.characterSet.Mid(prevChar, nextChar)

		if next > prevChar {
			result := append(prevRunes[:i], next)
			for j := i + 1; j < len(prevRunes); j++ {
				result = append(result, mid)
			}
			return Key(result), nil
		}
		if next < nextChar && runesGreaterThan(nextRunes[:i], prevRunes[:i]) {
			result := append(nextRunes[:i], next)
			for j := i + 1; j < len(prevRunes); j++ {
				result = append(result, mid)
			}
			return Key(result), nil
		}
	}

	return Key(prevRunes) + Key(mid), nil
}

func runesGreaterThan(a, b []rune) bool {
	if len(a) != len(b) {
		panic("runesGreaterThan: lengths of a and b must be equal")
	}
	for i := 0; i < len(a); i++ {
		if a[i] > b[i] {
			return true
		}
		if a[i] < b[i] {
			return false
		}
	}
	return false
}

// Next generates a key that comes after the given key.
func (g *Generator) Next(key Key) (Key, error) {
	return g.Between(key, "")
}

// Prev generates a key that comes before the given key.
func (g *Generator) Prev(key Key) (Key, error) {
	return g.Between("", key)
}

type generatorOption func(*Generator)

// GeneratorOption is a option for configuring the Generator.
type GeneratorOption generatorOption

// WithCharacterSet returns a GeneratorOption that sets the character set used by the Generator.
func WithCharacterSet(set CharacterSet) GeneratorOption {
	return func(g *Generator) {
		g.characterSet = set
	}
}

// WithInitial returns a GeneratorOption that sets the initial key value used by the Generator.
func WithInitial(initial string) GeneratorOption {
	return func(r *Generator) {
		r.initial = initial
	}
}

// Bucket represents a namespace for keys, allowing separate key sequences in different buckets.
type Bucket struct {
	defaultPrefix string
	separator     rune
	generator     *Generator
}

// NewBucket creates a new Bucket with the specified name and Generator.
func NewBucket(opts ...BucketOption) *Bucket {
	b := &Bucket{
		"0",
		'|',
		nil,
	}
	for _, opt := range opts {
		opt(b)
	}
	if b.generator == nil {
		b.generator = NewGenerator()
	}
	return b
}

// Between generates a key that comes between the prev and next keys within this bucket.
func (b *Bucket) Between(prev, next BucketKey) (BucketKey, error) {
	var prefix string
	var prevKey Key
	if prev != "" {
		prevBucket, key := b.SplitBucketKey(prev)
		if prevBucket == "" {
			return "", errors.New("prev key is not in format of bucket key")
		}
		prevKey = key
		prefix = prevBucket
	}
	var nextKey Key
	if next != "" {
		nextBucket, key := b.SplitBucketKey(next)
		if nextBucket == "" {
			return "", errors.New("next key is not in format of bucket key")
		}
		if prefix != "" && prefix != nextBucket {
			return "", fmt.Errorf("%w: %q != %q", ErrBucketMismatch, prefix, nextBucket)
		}
		nextKey = key
		prefix = nextBucket
	}

	k, err := b.generator.Between(prevKey, nextKey)
	if err != nil {
		return "", err
	}
	return b.createBucketKey(prefix, k), nil
}

var ErrBucketMismatch = errors.New("bucket mismatch")

func (b *Bucket) SplitBucketKey(key BucketKey) (string, Key) {
	parts := strings.SplitN(string(key), string(b.separator), 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], Key(parts[1])
}

func (b *Bucket) createBucketKey(bucket string, key Key) BucketKey {
	if bucket == "" {
		bucket = b.defaultPrefix
	}
	return BucketKey(fmt.Sprintf("%s%c%s", bucket, b.separator, key))
}

type bucketOption func(*Bucket)

// BucketOption is a option for configuring the Bucket.
type BucketOption bucketOption

// WithSeparator returns a BucketOption that sets the separator of BucketKey.
func WithSeparator(sep rune) BucketOption {
	return func(g *Bucket) {
		g.separator = sep
	}
}

// WithGenerator returns a BucketOption that sets the Generator of Bucket.
func WithGenerator(g *Generator) BucketOption {
	return func(b *Bucket) {
		b.generator = g
	}
}

// WithDefaultPrefix returns a BucketOption that sets the default prefix of BucketKey.
// The default prefix is only used for the initial key generation.
func WithDefaultPrefix(prefix string) BucketOption {
	return func(b *Bucket) {
		b.defaultPrefix = prefix
	}
}
