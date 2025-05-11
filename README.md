# go-lexorank

[![Go Reference](https://pkg.go.dev/badge/github.com/morikuni/go-lexorank.svg)](https://pkg.go.dev/github.com/morikuni/go-lexorank)

A Go implementation of LexoRank - a system for generating lexicographically sortable string keys. This is useful for
maintaining ordered lists where items need to be inserted between existing items without reordering the entire list.

## What is LexoRank?

LexoRank is a ranking system that generates string keys that can be lexicographically sorted. It's commonly used in
applications like Jira for ordering tasks in a list. The key feature is the ability to insert new items between existing
ones without having to reorder everything.

For example, if you have items with ranks "a" and "c", you can insert a new item with rank "b" between them. If you need
to insert another item between "a" and "b", you can generate a rank like "a5".

## Features

- Generate lexicographically sortable string keys
- Insert keys between existing keys
- Customizable character sets
- Support for bucketed keys (namespaced keys)
- No external dependencies

## Installation

```bash
go get github.com/morikuni/go-lexorank
```

## Usage

### Basic Usage

```go
package main

import (
	"fmt"

	"github.com/morikuni/go-lexorank"
)

func main() {
	// Create a new generator with default settings
	generator := lexorank.NewGenerator()

	// Generate an initial key
	key1, _ := generator.Between("", "")
	fmt.Println("Initial key:", key1)

	// Generate a key after key1
	key2, _ := generator.Next(key1)
	fmt.Println("Next key:", key2)

	// Generate a key before key1
	key0, _ := generator.Prev(key1)
	fmt.Println("Previous key:", key0)

	// Generate a key between key0 and key1
	keyMiddle, _ := generator.Between(key0, key1)
	fmt.Println("Middle key:", keyMiddle)
}
```

### Custom Character Set

```go
package main

import (
	"fmt"

	"github.com/morikuni/go-lexorank"
)

func main() {
	// Create a custom character set
	charSet, _ := lexorank.NewASCIICharacterSet("0123456789")

	// Create a generator with the custom character set
	generator := lexorank.NewGenerator(lexorank.WithCharacterSet(charSet))

	// Generate keys
	key1, _ := generator.Between("", "")
	key2, _ := generator.Next(key1)

	fmt.Println("Key 1:", key1)
	fmt.Println("Key 2:", key2)
}
```

### Using Buckets

```go
package main

import (
	"fmt"

	"github.com/morikuni/go-lexorank"
)

func main() {
	// Create a bucket
	bucket := lexorank.NewBucket()

	// Generate keys in the bucket
	key1, _ := bucket.Between("", "")
	fmt.Println("Initial bucket key:", key1)

	key2, _ := bucket.Between(key1, "")
	fmt.Println("Next bucket key:", key2)
}
```

## Try it out in the [Go Playground](https://go.dev/play/p/wIDGUfgrXhs?v=).

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
