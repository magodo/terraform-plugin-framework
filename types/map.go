package types

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/internal/reflect"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// MapType is an AttributeType representing a map of values. All values must
// be of the same type, which the provider must specify as the ElemType
// property. Keys will always be strings.
type MapType struct {
	ElemType attr.Type
}

// WithElementType returns a new copy of the type with its element type set.
func (m MapType) WithElementType(typ attr.Type) attr.TypeWithElementType {
	return MapType{
		ElemType: typ,
	}
}

// ElementType returns the type's element type.
func (m MapType) ElementType() attr.Type {
	return m.ElemType
}

// TerraformType returns the tftypes.Type that should be used to represent this
// type. This constrains what user input will be accepted and what kind of data
// can be set in state. The framework will use this to translate the
// AttributeType to something Terraform can understand.
func (m MapType) TerraformType(ctx context.Context) tftypes.Type {
	return tftypes.Map{
		AttributeType: m.ElemType.TerraformType(ctx),
	}
}

// ValueFromTerraform returns an AttributeValue given a tftypes.Value. This is
// meant to convert the tftypes.Value into a more convenient Go type for the
// provider to consume the data with.
func (m MapType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	ma := Map{
		ElemType: m.ElemType,
	}
	if !in.Type().Is(tftypes.Map{}) {
		return nil, fmt.Errorf("can't use %s as value of Map, can only use tftypes.Map values", in.String())
	}
	if !in.Type().Is(tftypes.Map{AttributeType: m.ElemType.TerraformType(ctx)}) {
		return nil, fmt.Errorf("can't use %s as value of Map with ElementType %T, can only use %s values", in.String(), m.ElemType, m.ElemType.TerraformType(ctx).String())
	}
	if !in.IsKnown() {
		ma.Unknown = true
		return ma, nil
	}
	if in.IsNull() {
		ma.Null = true
		return ma, nil
	}
	val := map[string]tftypes.Value{}
	err := in.As(&val)
	if err != nil {
		return nil, err
	}
	elems := make(map[string]attr.Value, len(val))
	for key, elem := range val {
		av, err := m.ElemType.ValueFromTerraform(ctx, elem)
		if err != nil {
			return nil, err
		}
		elems[key] = av
	}
	ma.Elems = elems
	return ma, nil
}

// Equal returns true if `o` is also a MapType and has the same ElemType.
func (m MapType) Equal(o attr.Type) bool {
	if m.ElemType == nil {
		return false
	}
	other, ok := o.(MapType)
	if !ok {
		return false
	}
	return m.ElemType.Equal(other.ElemType)
}

// ApplyTerraform5AttributePathStep applies the given AttributePathStep to the
// map.
func (m MapType) ApplyTerraform5AttributePathStep(step tftypes.AttributePathStep) (interface{}, error) {
	if _, ok := step.(tftypes.ElementKeyString); !ok {
		return nil, fmt.Errorf("cannot apply step %T to MapType", step)
	}

	return m.ElemType, nil
}

// Map represents a map of AttributeValues, all of the same type, indicated by
// ElemType. Keys for the map will always be strings.
type Map struct {
	// Unknown will be set to true if the entire map is an unknown value.
	// If only some of the elements in the map are unknown, their known or
	// unknown status will be represented however that AttributeValue
	// surfaces that information. The Map's Unknown property only tracks if
	// the number of elements in a Map is known, not whether the elements
	// that are in the map are known.
	Unknown bool

	// Null will be set to true if the map is null, either because it was
	// omitted from the configuration, state, or plan, or because it was
	// explicitly set to null.
	Null bool

	// Elems are the elements in the map.
	Elems map[string]attr.Value

	// ElemType is the AttributeType of the elements in the map. All
	// elements in the map must be of this type.
	ElemType attr.Type
}

// ElementsAs populates `target` with the elements of the Map, throwing an
// error if the elements cannot be stored in `target`.
func (m Map) ElementsAs(ctx context.Context, target interface{}, allowUnhandled bool) error {
	// we need a tftypes.Value for this Map to be able to use it with our
	// reflection code
	values := make(map[string]tftypes.Value, len(m.Elems))
	for key, elem := range m.Elems {
		val, err := elem.ToTerraformValue(ctx)
		if err != nil {
			return fmt.Errorf("error getting Terraform value for element %q: %w", key, err)
		}
		err = tftypes.ValidateValue(m.ElemType.TerraformType(ctx), val)
		if err != nil {
			return fmt.Errorf("error using created Terraform value for element %q: %w", key, err)
		}
		values[key] = tftypes.NewValue(m.ElemType.TerraformType(ctx), val)
	}
	return reflect.Into(ctx, MapType{ElemType: m.ElemType}, tftypes.NewValue(tftypes.Map{
		AttributeType: m.ElemType.TerraformType(ctx),
	}, values), target, reflect.Options{
		UnhandledNullAsEmpty:    allowUnhandled,
		UnhandledUnknownAsEmpty: allowUnhandled,
	})
}

// ToTerraformValue returns the data contained in the AttributeValue as a Go
// type that tftypes.NewValue will accept.
func (m Map) ToTerraformValue(ctx context.Context) (interface{}, error) {
	if m.Unknown {
		return tftypes.UnknownValue, nil
	}
	if m.Null {
		return nil, nil
	}
	vals := make(map[string]tftypes.Value, len(m.Elems))
	for key, elem := range m.Elems {
		val, err := elem.ToTerraformValue(ctx)
		if err != nil {
			return nil, err
		}
		err = tftypes.ValidateValue(m.ElemType.TerraformType(ctx), val)
		if err != nil {
			return nil, err
		}
		vals[key] = tftypes.NewValue(m.ElemType.TerraformType(ctx), val)
	}
	return vals, nil
}

// Equal must return true if the AttributeValue is considered semantically
// equal to the AttributeValue passed as an argument.
func (m Map) Equal(o attr.Value) bool {
	other, ok := o.(Map)
	if !ok {
		return false
	}
	if m.Unknown != other.Unknown {
		return false
	}
	if m.Null != other.Null {
		return false
	}
	if !m.ElemType.Equal(other.ElemType) {
		return false
	}
	if len(m.Elems) != len(other.Elems) {
		return false
	}
	for key, mElem := range m.Elems {
		oElem, ok := other.Elems[key]
		if !ok {
			return false
		}
		if !mElem.Equal(oElem) {
			return false
		}
	}
	return true
}
