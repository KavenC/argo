package argo

import "strings"

// State keeps the state withing a argument parsing call
type State struct {
	// String reply after arguments are parsed
	OutputStr strings.Builder
	doArgs    []string
}

// Args returns arguments consumed by triggering Action
// This function is only valid inside a Action.Do() call
func (s *State) Args() []string {
	return s.doArgs
}
