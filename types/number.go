package types

import (
	"context"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func numberValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	if !in.IsKnown() {
		return Number{Unknown: true}, nil
	}
	if in.IsNull() {
		return Number{Null: true}, nil
	}
	n := big.NewFloat(0)
	err := in.As(&n)
	if err != nil {
		return nil, err
	}
	return Number{Value: n}, nil
}

var _ attr.Value = Number{}

// Number represents a number value, exposed as a *big.Float. Numbers can be
// floats or integers.
type Number struct {
	// Unknown will be true if the value is not yet known.
	Unknown bool

	// Null will be true if the value was not set, or was explicitly set to
	// null.
	Null bool

	// Value contains the set value, as long as Unknown and Null are both
	// false.
	Value *big.Float
}

// ToTerraformValue returns the data contained in the *Number as a *big.Float.
// If Unknown is true, it returns a tftypes.UnknownValue. If Null is true, it
// returns nil.
func (n Number) ToTerraformValue(_ context.Context) (interface{}, error) {
	if n.Null {
		return nil, nil
	}
	if n.Unknown {
		return tftypes.UnknownValue, nil
	}
	return n.Value, nil
}

// Equal returns true if `other` is a *Number and has the same value as `n`.
func (n Number) Equal(other attr.Value) bool {
	o, ok := other.(Number)
	if !ok {
		return false
	}
	if n.Unknown != o.Unknown {
		return false
	}
	if n.Null != o.Null {
		return false
	}
	if n.Value == nil && o.Value == nil {
		return true
	}
	if n.Value == nil || o.Value == nil {
		return false
	}
	return n.Value.Cmp(o.Value) == 0
}
