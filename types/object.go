package types

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/internal/reflect"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// ObjectType is an AttributeType representing an object.
type ObjectType struct {
	AttrTypes map[string]attr.Type
}

// WithAttributeTypes returns a new copy of the type with its attribute types
// set.
func (o ObjectType) WithAttributeTypes(typs map[string]attr.Type) attr.TypeWithAttributeTypes {
	return ObjectType{
		AttrTypes: typs,
	}
}

// AttributeTypes returns the type's attribute types.
func (o ObjectType) AttributeTypes() map[string]attr.Type {
	return o.AttrTypes
}

// TerraformType returns the tftypes.Type that should be used to
// represent this type. This constrains what user input will be
// accepted and what kind of data can be set in state. The framework
// will use this to translate the AttributeType to something Terraform
// can understand.
func (o ObjectType) TerraformType(ctx context.Context) tftypes.Type {
	attributeTypes := map[string]tftypes.Type{}
	for k, v := range o.AttrTypes {
		attributeTypes[k] = v.TerraformType(ctx)
	}
	return tftypes.Object{
		AttributeTypes: attributeTypes,
	}
}

// ValueFromTerraform returns an AttributeValue given a tftypes.Value.
// This is meant to convert the tftypes.Value into a more convenient Go
// type for the provider to consume the data with.
func (o ObjectType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	object := Object{
		AttrTypes: o.AttrTypes,
	}
	if !in.Type().Is(o.TerraformType(ctx)) {
		return nil, fmt.Errorf("expected %s, got %s", o.TerraformType(ctx), in.Type())
	}
	if !in.IsKnown() {
		object.Unknown = true
		return object, nil
	}
	if in.IsNull() {
		object.Null = true
		return object, nil
	}
	attributes := map[string]attr.Value{}

	val := map[string]tftypes.Value{}
	err := in.As(&val)
	if err != nil {
		return nil, err
	}

	for k, v := range val {
		a, err := object.AttrTypes[k].ValueFromTerraform(ctx, v)
		if err != nil {
			return nil, err
		}
		attributes[k] = a
	}
	object.Attrs = attributes
	return object, nil
}

// Equal returns true if `candidate` is also an ObjectType and has the same
// AttributeTypes.
func (o ObjectType) Equal(candidate attr.Type) bool {
	other, ok := candidate.(ObjectType)
	if !ok {
		return false
	}
	if len(other.AttrTypes) != len(o.AttrTypes) {
		return false
	}
	for k, v := range o.AttrTypes {
		attr, ok := other.AttrTypes[k]
		if !ok {
			return false
		}
		if !v.Equal(attr) {
			return false
		}
	}
	return true
}

// ApplyTerraform5AttributePathStep applies the given AttributePathStep to the
// object.
func (o ObjectType) ApplyTerraform5AttributePathStep(step tftypes.AttributePathStep) (interface{}, error) {
	if _, ok := step.(tftypes.AttributeName); !ok {
		return nil, fmt.Errorf("cannot apply step %T to ObjectType", step)
	}

	return o.AttrTypes[string(step.(tftypes.AttributeName))], nil
}

// Object represents an object
type Object struct {
	// Unknown will be set to true if the entire object is an unknown value.
	// If only some of the elements in the object are unknown, their known or
	// unknown status will be represented however that AttributeValue
	// surfaces that information. The Object's Unknown property only tracks
	// if the number of elements in a Object is known, not whether the
	// elements that are in the object are known.
	Unknown bool

	// Null will be set to true if the object is null, either because it was
	// omitted from the configuration, state, or plan, or because it was
	// explicitly set to null.
	Null bool

	Attrs map[string]attr.Value

	AttrTypes map[string]attr.Type
}

// ObjectAsOptions is a collection of toggles to control the behavior of
// Object.As.
type ObjectAsOptions struct {
	// UnhandledNullAsEmpty controls what happens when As needs to put a
	// null value in a type that has no way to preserve that distinction.
	// When set to true, the type's empty value will be used.  When set to
	// false, an error will be returned.
	UnhandledNullAsEmpty bool

	// UnhandledUnknownAsEmpty controls what happens when As needs to put
	// an unknown value in a type that has no way to preserve that
	// distinction. When set to true, the type's empty value will be used.
	// When set to false, an error will be returned.
	UnhandledUnknownAsEmpty bool
}

// As populates `target` with the data in the Object, throwing an error if the
// data cannot be stored in `target`.
func (o Object) As(ctx context.Context, target interface{}, opts ObjectAsOptions) error {
	// we need a tftypes.Value for this Object to be able to use it with
	// our reflection code
	obj := ObjectType{AttrTypes: o.AttrTypes}
	typ := obj.TerraformType(ctx)
	val, err := o.ToTerraformValue(ctx)
	if err != nil {
		return err
	}
	err = tftypes.ValidateValue(typ, val)
	if err != nil {
		return err
	}
	return reflect.Into(ctx, obj, tftypes.NewValue(typ, val), target, reflect.Options{
		UnhandledNullAsEmpty:    opts.UnhandledNullAsEmpty,
		UnhandledUnknownAsEmpty: opts.UnhandledUnknownAsEmpty,
	})
}

// ToTerraformValue returns the data contained in the AttributeValue as
// a Go type that tftypes.NewValue will accept.
func (o Object) ToTerraformValue(ctx context.Context) (interface{}, error) {
	if o.Unknown {
		return tftypes.UnknownValue, nil
	}
	if o.Null {
		return nil, nil
	}
	vals := map[string]tftypes.Value{}

	for k, v := range o.Attrs {
		val, err := v.ToTerraformValue(ctx)
		if err != nil {
			return nil, err
		}
		err = tftypes.ValidateValue(o.AttrTypes[k].TerraformType(ctx), val)
		if err != nil {
			return nil, err
		}
		vals[k] = tftypes.NewValue(o.AttrTypes[k].TerraformType(ctx), val)
	}
	return vals, nil
}

// Equal must return true if the AttributeValue is considered
// semantically equal to the AttributeValue passed as an argument.
func (o Object) Equal(c attr.Value) bool {
	other, ok := c.(Object)
	if !ok {
		return false
	}
	if o.Unknown != other.Unknown {
		return false
	}
	if o.Null != other.Null {
		return false
	}
	if len(o.AttrTypes) != len(other.AttrTypes) {
		return false
	}
	for k, v := range o.AttrTypes {
		attr, ok := other.AttrTypes[k]
		if !ok {
			return false
		}
		if !v.Equal(attr) {
			return false
		}
	}
	if len(o.Attrs) != len(other.Attrs) {
		return false
	}
	for k, v := range o.Attrs {
		attr, ok := other.Attrs[k]
		if !ok {
			return false
		}
		if !v.Equal(attr) {
			return false
		}
	}

	return true
}
