package lexorank

import (
	"fmt"
	"testing"
)

func TestNewASCIICharacterSet(t *testing.T) {
	charSet, err := NewASCIICharacterSet("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")
	noError(t, err)

	err = ValidateCharacterSet(charSet)
	noError(t, err)
}

func TestGenerator(t *testing.T) {
	charSet, err := NewASCIICharacterSet("0123456789")
	noError(t, err)

	g, err := NewGenerator(WithCharacterSet(charSet), WithInitial("555"))
	noError(t, err)

	for _, tt := range []struct {
		prev Key
		next Key
		want Key
	}{
		{"", "", "555"},
		{"555", "", "556"},
		{"599", "", "600"},
		{"", "701", "700"},
		{"", "700", "699"},
		{"699", "700", "6994"},
		{"6994", "700", "6996"},
		{"999", "", "9991"},
		{"999", "9991", "99904"},
		{"700", "701", "7004"},
		{"700", "7004", "7002"},
		{"7004", "701", "7006"},
		{"7004", "7040", "7020"},
	} {
		t.Run(fmt.Sprintf("%s_%s", tt.prev, tt.next), func(t *testing.T) {
			key, err := g.Between(tt.prev, tt.next)
			noError(t, err)
			equalKey(t, key, tt.want)
			validateKey(t, key, tt.prev, tt.next)
		})
	}

	t.Run("error on all min value", func(t *testing.T) {
		key, err := g.Between("", "001")
		noError(t, err)
		equalKey(t, key, "000")
		validateKey(t, key, "", "001")

		key, err = g.Between("", "000")
		if err == nil {
			t.Fatalf("expected error, got %s", key)
		}
	})

	t.Run("recursive", func(t *testing.T) {
		var check func(prev, next Key, depth int)
		check = func(prev, next Key, depth int) {
			if depth == 0 {
				return
			}
			key, err := g.Between(prev, next)
			noError(t, err)
			validateKey(t, key, prev, next)
			check(key, next, depth-1)
			check(prev, key, depth-1)
		}

		check("", "", 20) // 2^N times tested
	})
}

func noError(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func equalKey(t *testing.T, got, want Key) {
	t.Helper()
	if want != got {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func validateKey(t *testing.T, got, prev, next Key) {
	t.Helper()
	if prev != "" && got <= prev {
		t.Fatalf("%s-%s key %s should be greater than %s", prev, next, got, prev)
	}
	if next != "" && got >= next {
		t.Fatalf("%s-%s key %s should be less than %s", prev, next, got, next)
	}
}

func FuzzGenerator_Between(f *testing.F) {
	chars := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	charSet, err := NewASCIICharacterSet(chars)
	if err != nil {
		f.Fatalf("failed to create character set: %v", err)
	}

	g, err := NewGenerator(WithCharacterSet(charSet))
	if err != nil {
		f.Fatalf("failed to create generator: %v", err)
	}

	f.Add("", "")
	f.Add("a", "")
	f.Add("", "z")
	f.Add("a", "z")
	f.Add("abc", "def")
	f.Add("AAA", "AAA1")
	f.Add("YZZ0", "Z00")

	isValidCharInput := func(s string) bool {
		for _, r := range s {
			if !strings.ContainsRune(chars, r) {
				return false
			}
		}
		return true
	}
	isSameKey := func(next, prev string) bool {
		idx := strings.Index(next, prev)
		if idx == -1 {
			return false
		}
		for _, r := range next[len(prev):] {
			if r != '0' {
				return false
			}
		}
		return true
	}

	f.Fuzz(func(t *testing.T, prev, next string) {
		prevKey := Key(prev)
		nextKey := Key(next)

		if prev != "" && next != "" && prevKey >= nextKey {
			t.Skip("prev key must be less than next key")
		}
		if strings.HasPrefix(next, prev) {
			t.Skip("next key must not start with prev key")
		}
		if !isValidCharInput(prev) || !isValidCharInput(next) {
			t.Skip("keys must be in the character set")
		}
		if isSameKey(next, prev) {
			t.Skip("next key must not be the same as prev key")
		}

		key, err := g.Between(prevKey, nextKey)
		noError(t, err)
		validateKey(t, key, prevKey, nextKey)
	})
}
