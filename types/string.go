package types

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func stringValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	if !in.IsKnown() {
		return String{Unknown: true}, nil
	}
	if in.IsNull() {
		return String{Null: true}, nil
	}
	var s string
	err := in.As(&s)
	if err != nil {
		return nil, err
	}
	return String{Value: s}, nil
}

var _ attr.Value = String{}

// String represents a UTF-8 string value.
type String struct {
	// Unknown will be true if the value is not yet known.
	Unknown bool

	// Null will be true if the value was not set, or was explicitly set to
	// null.
	Null bool

	// Value contains the set value, as long as Unknown and Null are both
	// false.
	Value string
}

// ToTerraformValue returns the data contained in the *String as a string. If
// Unknown is true, it returns a tftypes.UnknownValue. If Null is true, it
// returns nil.
func (s String) ToTerraformValue(_ context.Context) (interface{}, error) {
	if s.Null {
		return nil, nil
	}
	if s.Unknown {
		return tftypes.UnknownValue, nil
	}
	return s.Value, nil
}

// Equal returns true if `other` is a *String and has the same value as `s`.
func (s String) Equal(other attr.Value) bool {
	o, ok := other.(String)
	if !ok {
		return false
	}
	if s.Unknown != o.Unknown {
		return false
	}
	if s.Null != o.Null {
		return false
	}
	return s.Value == o.Value
}
