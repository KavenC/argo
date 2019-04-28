package argo

import (
	"fmt"
	"strings"
)

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

	// ArgNames optional slice of strings used as references for generating help text
	ArgNames []string

	// Hidden is true if this action should be hidden in help text
	Hidden bool

	// DisableHelp avoids auto injecting help SubAction for generating help text
	DisableHelp bool

	// HelpTrigger will be used as Trigger for the auto injected Help SubAction
	// If the string is not set (default), "help" will be used as Trigger
	HelpTrigger string

	// HelpGen is used to generate help text for this Action
	// If this is not set, it will be assigned as a default generator in Finalize()
	HelpGen func(Action) string

	parent              *Action
	pathCached          string
	subActionLookupTemp map[string]Action
	subActionLookup     map[string]*Action
	subActionTrigger    []string
	helpTextCached      string
	finalized           bool
}

// Help returns help text for this action
func (act *Action) Help() string {
	if act.helpTextCached == "" && act.HelpGen != nil {
		act.helpTextCached = act.HelpGen(*act)
	}
	return act.helpTextCached
}

// SubActions returns all immediate SubActions
func (act Action) SubActions() []string {
	return act.subActionTrigger
}

// GetSubAction retrieve subaction with Trigger is `trigger`
// If there is no matched subaction, and empty Action{} is returned
func (act Action) GetSubAction(trigger string) Action {
	if act.subActionLookup == nil {
		return act.subActionLookupTemp[trigger]
	}
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
	Path string
}

func (e EmptyTriggerError) Error() string {
	return fmt.Sprintf("Action with empty Trigger is not allowed. Path: %s", e.Path)
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

// UnreachableActionError indicates an Action will never be reached due to its parent consumed all args
type UnreachableActionError struct {
	Err
	Path string
}

func (e UnreachableActionError) Error() string {
	return fmt.Sprintf("Action is unreachable: %s", e.Path)
}

// AddSubAction append an SubAction to handle further triggering args
func (act *Action) AddSubAction(subAct Action) error {
	if subAct.Trigger == "" {
		return EmptyTriggerError{}
	}

	if subAct.parent != nil {
		return ActionAlreadyAssginedError{AssignedPath: subAct.Path()}
	}

	if act.MaxConsume < 0 {
		return UnreachableActionError{Path: act.Path() + " " + subAct.Trigger}
	}

	if act.subActionLookupTemp == nil {
		act.subActionLookupTemp = make(map[string]Action)
	} else if _, ok := act.subActionLookupTemp[subAct.Trigger]; ok {
		return DuplicatedSubActionError{Trigger: subAct.Trigger}
	}

	subAct.parent = act
	subAct.pathCached = subAct.parent.Path() + " " + subAct.Trigger
	act.subActionTrigger = append(act.subActionTrigger, subAct.Trigger)
	act.subActionLookupTemp[subAct.Trigger] = subAct
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

func defaultHelpGenerator(act Action) string {
	text := strings.Builder{}

	text.WriteString("[Usage]\n")
	genUsage := func(act Action) string {
		text := strings.Builder{}
		text.WriteString(act.Path())

		if act.MaxConsume != 0 {
			argNum := 0
			if act.MaxConsume > 0 {
				argNum = act.MaxConsume
			} else {
				argNum = act.MinConsume
			}

			requiredArgs := make([]string, argNum)
			if len(act.ArgNames) > 0 {
				copy(requiredArgs, act.ArgNames)
			}

			for index, arg := range requiredArgs[:act.MinConsume] {
				if arg == "" {
					text.WriteString(fmt.Sprintf(" <%s%d>", "arg", index+1))
				} else {
					text.WriteString(fmt.Sprintf(" <%s>", arg))
				}
			}

			if act.MaxConsume < 0 {
				if len(act.ArgNames) > act.MinConsume {
					text.WriteString(fmt.Sprintf(" [%s ...]", act.ArgNames[act.MinConsume]))
				} else {
					text.WriteString(" [argN ...]")
				}
			} else {
				if act.MaxConsume > act.MinConsume {
					text.WriteString(" [")
					argText := strings.Builder{}
					for index, arg := range requiredArgs[act.MinConsume:] {
						if arg == "" {
							argText.WriteString(fmt.Sprintf("%s%d ", "arg", index+act.MinConsume+1))
						} else {
							argText.WriteString(fmt.Sprintf("%s ", arg))
						}
					}
					text.WriteString(strings.TrimSpace(argText.String()))
					text.WriteString("]")
				}
			}
		} else {
			text.WriteString(" [sub-action]")
		}

		return text.String()
	}
	text.WriteString(genUsage(act))

	if act.LongDescr != "" {
		text.WriteString("\n\n[Description]\n")
		text.WriteString(fmt.Sprint(act.LongDescr))
	} else if act.ShortDescr != "" {
		text.WriteString("\n\n[Description]\n")
		text.WriteString(fmt.Sprint(act.ShortDescr))
	}

	subAct := act.SubActions()
	if len(subAct) != 0 {
		text.WriteString("\n\n[Sub-actions]")
		for _, sub := range subAct {
			subAct := act.GetSubAction(sub)
			text.WriteString(fmt.Sprintf("\n%s\n- %s", subAct.Trigger, subAct.ShortDescr))
		}
	}

	return text.String()
}

func finalizeActionTree(parent *Action, act *Action) error {
	if act.finalized {
		return DoubleFinalizeError{Victim: *act}
	}

	if act.Trigger == "" {
		return EmptyTriggerError{Path: act.Path()}
	}

	// Retarget parent
	act.parent = parent

	// Normalize Min/MaxConsume settings
	if act.MinConsume < 0 {
		act.MinConsume = 0
	}

	if act.MaxConsume >= 0 && act.MaxConsume < act.MinConsume {
		act.MaxConsume = act.MinConsume
	}

	// Setup Path
	if act.parent == nil {
		act.pathCached = act.Trigger
	} else {
		act.pathCached = act.parent.Path() + " " + act.Trigger
	}

	// Setup Help text
	if act.HelpGen == nil {
		if act.parent == nil {
			act.HelpGen = defaultHelpGenerator
		} else {
			act.HelpGen = act.parent.HelpGen
		}
	}

	// Inject help SubAction
	if act.HelpTrigger == "" {
		if act.parent == nil {
			act.HelpTrigger = "help"
		} else {
			act.HelpTrigger = act.parent.HelpTrigger
		}
	}

	if !act.DisableHelp && act.MaxConsume == 0 {
		err := act.AddSubAction(Action{
			Trigger:    act.HelpTrigger,
			MaxConsume: 1,
			Do: func(state *State, _ ...interface{}) error {
				args := state.Args()
				if len(args) > 0 {
					cmd := args[0]
					targetAct := act.GetSubAction(cmd)
					if targetAct.Trigger == "" {
						fmt.Fprintf(&state.OutputStr, "Sub action not found: %s %s", act.Path(), cmd)
					} else {
						state.OutputStr.WriteString(targetAct.Help())
					}
				} else {
					state.OutputStr.WriteString(act.Help())
				}
				return nil
			},
			ShortDescr:  "Display help for commands",
			DisableHelp: true,
		})

		if err != nil {
			_, helpExists := err.(DuplicatedSubActionError)
			if !helpExists {
				return err // should not reach
			}
		}
	}

	// Create lookupTable
	act.subActionLookup = make(map[string]*Action)
	for subTrigger, subAct := range act.subActionLookupTemp {
		tempAct := subAct
		act.subActionLookup[subTrigger] = &tempAct
	}

	act.finalized = true

	for _, subAct := range act.subActionLookup {
		if err := finalizeActionTree(act, subAct); err != nil {
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
	return finalizeActionTree(nil, act)
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
			return subAct.Parse(state, args, vargs...)
		}

		return nil
	}

	return nil
}
