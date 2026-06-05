package model

import (
	"fmt"
	"strings"
)

const MaxURLLength = 2000

func ValidateURL(u string) error {
	if u == "" {
		return nil
	}
	if len(u) > MaxURLLength {
		return fmt.Errorf("url must be at most %d characters", MaxURLLength)
	}
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		return fmt.Errorf("url must start with http:// or https://")
	}
	return nil
}
