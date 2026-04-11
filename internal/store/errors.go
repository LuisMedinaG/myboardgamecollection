package store

import "errors"

// ErrDuplicate is returned by store methods when a unique-constraint violation
// occurs (e.g. duplicate vibe name or username). Callers should use errors.Is
// instead of matching the underlying driver error string.
var ErrDuplicate = errors.New("already exists")

// ErrWrongPassword is returned by ChangePassword when the supplied current
// password does not match the stored hash.
var ErrWrongPassword = errors.New("current password is incorrect")
