package lexorank

import (
	"errors"
	"fmt"
	"strings"
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

	g := NewGenerator(WithCharacterSet(charSet), WithInitial("555"))

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
		{"6994", "700", "6997"},
		{"999", "", "9991"},
		{"999", "9991", "99904"},
		{"700", "701", "7004"},
		{"700", "7004", "7002"},
		{"7004", "701", "7007"},
		{"7004", "7040", "7024"},
		{"079", "1", "084"},
		{"08", "1", "09"},
		{"098", "1", "099"},
		{"0998", "1", "0999"},
		{"088", "089", "0884"},
		{"569", "570", "5694"},
		{"569", "571", "570"},
		{"569", "572", "570"},
		{"569", "573", "571"},
		{"5690", "573", "5714"},
		{"5699", "573", "5714"},
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
		testRecursive(t, g, "", "", 20)
	})

	t.Run("keep generating key between target", func(t *testing.T) {
		var keep Key
		for i := 0; i < 9999; i++ {
			key, err := g.Between(keep, "1")
			noError(t, err)
			validateKey(t, key, keep, "1")
			keep = key
		}
		t.Logf("last key: %s", keep)
		keep = ""
		for i := 0; i < 9999; i++ {
			key, err := g.Between("0", keep)
			noError(t, err)
			validateKey(t, key, "0", keep)
			keep = key
		}
		t.Logf("last key: %s", keep)
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

func equalBucketKey(t *testing.T, got, want BucketKey) {
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

func testRecursive(t *testing.T, g *Generator, prev, next Key, depth int) {
	if depth == 0 {
		return
	}
	key, err := g.Between(prev, next)
	noError(t, err)
	validateKey(t, key, prev, next)
	testRecursive(t, g, key, next, depth-1)
	testRecursive(t, g, prev, key, depth-1)
}

func TestBucket(t *testing.T) {
	charSet, err := NewASCIICharacterSet("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")
	noError(t, err)

	g := NewGenerator(WithCharacterSet(charSet), WithInitial("555"))
	bucket := NewBucket(WithGenerator(g))

	for _, tt := range []struct {
		prev BucketKey
		next BucketKey
		want BucketKey
	}{
		{"", "", "0|555"},
		{"0|555", "", "0|556"},
		{"", "1|555", "1|554"},
	} {
		t.Run(fmt.Sprintf("%s_%s", tt.prev, tt.next), func(t *testing.T) {
			key, err := bucket.Between(tt.prev, tt.next)
			noError(t, err)
			equalBucketKey(t, key, tt.want)
		})
	}

	t.Run("error on bucket mismatch", func(t *testing.T) {
		_, err := bucket.Between("0|555", "1|555")
		if !errors.Is(err, ErrBucketMismatch) {
			t.Fatal("expected error, but got nil")
		}
	})
}

func FuzzGenerator_Between(f *testing.F) {
	chars := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	charSet, err := NewASCIICharacterSet(chars)
	if err != nil {
		f.Fatalf("failed to create character set: %v", err)
	}

	g := NewGenerator(WithCharacterSet(charSet))

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
	allZeros := func(s string) bool {
		for _, r := range s {
			if r != '0' {
				return false
			}
		}
		return true
	}
	isSameKey := func(prev, next string) bool {
		idx := strings.Index(next, prev)
		if idx == -1 {
			return false
		}
		if allZeros(next[idx+len(prev):]) {
			return true
		}
		return false
	}

	f.Fuzz(func(t *testing.T, prev, next string) {
		prevKey := Key(prev)
		nextKey := Key(next)

		if prevKey >= nextKey {
			t.Skip("prev key must be less than next key")
		}
		if !isValidCharInput(prev) || !isValidCharInput(next) {
			t.Skip("keys must be in the character set")
		}
		if allZeros(prev) || allZeros(next) {
			t.Skip("keys must not be all zeros")
		}
		if isSameKey(prev, next) {
			t.Skip("next key must not be the same as prev key")
		}

		testRecursive(t, g, prevKey, nextKey, 3)
	})
}
