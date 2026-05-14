package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateColor(t *testing.T) {
	valid := []string{"", "red", "dodgerblue", "gold", "mediumturquoise", "rebeccapurple"}
	for _, c := range valid {
		assert.NoError(t, ValidateColor(c), "ValidateColor(%q)", c)
	}

	invalid := []string{"notacolor", "RED", "DodgerBlue", "#ff0000", "rgb(0,0,0)", "123"}
	for _, c := range invalid {
		assert.Error(t, ValidateColor(c), "ValidateColor(%q)", c)
	}
}
