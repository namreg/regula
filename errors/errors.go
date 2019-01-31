package errors

import "errors"

var (
	// ErrTypeMismatch is returned when the evaluated rule doesn't return the expected result type.
	ErrTypeMismatch = errors.New("type returned by rule doesn't match")

	// ErrRulesetNotFound must be returned when no ruleset is found for a given key.
	ErrRulesetNotFound = errors.New("ruleset not found")

	// ErrRulesetIncoherentType is returned when a ruleset contains rules of different types.
	ErrRulesetIncoherentType = errors.New("types in ruleset are incoherent")

	// ErrParamTypeMismatch is returned when a parameter type is different from expected.
	ErrParamTypeMismatch = errors.New("parameter type mismatches")

	// ErrSignatureMismatch is returned when a rule doesn't respect a given signature.
	ErrSignatureMismatch = errors.New("signature mismatch")

	// ErrParamNotFound is returned when a parameter is not defined.
	ErrParamNotFound = errors.New("parameter not found")

	// ErrNoMatch is returned when the rule doesn't match the given params.
	ErrNoMatch = errors.New("rule doesn't match the given params")
)
