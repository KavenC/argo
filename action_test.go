package argo

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func checkEq(t *testing.T, target interface{}, expected interface{}) {
	if !reflect.DeepEqual(target, expected) {
		t.Logf("%s (Expected: %s)", target, expected)
		t.FailNow()
	}
}

func checkNe(t *testing.T, target interface{}, expected interface{}) {
	if reflect.DeepEqual(target, expected) {
		t.Logf("Expected not to be: %s", expected)
		t.FailNow()
	}
}

func checkTypeEq(t *testing.T, target interface{}, expected interface{}) {
	typeTarget := reflect.TypeOf(target)
	typeExpected := reflect.TypeOf(expected)
	checkEq(t, typeTarget, typeExpected)
}

func TestTrigger(t *testing.T) {
	act := Action{
		Trigger: "test",
		Do: func(state *State, _ ...interface{}) error {
			state.OutputStr = "called"
			return nil
		},
	}
	act.Finalize()
	state := &State{}
	act.Parse(state, []string{"test"})

	checkEq(t, state.OutputStr, "called")
}

func TestSubTrigger(t *testing.T) {
	act := Action{
		Trigger: "test",
		Do: func(state *State, _ ...interface{}) error {
			state.OutputStr = "called"
			return nil
		},
	}

	subAct := Action{
		Trigger: "sub",
		Do: func(state *State, _ ...interface{}) error {
			state.OutputStr += " sub"
			return nil
		},
	}

	act.AddSubAction(subAct)
	act.Finalize()

	state := &State{}
	act.Parse(state, []string{"test", "sub"})

	checkEq(t, state.OutputStr, "called sub")
}

func TestPath(t *testing.T) {
	checkEq(t, Action{}.Path(), "")
	checkEq(t, Action{Trigger: "test"}.Path(), "test")
	// TODO
}

func TestConsumeMin(t *testing.T) {
	act := Action{
		Trigger:    "test",
		MinConsume: 2,
		Do: func(state *State, _ ...interface{}) error {
			args := state.Args()
			if args[0] != "arg1" || args[1] != "arg2" || len(args) != 2 {
				state.OutputStr = "failed"
			} else {
				state.OutputStr = "called"
			}
			return nil
		},
	}
	act.Finalize()
	state := &State{}
	act.Parse(state, []string{"test", "arg1", "arg2", "arg3"})

	checkEq(t, state.OutputStr, "called")
}

func TestConsumeMinMax(t *testing.T) {
	act := Action{
		Trigger:    "test",
		MinConsume: 2,
		MaxConsume: 4,
		Do: func(state *State, _ ...interface{}) error {
			args := state.Args()
			if args[0] != "arg1" || args[1] != "arg2" || args[2] != "arg3" || len(args) != 3 {
				state.OutputStr = "failed"
			} else {
				state.OutputStr = "called"
			}
			return nil
		},
	}
	act.Finalize()
	state := &State{}
	act.Parse(state, []string{"test", "arg1", "arg2", "arg3"})

	checkEq(t, state.OutputStr, "called")
}

func TestConsumeMax(t *testing.T) {
	act := Action{
		Trigger:    "test",
		MaxConsume: 2,
		Do: func(state *State, _ ...interface{}) error {
			args := state.Args()
			if args[0] != "arg1" || args[1] != "arg2" || len(args) != 2 {
				state.OutputStr = "failed"
			} else {
				state.OutputStr = "called"
			}
			return nil
		},
	}
	act.Finalize()
	state := &State{}
	act.Parse(state, []string{"test", "arg1", "arg2", "arg3"})

	checkEq(t, state.OutputStr, "called")
}

func TestConsumeAll(t *testing.T) {
	act := Action{
		Trigger:    "test",
		MaxConsume: -1,
		Do: func(state *State, _ ...interface{}) error {
			args := state.Args()
			if args[0] != "arg1" || args[1] != "arg2" || args[2] != "arg3" || len(args) != 3 {
				state.OutputStr = "failed"
			} else {
				state.OutputStr = "called"
			}
			return nil
		},
	}
	err := act.Finalize()
	checkEq(t, err, nil)
	state := &State{}
	err = act.Parse(state, []string{"test", "arg1", "arg2", "arg3"})
	checkEq(t, err, nil)

	checkEq(t, state.OutputStr, "called")
}

func TestConsumeNormalize(t *testing.T) {
	act := Action{
		Trigger:    "test",
		MinConsume: -1,
		Do: func(state *State, _ ...interface{}) error {
			args := state.Args()
			if len(args) != 0 {
				state.OutputStr = "failed"
			} else {
				state.OutputStr = "called"
			}
			return nil
		},
	}
	err := act.Finalize()
	checkEq(t, err, nil)
	state := &State{}
	err = act.Parse(state, []string{"test", "arg1", "arg2", "arg3"})
	checkEq(t, err, nil)

	checkEq(t, state.OutputStr, "called")
}

func TestConsumeThenTrigger(t *testing.T) {
	act := Action{
		Trigger:    "test",
		MinConsume: 2,
		Do: func(state *State, _ ...interface{}) error {
			args := state.Args()
			if args[0] != "arg1" || args[1] != "arg2" || len(args) != 2 {
				state.OutputStr = "failed"
			} else {
				state.OutputStr = "called"
			}
			return nil
		},
	}

	subAct := Action{
		Trigger: "arg1",
		Do: func(state *State, _ ...interface{}) error {
			state.OutputStr += " sub"
			return nil
		},
	}

	act.AddSubAction(subAct)

	act.Finalize()
	state := &State{}
	act.Parse(state, []string{"test", "arg1", "arg2", "arg1"})

	checkEq(t, state.OutputStr, "called sub")
}

func TestConsumeNotTrigger(t *testing.T) {
	act := Action{
		Trigger:    "test",
		MinConsume: 2,
		MaxConsume: 3,
		Do: func(state *State, _ ...interface{}) error {
			args := state.Args()
			if args[0] != "arg1" || args[1] != "arg2" || args[2] != "arg1" || len(args) != 3 {
				state.OutputStr = "failed"
			} else {
				state.OutputStr = "called"
			}
			return nil
		},
	}

	subAct := Action{
		Trigger: "arg1",
		Do: func(state *State, _ ...interface{}) error {
			state.OutputStr += " sub"
			return nil
		},
	}

	act.AddSubAction(subAct)

	act.Finalize()
	state := &State{}
	act.Parse(state, []string{"test", "arg1", "arg2", "arg1"})

	checkEq(t, state.OutputStr, "called")
}

func TestNoConsume(t *testing.T) {
	act := Action{
		Trigger:    "test",
		MinConsume: 2,
		Do: func(state *State, _ ...interface{}) error {
			args := state.Args()
			if args[0] != "arg1" || args[1] != "arg2" || len(args) != 2 {
				state.OutputStr = "failed"
			} else {
				state.OutputStr = "called"
			}
			return nil
		},
	}

	subAct := Action{
		Trigger:    "arg1",
		MaxConsume: 1,
		Do: func(state *State, _ ...interface{}) error {
			args := state.Args()
			if len(args) != 0 {
				state.OutputStr += "failed"
			} else {
				state.OutputStr += " sub"
			}
			return nil
		},
	}

	act.AddSubAction(subAct)

	act.Finalize()
	state := &State{}
	act.Parse(state, []string{"test", "arg1", "arg2", "arg1"})

	checkEq(t, state.OutputStr, "called sub")
}

func TestCommonChildren(t *testing.T) {
	root := Action{
		Trigger: "root",
		Do: func(state *State, _ ...interface{}) error {
			state.OutputStr = "root"
			return nil
		},
	}

	sub1 := Action{
		Trigger: "sub1",
		Do: func(state *State, _ ...interface{}) error {
			state.OutputStr += " sub1"
			return nil
		},
	}

	sub2 := Action{
		Trigger: "sub2",
		Do: func(state *State, _ ...interface{}) error {
			state.OutputStr += " sub2"
			return nil
		},
	}

	common := Action{
		Trigger: "common",
		Do: func(state *State, _ ...interface{}) error {
			state.OutputStr += " common"
			return nil
		},
	}

	sub1.AddSubAction(common)
	sub2.AddSubAction(common)
	root.AddSubAction(sub1)
	root.AddSubAction(sub2)

	err := root.Finalize()
	checkEq(t, err, nil)

	state := &State{}
	err = root.Parse(state, []string{"root", "sub1", "common"})
	checkEq(t, err, nil)
	checkEq(t, state.OutputStr, "root sub1 common")

	err = root.Parse(state, []string{"root", "sub2", "common"})
	checkEq(t, err, nil)
	checkEq(t, state.OutputStr, "root sub2 common")
}

func TestEmptyTriggerError(t *testing.T) {
	act := Action{
		Do: func(state *State, _ ...interface{}) error {
			return nil
		},
	}

	err := act.Finalize()
	checkTypeEq(t, err, EmptyTriggerError{})
	argoErr, _ := err.(EmptyTriggerError)
	checkEq(t, strings.Contains(argoErr.Error(), "empty Trigger"), true)
}

func TestSubEmptyTriggerError(t *testing.T) {
	act := Action{
		Trigger: "act",
		Do: func(state *State, _ ...interface{}) error {
			return nil
		},
	}
	err := act.AddSubAction(Action{})
	checkTypeEq(t, err, EmptyTriggerError{})
}

func TestDuplicatedSubActionError(t *testing.T) {
	root := Action{
		Trigger: "root",
		Do: func(state *State, _ ...interface{}) error {
			state.OutputStr = "root"
			return nil
		},
	}

	sub1 := Action{
		Trigger: "sub1",
		Do: func(state *State, _ ...interface{}) error {
			state.OutputStr += " sub1"
			return nil
		},
	}

	sub2 := Action{
		Trigger: "sub1",
		Do: func(state *State, _ ...interface{}) error {
			state.OutputStr += " sub1"
			return nil
		},
	}

	root.AddSubAction(sub1)
	err := root.AddSubAction(sub2)
	checkNe(t, err, nil)
	argoErr, ok := err.(DuplicatedSubActionError)
	checkEq(t, ok, true)
	checkEq(t, argoErr.Trigger, "sub1")
	checkEq(t, strings.Contains(argoErr.Error(), argoErr.Trigger), true)
}

func TestDoubleFinalizeError(t *testing.T) {
	act := Action{
		Trigger: "arg",
	}
	act.AddSubAction(Action{Trigger: "sub"})

	err := act.Finalize()
	checkEq(t, err, nil)
	err = act.Finalize()
	argoErr, ok := err.(DoubleFinalizeError)
	checkEq(t, ok, true)
	checkEq(t, argoErr.Victim.Trigger, "arg")
	checkEq(t, strings.Contains(argoErr.Error(), argoErr.Victim.Path()), true)
}

func TestActionNotFinalizedError(t *testing.T) {
	act := Action{
		Trigger: "arg",
	}
	act.AddSubAction(Action{Trigger: "sub"})

	state := &State{}
	err := act.Parse(state, []string{"arg", "sub"})

	argoErr, ok := err.(ActionNotFinalizedError)
	checkEq(t, ok, true)
	checkEq(t, argoErr.Victim.Trigger, "arg")
	checkEq(t, strings.Contains(argoErr.Error(), argoErr.Victim.Path()), true)
}

func TestNilStateError(t *testing.T) {
	act := Action{
		Trigger: "arg",
	}
	act.AddSubAction(Action{Trigger: "sub"})

	err := act.Finalize()
	checkEq(t, err, nil)
	err = act.Parse(nil, []string{"arg", "sub"})
	checkTypeEq(t, err, NilStateError{})
	argoErr, _ := err.(NilStateError)
	checkEq(t, strings.Contains(argoErr.Error(), "nil"), true)
}

func TestTooFewArgsError(t *testing.T) {
	act := Action{
		Trigger:    "arg",
		MinConsume: 2,
	}

	err := act.Finalize()
	checkEq(t, err, nil)

	state := &State{}
	err = act.Parse(state, []string{"arg", "sub"})
	argoErr, ok := err.(TooFewArgsError)
	checkEq(t, ok, true)
	checkEq(t, argoErr.Victim.Trigger, "arg")
	checkEq(t, argoErr.Args, []string{"sub"})
	checkEq(t, strings.Contains(argoErr.Error(), argoErr.Victim.Path()), true)
}

type CustomError struct {
}

func (CustomError) Error() string {
	return "cerr"
}

func TestDoReturnsError(t *testing.T) {
	act := Action{
		Trigger: "test",
		Do: func(_ *State, _ ...interface{}) error {
			return CustomError{}
		},
	}

	act.Finalize()
	err := act.Parse(&State{}, []string{"test"})
	_, ok := err.(CustomError)
	checkEq(t, ok, true)
}

func TestConsumeAndReturnsError(t *testing.T) {
	act := Action{
		Trigger:    "test",
		MaxConsume: 1,
		Do: func(_ *State, _ ...interface{}) error {
			return CustomError{}
		},
	}

	act.Finalize()
	err := act.Parse(&State{}, []string{"test", "arg", "arg2"})
	_, ok := err.(CustomError)
	checkEq(t, ok, true)
}

func TestParseVargs(t *testing.T) {
	act := Action{
		Trigger: "test",
		Do: func(_ *State, vargs ...interface{}) error {
			if len(vargs) != 1 {
				return errors.New("error")
			}

			v, ok := vargs[0].(int)
			if !ok || v != 9527 {
				return errors.New("error")
			}

			return nil
		},
	}

	act.Finalize()
	err := act.Parse(&State{}, []string{"test"}, 9527)
	checkEq(t, err, nil)
}

func checkSubActions(t *testing.T, target []Action, check []string) {
	checkEq(t, len(target), len(check))
	for index, act := range target {
		checkEq(t, act.Trigger, check[index])
	}
}

func TestSubActions(t *testing.T) {
	root := Action{Trigger: "root"}
	sub2 := Action{Trigger: "sub2"}
	sub2.AddSubAction(Action{Trigger: "subsub1"})
	sub2.AddSubAction(Action{Trigger: "subsub2"})
	root.AddSubAction(Action{Trigger: "sub1"})
	root.AddSubAction(sub2)

	checkSubActions(t, root.SubActions(), []string{"sub1", "sub2"})
	checkSubActions(t,
		root.GetSubAction("sub2").SubActions(),
		[]string{"subsub1", "subsub2"})
}

func TestGetSubAction(t *testing.T) {
	root := Action{Trigger: "root"}
	sub2 := Action{Trigger: "sub2"}
	sub2.AddSubAction(Action{Trigger: "subsub1"})
	sub2.AddSubAction(Action{Trigger: "subsub2"})
	root.AddSubAction(Action{Trigger: "sub1"})
	root.AddSubAction(sub2)

	checkEq(t, root.GetSubAction("sub1").Trigger, "sub1")
	checkEq(t, root.GetSubAction("sub2").GetSubAction("subsub1").Trigger, "subsub1")
	checkEq(t, root.GetSubAction("none").Trigger, "")
	checkEq(t, root.GetSubAction("sub1").GetSubAction("none").Trigger, "")
}

// Corner cases to fill-up coverage
func TestActionAlreadyAssignedError(t *testing.T) {
	act := Action{
		Trigger: "arg",
	}

	sub := Action{
		Trigger:    "sub",
		MinConsume: 1,
	}

	act.AddSubAction(sub)

	err := act.Finalize()
	checkEq(t, err, nil)

	state := &State{}
	err = act.Parse(state, []string{"arg", "sub"})
	argoErr, ok := err.(TooFewArgsError)
	checkEq(t, ok, true)

	new := Action{Trigger: "new"}
	err = new.AddSubAction(argoErr.Victim)
	newArgoErr, ok := err.(ActionAlreadyAssginedError)
	checkEq(t, ok, true)
	checkEq(t, newArgoErr.AssignedPath, "arg sub")
	checkEq(t, strings.Contains(newArgoErr.Error(), argoErr.Victim.Path()), true)
}

func TestSubErrorInFinalize(t *testing.T) {
	act := Action{
		Trigger:    "arg",
		MinConsume: 1,
	}

	err := act.Finalize()
	checkEq(t, err, nil)

	state := &State{}
	err = act.Parse(state, []string{"arg"})
	argoErr, ok := err.(TooFewArgsError)
	checkEq(t, ok, true)

	new := Action{Trigger: "new"}
	err = new.AddSubAction(argoErr.Victim)
	checkEq(t, err, nil)
	err = new.Finalize()
	newArgoErr, ok := err.(DoubleFinalizeError)
	checkEq(t, ok, true)
	checkEq(t, newArgoErr.Victim.Trigger, "arg")
}

func TestParseWithEmptyArgs(t *testing.T) {
	act := Action{
		Trigger: "test",
		Do: func(state *State, _ ...interface{}) error {
			state.OutputStr = "called"
			return nil
		},
	}
	err := act.Finalize()
	checkEq(t, err, nil)

	state := &State{}
	err = act.Parse(state, []string{})
	checkEq(t, err, nil)
	checkEq(t, state.OutputStr, "")
}

func TestConsumeAllButDoNothing(t *testing.T) {
	act := Action{
		Trigger:    "test",
		MaxConsume: -1,
	}
	err := act.Finalize()
	checkEq(t, err, nil)

	state := &State{}
	err = act.Parse(state, []string{"test", "arg", "arg", "arg"})
	checkEq(t, err, nil)
}

func TestNothingIsTriggered(t *testing.T) {
	act := Action{
		Trigger:    "test",
		MaxConsume: -1,
	}
	act.AddSubAction(Action{Trigger: "arg1"})
	err := act.Finalize()
	checkEq(t, err, nil)

	state := &State{}
	err = act.Parse(state, []string{"test1", "arg", "arg", "arg"})
	checkEq(t, err, nil)
}