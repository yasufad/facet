package input

import "testing"

type testMoveUp struct{}

func (testMoveUp) ActionName() string { return "editor::MoveUp" }

type testMove struct {
	Direction string
	Select    bool
}

func (testMove) ActionName() string { return "editor::Move" }

func TestActions(t *testing.T) {
	up1 := testMoveUp{}
	up2 := testMoveUp{}
	move1 := testMove{Direction: "up", Select: false}
	move2 := testMove{Direction: "up", Select: false}
	move3 := testMove{Direction: "down", Select: true}

	if up1.ActionName() != "editor::MoveUp" {
		t.Fatalf("unexpected name: got %q, want %q", up1.ActionName(), "editor::MoveUp")
	}

	if !ActionsEqual(up1, up2) {
		t.Fatalf("expected up1 == up2")
	}
	if !ActionsEqual(move1, move2) {
		t.Fatalf("expected move1 == move2")
	}
	if ActionsEqual(move1, move3) {
		t.Fatalf("expected move1 != move3")
	}
	if ActionsEqual(up1, move1) {
		t.Fatalf("expected up1 != move1")
	}
}

func TestNoActionAndUnbind(t *testing.T) {
	var noAction Action = NoAction{}
	if noAction.ActionName() != "NoAction" {
		t.Fatalf("unexpected name: %q", noAction.ActionName())
	}
	if !IsNoAction(noAction) {
		t.Fatalf("expected IsNoAction to be true")
	}
	if IsNoAction(nil) {
		t.Fatalf("expected IsNoAction(nil) to be false")
	}

	var unbind Action = Unbind{TargetAction: "editor::MoveUp"}
	if unbind.ActionName() != "Unbind" {
		t.Fatalf("unexpected name: %q", unbind.ActionName())
	}
	if !IsUnbind(unbind) {
		t.Fatalf("expected IsUnbind to be true")
	}
	if IsUnbind(nil) {
		t.Fatalf("expected IsUnbind(nil) to be false")
	}
	if IsNoAction(unbind) {
		t.Fatalf("expected IsNoAction(unbind) to be false")
	}
}
