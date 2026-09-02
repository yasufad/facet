package app

import (
	"testing"
)

type counter struct {
	count int
}

func newCounter(t *testing.T, app *App, start int) Entity[counter] {
	t.Helper()
	return New(app, func(cx *Context[counter]) counter {
		return counter{count: start}
	})
}

func TestNewAndRead(t *testing.T) {
	app := NewApp()
	defer app.Close()

	c := newCounter(t, app, 3)
	if got := c.Read(app).count; got != 3 {
		t.Fatalf("got %d, want 3", got)
	}
	c.Release()
}

func TestUpdateMutatesAndFlushes(t *testing.T) {
	app := NewApp()
	defer app.Close()

	c := newCounter(t, app, 0)
	c.Update(app, func(v *counter, cx *Context[counter]) {
		v.count++
		cx.Notify()
	})
	if got := c.Read(app).count; got != 1 {
		t.Fatalf("got %d, want 1", got)
	}
	c.Release()
}

func TestReentrantUpdatePanics(t *testing.T) {
	app := NewApp()
	defer app.Close()

	c := newCounter(t, app, 0)
	defer func() {
		if recover() == nil {
			t.Fatal("re-entrant update did not panic")
		}
		// c is left in a leased state by the panic; release to avoid noise.
		c.Release()
	}()
	c.Update(app, func(v *counter, cx *Context[counter]) {
		// Updating the same entity while it is on lease must panic.
		c.Update(app, func(v *counter, cx *Context[counter]) {})
	})
}

func TestWeakHandleUpgradeAndDrop(t *testing.T) {
	app := NewApp()
	defer app.Close()

	c := newCounter(t, app, 7)
	w := c.Downgrade()
	up, ok := w.Upgrade()
	if !ok {
		t.Fatal("weak handle should upgrade while strong handle is alive")
	}
	up.Release() // Upgrade returns an owning handle; release it.
	c.Release()
	app.Flush()
	if _, ok := w.Upgrade(); ok {
		t.Fatal("weak handle should not upgrade after the strong handle is released and reaped")
	}
}

func TestRefcountCloneRelease(t *testing.T) {
	app := NewApp()
	defer app.Close()

	c := newCounter(t, app, 0)
	c2 := c.Clone()
	// Releasing one strong handle must not drop the entity while another lives.
	c.Release()
	if got := c2.Read(app).count; got != 0 {
		t.Fatalf("got %d, want 0", got)
	}
	c2.Release()
}

func TestUpdatePointerAliasesStoredValue(t *testing.T) {
	// The pointer UpdateEntity hands to f is the one stored in the entity map,
	// not the address of a leased copy. A closure that captures it and writes
	// after the update that captured it has returned (the shape of an event
	// handler registered during Render and invoked later from dispatch) must
	// still land its write, and Read must observe it without an intervening
	// update.
	app := NewApp()
	defer app.Close()

	c := newCounter(t, app, 0)
	var captured *counter
	c.Update(app, func(v *counter, cx *Context[counter]) {
		captured = v
	})

	captured.count = 42
	if got := c.Read(app).count; got != 42 {
		t.Fatalf("got %d, want 42: write through a pointer captured during a prior update was not visible", got)
	}
	c.Release()
}

func TestOnReleaseFiresOnDrop(t *testing.T) {
	app := NewApp()
	defer app.Close()

	c := newCounter(t, app, 0)
	fired := false
	UpdateEntity(app, c, func(v *counter, cx *Context[counter]) {
		cx.OnRelease(func(v *counter, app *App) {
			fired = true
		})
	})
	c.Release()
	app.Flush()
	if !fired {
		t.Fatal("OnRelease callback did not fire when the entity was dropped")
	}
}

func TestOnReleaseCascadesHandleDrop(t *testing.T) {
	// An entity that owns a strong handle to another should release it in its
	// OnRelease, so dropping the owner cascades to the owned.
	app := NewApp()
	defer app.Close()

	owned := newCounter(t, app, 0)
	ownedWeak := owned.Downgrade()

	type owner struct {
		held Entity[counter]
	}
	o := New(app, func(cx *Context[owner]) owner {
		return owner{held: owned.Clone()}
	})
	// The owner took its own owning handle; release the original.
	owned.Release()

	UpdateEntity(app, o, func(v *owner, cx *Context[owner]) {
		cx.OnRelease(func(v *owner, app *App) {
			v.held.Release()
		})
	})

	o.Release()
	app.Flush()
	if _, ok := ownedWeak.Upgrade(); ok {
		t.Fatal("owned entity should have been dropped when its owner was dropped")
	}
}
