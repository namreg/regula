package regula

import (
	"encoding/json"
	"testing"

	"github.com/heetch/regula/errors"
	"github.com/heetch/regula/rule"
	"github.com/stretchr/testify/require"
)

func TestRulesetEval(t *testing.T) {
	t.Run("Match string", func(t *testing.T) {
		r := NewRuleset(
			rule.New(rule.Eq(rule.StringParam("foo"), rule.StringValue("baz")), rule.StringValue("first")),
			rule.New(rule.Eq(rule.StringParam("foo"), rule.StringValue("bar")), rule.StringValue("second")),
		)

		res, err := r.Eval(Params{
			"foo": "bar",
		})
		require.NoError(t, err)
		require.Equal(t, "second", res.Data)
	})

	t.Run("Match bool", func(t *testing.T) {
		r := NewRuleset(
			rule.New(rule.Eq(rule.StringValue("foo"), rule.StringValue("bar")), rule.BoolValue(false)),
			rule.New(rule.Eq(rule.StringValue("foo"), rule.StringValue("foo")), rule.BoolValue(true)),
		)

		res, err := r.Eval(nil)
		require.NoError(t, err)
		require.Equal(t, "true", res.Data)
	})

	t.Run("No match", func(t *testing.T) {
		r := NewRuleset(
			rule.New(rule.Eq(rule.StringValue("foo"), rule.StringValue("bar")), rule.StringValue("first")),
			rule.New(rule.Eq(rule.StringValue("bar"), rule.StringValue("foo")), rule.StringValue("second")),
		)

		_, err := r.Eval(nil)
		require.Equal(t, errors.ErrNoMatch, err)
	})

	t.Run("Default", func(t *testing.T) {
		r := NewRuleset(
			rule.New(rule.Eq(rule.StringValue("foo"), rule.StringValue("bar")), rule.StringValue("first")),
			rule.New(rule.Eq(rule.StringValue("bar"), rule.StringValue("foo")), rule.StringValue("second")),
			rule.New(rule.True(), rule.StringValue("default")),
		)

		res, err := r.Eval(nil)
		require.NoError(t, err)
		require.Equal(t, "default", res.Data)
	})
}

func TestRulesetEncDec(t *testing.T) {
	r1 := NewRuleset(
		rule.New(rule.Eq(rule.StringValue("foo"), rule.StringValue("bar")), rule.StringValue("first")),
		rule.New(rule.Eq(rule.StringValue("bar"), rule.StringParam("foo")), rule.StringValue("second")),
		rule.New(rule.True(), rule.StringValue("default")),
	)

	raw, err := json.Marshal(r1)
	require.NoError(t, err)

	var r2 Ruleset
	err = json.Unmarshal(raw, &r2)
	require.NoError(t, err)

	require.Equal(t, r1, &r2)
}

func TestRulesetParams(t *testing.T) {
	r1 := NewRuleset(
		rule.New(rule.Eq(rule.StringParam("foo"), rule.Int64Param("bar")), rule.StringValue("first")),
		rule.New(rule.Eq(rule.StringParam("foo"), rule.Float64Param("baz")), rule.StringValue("second")),
		rule.New(rule.True(), rule.StringValue("default")),
	)
	require.Equal(t, []rule.Param{
		*rule.StringParam("foo"),
		*rule.Int64Param("bar"),
		*rule.Float64Param("baz"),
	}, r1.Params())
}
