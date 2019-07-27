package cli

import "strings"

type ArrayFlags []string

func (it *ArrayFlags) String() string {
	return strings.Join(*it, ", ")
}

func (it *ArrayFlags) Set(value string) error {
	*it = append(*it, value)
	return nil
}