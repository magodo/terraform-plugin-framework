package types

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/internal/reflect"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var (
	_ attr.Type  = ListType{}
	_ attr.Value = &List{}
)

// ListType is an AttributeType representing a list of values. All values must
// be of the same type, which the provider must specify as the ElemType
// property.
type ListType struct {
	ElemType attr.Type
}

// ElementType returns the attr.Type elements will be created from.
func (l ListType) ElementType() attr.Type {
	return l.ElemType
}

// WithElementType returns a ListType that is identical to `l`, but with the
// element type set to `typ`.
func (l ListType) WithElementType(typ attr.Type) attr.TypeWithElementType {
	return ListType{ElemType: typ}
}

// TerraformType returns the tftypes.Type that should be used to
// represent this type. This constrains what user input will be
// accepted and what kind of data can be set in state. The framework
// will use this to translate the AttributeType to something Terraform
// can understand.
func (l ListType) TerraformType(ctx context.Context) tftypes.Type {
	return tftypes.List{
		ElementType: l.ElemType.TerraformType(ctx),
	}
}

// ValueFromTerraform returns an AttributeValue given a tftypes.Value.
// This is meant to convert the tftypes.Value into a more convenient Go
// type for the provider to consume the data with.
func (l ListType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	if !in.Type().Is(l.TerraformType(ctx)) {
		return nil, fmt.Errorf("can't use %s as value of List with ElementType %T, can only use %s values", in.String(), l.ElemType, l.ElemType.TerraformType(ctx).String())
	}
	list := List{
		ElemType: l.ElemType,
	}
	if !in.IsKnown() {
		list.Unknown = true
		return list, nil
	}
	if in.IsNull() {
		list.Null = true
		return list, nil
	}
	val := []tftypes.Value{}
	err := in.As(&val)
	if err != nil {
		return nil, err
	}
	elems := make([]attr.Value, 0, len(val))
	for _, elem := range val {
		av, err := l.ElemType.ValueFromTerraform(ctx, elem)
		if err != nil {
			return nil, err
		}
		elems = append(elems, av)
	}
	list.Elems = elems
	return list, nil
}

// Equal returns true if `o` is also a ListType and has the same ElemType.
func (l ListType) Equal(o attr.Type) bool {
	if l.ElemType == nil {
		return false
	}
	other, ok := o.(ListType)
	if !ok {
		return false
	}
	return l.ElemType.Equal(other.ElemType)
}

// ApplyTerraform5AttributePathStep applies the given AttributePathStep to the
// list.
func (l ListType) ApplyTerraform5AttributePathStep(step tftypes.AttributePathStep) (interface{}, error) {
	if _, ok := step.(tftypes.ElementKeyInt); !ok {
		return nil, fmt.Errorf("cannot apply step %T to ListType", step)
	}

	return l.ElemType, nil
}

// List represents a list of AttributeValues, all of the same type, indicated
// by ElemType.
type List struct {
	// Unknown will be set to true if the entire list is an unknown value.
	// If only some of the elements in the list are unknown, their known or
	// unknown status will be represented however that AttributeValue
	// surfaces that information. The List's Unknown property only tracks
	// if the number of elements in a List is known, not whether the
	// elements that are in the list are known.
	Unknown bool

	// Null will be set to true if the list is null, either because it was
	// omitted from the configuration, state, or plan, or because it was
	// explicitly set to null.
	Null bool

	// Elems are the elements in the list.
	Elems []attr.Value

	// ElemType is the tftypes.Type of the elements in the list. All
	// elements in the list must be of this type.
	ElemType attr.Type
}

// ElementsAs populates `target` with the elements of the List, throwing an
// error if the elements cannot be stored in `target`.
func (l List) ElementsAs(ctx context.Context, target interface{}, allowUnhandled bool) error {
	// we need a tftypes.Value for this List to be able to use it with our
	// reflection code
	values, err := l.ToTerraformValue(ctx)
	if err != nil {
		return err
	}
	return reflect.Into(ctx, ListType{ElemType: l.ElemType}, tftypes.NewValue(tftypes.List{
		ElementType: l.ElemType.TerraformType(ctx),
	}, values), target, reflect.Options{
		UnhandledNullAsEmpty:    allowUnhandled,
		UnhandledUnknownAsEmpty: allowUnhandled,
	})
}

// ToTerraformValue returns the data contained in the AttributeValue as
// a Go type that tftypes.NewValue will accept.
func (l List) ToTerraformValue(ctx context.Context) (interface{}, error) {
	if l.Unknown {
		return tftypes.UnknownValue, nil
	}
	if l.Null {
		return nil, nil
	}
	vals := make([]tftypes.Value, 0, len(l.Elems))
	for _, elem := range l.Elems {
		val, err := elem.ToTerraformValue(ctx)
		if err != nil {
			return nil, err
		}
		err = tftypes.ValidateValue(l.ElemType.TerraformType(ctx), val)
		if err != nil {
			return nil, fmt.Errorf("error validating terraform type: %w", err)
		}
		vals = append(vals, tftypes.NewValue(l.ElemType.TerraformType(ctx), val))
	}
	return vals, nil
}

// Equal must return true if the AttributeValue is considered
// semantically equal to the AttributeValue passed as an argument.
func (l List) Equal(o attr.Value) bool {
	other, ok := o.(List)
	if !ok {
		return false
	}
	if l.Unknown != other.Unknown {
		return false
	}
	if l.Null != other.Null {
		return false
	}
	if !l.ElemType.Equal(other.ElemType) {
		return false
	}
	if len(l.Elems) != len(other.Elems) {
		return false
	}
	for pos, lElem := range l.Elems {
		oElem := other.Elems[pos]
		if !lElem.Equal(oElem) {
			return false
		}
	}
	return true
}
