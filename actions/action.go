package actions

import (
	"fmt"
	"strings"
)

type Action int

const (
	Nothing Action = iota
	Print
	Delete
)

func GetAction(action string) Action {
	switch strings.ToLower(action) {
	case "nothing":
		return Nothing
	case "print":
		return Print
	case "delete":
		return Delete
	default:
		fmt.Printf("invalid action '%s'\n", action)
		return Nothing
	}
}
