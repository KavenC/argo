package argo

import "fmt"

// Action defines the action to be done for the specified matching args
type Action struct {
	// Argument string that trigger this action
	Trigger string

	// Do is the fuction which will be executed if this Action is triggered
	// *State keeps the state of current parsing run. Vardic args will be forwarded from the Parse() call
	Do func(*State, ...interface{}) error

	// Minimum number of arguments, other than the triggering arg, that should be consumed by this action
	// Consumed args will be passed to Do() in State object
	// If MinConsume < 0, it will be fixed as MinConsume = 0 in Finalize() call
	MinConsume int

	// Maximum number of arguments, other than the triggering arg, that should be consumed by this action
	// Consumed args will be passed to Do() in State object
	// If MaxConsume < MinConsume, excpet MaxConsume < 0, MaxConsume will be set as MinConsume in Finalize() call
	// MaxConsume < 0 implies consuming all remaining args
	MaxConsume int

	// ShortDescr the one-line description of this Action
	ShortDescr string

	// LongDescr the complete description of this Action
	LongDescr string

	// HelpTextGenerator is used to format help text for this action and all it's subactions which do not have a HelpTextGenerator
	// If HelpTextGenerator is not assigned in this Action and any of its ancestors, a default HelpTextGenerator will be used
	HelpTextGenerator func(Action) string

	parent          *Action
	pathCached      string
	subActionLookup map[string]*Action
	subActionCopy   []Action
	finalized       bool
}

// SubActions returns all immediate SubActions
func (act Action) SubActions() []Action {
	return act.subActionCopy
}

// GetSubAction retrieve subaction with Trigger is `trigger`
// If there is no matched subaction, and empty Action{} is returned
func (act Action) GetSubAction(trigger string) Action {
	ret := act.subActionLookup[trigger]
	if ret == nil {
		return Action{}
	}
	return *ret
}

// Path returns the arguments needed to trigger this action
func (act Action) Path() string {
	if act.pathCached == "" {
		return act.Trigger
	}
	return act.pathCached
}

// EmptyTriggerError indicates an invalid Action which has empty Trigger string
type EmptyTriggerError struct {
	Err
}

func (EmptyTriggerError) Error() string {
	return "Action with empty Trigger is not allowed"
}

// ActionAlreadyAssginedError indicates adding an Action which belongs to an ActionTree as SubAction
type ActionAlreadyAssginedError struct {
	Err
	AssignedPath string
}

func (e ActionAlreadyAssginedError) Error() string {
	return fmt.Sprintf("Action already belongs to an ActionTree\nActionPath: %s", e.AssignedPath)
}

// DuplicatedSubActionError indicates attempting to add a SubAction with Trigger
// that is already in the sub action list
type DuplicatedSubActionError struct {
	Err
	Trigger string
}

func (e DuplicatedSubActionError) Error() string {
	return fmt.Sprintf("SubAction Already Exists, Trigger: %s", e.Trigger)
}

// AddSubAction append an SubAction to handle further triggering args
func (act *Action) AddSubAction(subAct Action) error {
	if subAct.Trigger == "" {
		return EmptyTriggerError{}
	}

	if subAct.parent != nil {
		return ActionAlreadyAssginedError{AssignedPath: subAct.Path()}
	}

	if act.subActionLookup == nil {
		act.subActionLookup = make(map[string]*Action)
	} else if _, ok := act.subActionLookup[subAct.Trigger]; ok {
		return DuplicatedSubActionError{Trigger: subAct.Trigger}
	}

	subAct.parent = act
	subAct.pathCached = subAct.parent.Path() + " " + subAct.Trigger
	act.subActionCopy = append(act.subActionCopy, subAct)
	act.subActionLookup[subAct.Trigger] = &subAct
	return nil
}

// ActionNotFinalizedError indicates Action APIs are called before Action is finalized
type ActionNotFinalizedError struct {
	Err
	Victim Action
}

func (e ActionNotFinalizedError) Error() string {
	str := fmt.Sprintf("Action Not Finalized\nActionPath: %s", (&e.Victim).Path())
	return str
}

// DoubleFinalizeError indicates attempting to Finalize an Action second time.
type DoubleFinalizeError struct {
	Err
	Victim Action
}

func (e DoubleFinalizeError) Error() string {
	str := fmt.Sprintf("Action Double Finalized\nActionPath: %s", (&e.Victim).Path())
	return str
}

func finalizeActionTree(act *Action, finalized []*Action) error {
	if act.finalized {
		return DoubleFinalizeError{Victim: *act}
	}

	if act.Trigger == "" {
		return EmptyTriggerError{}
	}

	// Normalize Min/MaxConsume settings
	if act.MinConsume < 0 {
		act.MinConsume = 0
	}

	if act.MaxConsume >= 0 && act.MaxConsume < act.MinConsume {
		act.MaxConsume = act.MinConsume
	}

	// Create empty lookupTable for leaf Actions to simplify parsing logics
	if act.subActionLookup == nil {
		act.subActionLookup = make(map[string]*Action)
	}

	act.finalized = true

	for _, nextAct := range act.subActionLookup {
		if err := finalizeActionTree(nextAct, finalized); err != nil {
			return err
		}
	}

	return nil
}

// Finalize should be called after Action tree is created before calling Parse()
// It initializes internal data for current Action and all SubActions for later Parse() calls
// Finalize should be called only once
// Do not attempt to modified any members of Actions in the Action tree after a Finalize() call
func (act *Action) Finalize() error {
	var finalized []*Action
	return finalizeActionTree(act, finalized)
}

// TooFewArgsError indicates an Action is triggered with few args then Action.MinConsume
type TooFewArgsError struct {
	Err
	Victim Action
	Args   []string
}

func (e TooFewArgsError) Error() string {
	return fmt.Sprintf("Parsing Error: Too Few Arguments: %s\nActionPath: %s",
		e.Args, (&e.Victim).Path())
}

// NilStateError indicates calling Action.Parse with state == nil
type NilStateError struct {
	Err
}

func (NilStateError) Error() string {
	return "Calling Parse() with state == nil"
}

// Parse args with current Action and all SubActions
// A state object needs to be provided to keep the states while visiting SubActions
// state is also used to retrieve string outputs from triggered SubActions
// optionally specified vargs will be forwarded to all Action.Do() calls
func (act Action) Parse(state *State, args []string, vargs ...interface{}) error {
	if !act.finalized {
		return ActionNotFinalizedError{Victim: act}
	}

	if len(args) == 0 {
		return nil
	}

	if state == nil {
		return NilStateError{}
	}

	if act.Trigger == args[0] {
		// Action is triggered
		// Consume args
		if len(args[1:]) < act.MinConsume {
			// Not enough arguments
			return TooFewArgsError{
				Victim: act,
				Args:   args[1:],
			}
		}

		if act.MaxConsume < 0 || len(args[1:]) <= act.MaxConsume {
			state.doArgs = args[1:]
			// all args are consumed
			if act.Do != nil {
				return act.Do(state, vargs...)
			}
			return nil
		}

		state.doArgs = args[1 : act.MaxConsume+1]
		args = args[act.MaxConsume+1:]
		if act.Do != nil {
			err := act.Do(state, vargs...)
			if err != nil {
				return err
			}
		}

		// Try to trigger SubActions with next arg
		if subAct, ok := act.subActionLookup[args[0]]; ok {
			return subAct.Parse(state, args, vargs)
		}

		return nil
	}

	return nil
}
