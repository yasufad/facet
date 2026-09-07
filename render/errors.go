package render

import "errors"

// ErrNoAdapter indicates that device creation failed because the machine
// has no compatible graphics adapter — either no adapter is present at all,
// or the available adapter does not support the backend's required feature
// set.
//
// A test that hits this error cannot run in the environment it was given,
// which is a skip, not a failure. Callers that construct a [Renderer]
// branch on it with [errors.Is] to distinguish "cannot run here" from
// "ran and got the wrong answer".
var ErrNoAdapter = errors.New("render: no compatible graphics adapter")
