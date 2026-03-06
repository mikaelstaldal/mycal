package model

import "testing"

func TestValidateColor(t *testing.T) {
	valid := []string{"", "red", "dodgerblue", "gold", "mediumturquoise", "rebeccapurple"}
	for _, c := range valid {
		if err := ValidateColor(c); err != nil {
			t.Errorf("ValidateColor(%q) = %v, want nil", c, err)
		}
	}

	invalid := []string{"notacolor", "RED", "DodgerBlue", "#ff0000", "rgb(0,0,0)", "123"}
	for _, c := range invalid {
		if err := ValidateColor(c); err == nil {
			t.Errorf("ValidateColor(%q) = nil, want error", c)
		}
	}
}
