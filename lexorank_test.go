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
	charSet, err := NewASCIICharacterSet("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")
	noError(t, err)

	g, err := NewGenerator(WithCharacterSet(charSet), WithInitial("aaa"))
	noError(t, err)

	for _, tt := range []struct {
		prev Key
		next Key
		want Key
	}{
		{"", "", "aaa"},
		{"aaa", "", "aab"},
		{"azy", "", "azz"},
		{"", "b01", "b00"},
		{"b00", "b01", "b00U"},
		{"b00", "b00U", "b00F"},
		{"b00U", "b010", "b00V"},
		{"b00U", "b040", "b020"},
		{"", "b00", "azz"},
		{"azz", "b00", "azzU"},
		{"azzU", "b00", "azzV"},
	} {
		t.Run(fmt.Sprintf("%s_%s", tt.prev, tt.next), func(t *testing.T) {
			key, err := g.Between(tt.prev, tt.next)
			noError(t, err)
			equalKey(t, key, tt.want)
			validateKey(t, key, tt.prev, tt.next)
		})
	}

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
