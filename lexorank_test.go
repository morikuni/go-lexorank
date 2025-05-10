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
		{"700", "701", "7004"},
		{"700", "7004", "7002"},
		{"7004", "701", "7005"},
		{"7004", "7040", "7020"},
		{"", "700", "699"},
		{"699", "700", "6994"},
		{"6994", "700", "6995"},
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
