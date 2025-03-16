package actions

import (
	"fmt"
	"strings"
)

type Action int

const (
	Nothing Action = iota
	Print
	DeleteSource
	DeleteTarget
)

func GetAction(action string) Action {
	switch strings.ToLower(action) {
	case "nothing":
		return Nothing
	case "print":
		return Print
	case "delete-source":
		return DeleteSource
	case "delete-target":
		return DeleteTarget
	default:
		fmt.Printf("invalid action '%s'\n", action)
		return Nothing
	}
}
