package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDesktopQuote(t *testing.T) {
	assert.Equal(t, `"/home/test/My Cal/mycal-open"`, desktopQuote("/home/test/My Cal/mycal-open"))
	assert.Equal(t, `"https://example.com/a%%20b/"`, desktopQuote("https://example.com/a%20b/"))
	assert.Equal(t, `"a\\$b\\`+"`"+`c\\"d\\\\e"`, desktopQuote("a$b`c\"d\\e"))
}
