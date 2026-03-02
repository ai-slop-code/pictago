package utils

import (
	"reflect"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

type mockFieldLevel struct {
	field string
}

func (mockFieldLevel) Top() reflect.Value {
	panic("implement me")
}

func (mockFieldLevel) Parent() reflect.Value {
	panic("implement me")
}

func (m mockFieldLevel) Field() reflect.Value {
	return reflect.ValueOf(m.field)
}

func (mockFieldLevel) FieldName() string {
	panic("implement me")
}

func (mockFieldLevel) StructFieldName() string {
	panic("implement me")
}

func (mockFieldLevel) Param() string {
	panic("implement me")
}

func (mockFieldLevel) GetTag() string {
	panic("implement me")
}

func (mockFieldLevel) ExtractType(_ reflect.Value) (value reflect.Value, kind reflect.Kind, nullable bool) {
	panic("implement me")
}

func (mockFieldLevel) GetStructFieldOK() (reflect.Value, reflect.Kind, bool) {
	panic("implement me")
}

func (mockFieldLevel) GetStructFieldOKAdvanced(
	_ reflect.Value, _ string,
) (reflect.Value, reflect.Kind, bool) {
	panic("implement me")
}

func (mockFieldLevel) GetStructFieldOK2() (reflect.Value, reflect.Kind, bool, bool) {
	panic("implement me")
}

func (mockFieldLevel) GetStructFieldOKAdvanced2(
	_ reflect.Value, _ string,
) (reflect.Value, reflect.Kind, bool, bool) {
	panic("implement me")
}

var _ validator.FieldLevel = &mockFieldLevel{}

func NewMockFieldLevel(field string) validator.FieldLevel {
	return &mockFieldLevel{field: field}
}

func TestValidator_NotBlank(t *testing.T) {
	t.Run("Should return false on blank string", func(t *testing.T) {
		tests := []string{
			"",
			"   ",
			"\n",
			"\t",
		}
		for _, tt := range tests {
			result := notBlank(NewMockFieldLevel(tt))
			assert.False(t, result)
		}
	})

	t.Run("Should return true on non-blank string", func(t *testing.T) {
		tests := []string{
			"a",
			"  a  ",
			"hello",
			"  hello world  ",
		}
		for _, tt := range tests {
			result := notBlank(NewMockFieldLevel(tt))
			assert.True(t, result)
		}
	})
}
