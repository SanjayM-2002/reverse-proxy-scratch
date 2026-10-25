package router

import (
	"fmt"
	"strings"
)

type Op string

const (
	Read  Op = "READ"
	Write Op = "WRITE"
)

// Command is everything the router needs to know about a request: is it a read or a write, and which key.
type Command struct {
	Op  Op
	Key string
}

// ParseCommand turns "READ user:42" into Command{Op: Read, Key: "user:42"}.
func ParseCommand(line string) (Command, error) {
	parts := strings.Fields(line)
	if len(parts) != 2 {
		return Command{}, fmt.Errorf("expected '<READ|WRITE> <key>', got %q", line)
	}

	op := Op(strings.ToUpper(parts[0]))
	if op != Read && op != Write {
		return Command{}, fmt.Errorf("unknown op %q (use READ or WRITE)", parts[0])
	}

	return Command{Op: op, Key: parts[1]}, nil
}
