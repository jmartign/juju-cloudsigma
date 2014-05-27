// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package state_test

import (
	jc "github.com/juju/testing/checkers"
	gc "launchpad.net/gocheck"

	"launchpad.net/juju-core/state"
)

type ActionSuite struct {
	ConnSuite
	charm   *state.Charm
	service *state.Service
	unit    *state.Unit
}

var _ = gc.Suite(&ActionSuite{})

func (s *ActionSuite) SetUpTest(c *gc.C) {
	s.ConnSuite.SetUpTest(c)
	s.charm = s.AddTestingCharm(c, "wordpress")
	var err error
	s.service = s.AddTestingService(c, "wordpress", s.charm)
	c.Assert(err, gc.IsNil)
	s.unit, err = s.service.AddUnit()
	c.Assert(err, gc.IsNil)
	c.Assert(s.unit.Series(), gc.Equals, "quantal")
}

func (s *ActionSuite) TestAddAction(c *gc.C) {
	actionName := "fakeaction"
	actionParams := map[string]interface{}{"outfile": "outfile.tar.bz2"}

	// verify can add an Action
	actionId, err := s.unit.AddAction(actionName, actionParams)
	c.Assert(err, gc.IsNil)
	assertSaneActionId(c, actionId, s.unit.Name())

	// verify we can get it back out by Id
	action, err := s.State.Action(actionId)
	c.Assert(err, gc.IsNil)
	c.Assert(action, gc.NotNil)
	c.Assert(action.Id(), gc.Equals, actionId)

	// verify we get out what we put in
	c.Assert(action.Name(), gc.Equals, actionName)
	c.Assert(action.Payload(), jc.DeepEquals, actionParams)
}

func (s *ActionSuite) TestAddActionLifecycle(c *gc.C) {
	unit, err := s.State.Unit(s.unit.Name())
	c.Assert(err, gc.IsNil)
	preventUnitDestroyRemove(c, unit)

	// make unit state Dying
	err = unit.Destroy()
	c.Assert(err, gc.IsNil)

	// can add action to a dying unit
	actionId, err := unit.AddAction("fakeaction1", map[string]interface{}{})
	c.Assert(err, gc.IsNil)
	assertSaneActionId(c, actionId, s.unit.Name())

	// make sure unit is dead
	err = unit.EnsureDead()
	c.Assert(err, gc.IsNil)

	// cannot add action to a dead unit
	_, err = unit.AddAction("fakeaction2", map[string]interface{}{})
	c.Assert(err, gc.ErrorMatches, "unit .* is dead")
}

func (s *ActionSuite) TestAddActionFailsOnDeadUnitInTransaction(c *gc.C) {
	unit, err := s.State.Unit(s.unit.Name())
	c.Assert(err, gc.IsNil)
	preventUnitDestroyRemove(c, unit)

	killUnit := state.TransactionHook{
		Before: func() {
			c.Assert(unit.Destroy(), gc.IsNil)
			c.Assert(unit.EnsureDead(), gc.IsNil)
		},
	}
	defer state.SetTransactionHooks(c, s.State, killUnit).Check()

	_, err = unit.AddAction("fakeaction", map[string]interface{}{})
	c.Assert(err, gc.ErrorMatches, "unit .* is dead")
}

// assertSaneActionId verifies that the actionId is of the expected
// form (unit id prefix + sequence)
// This is a temporary assertion, we shouldn't be leaking the actual
// mongo _id
func assertSaneActionId(c *gc.C, actionId, unitName string) {
	c.Assert(actionId, gc.Matches, "^u#"+unitName+"#\\d+")
}