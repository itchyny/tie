package tie

import (
	"fmt"
	"reflect"
	"unsafe"
)

// Builder ...
type Builder interface {
	With(interface{}) Builder
	Build() (interface{}, error)
	MustBuild() interface{}
}

type builder struct {
	value  interface{}
	vValue reflect.Value
	vType  reflect.Type
}

// New creates a new Builder.
func New(v interface{}) Builder {
	value := reflect.ValueOf(v)
	return &builder{v, value, value.Type().Elem()}
}

func (b *builder) With(v interface{}) Builder {
	t := reflect.TypeOf(v)
	for i := 0; i < b.vType.NumField(); i++ {
		if t.ConvertibleTo(b.vType.Field(i).Type) {
			w := b.vValue.Elem().Field(i)
			reflect.NewAt(w.Type(), unsafe.Pointer(w.UnsafeAddr())).Elem().Set(reflect.ValueOf(v))
			break
		}
	}
	return b
}

func (b *builder) Build() (interface{}, error) {
	m := make(map[string]struct{}, b.vType.NumField())
	for i := 0; i < b.vType.NumField(); i++ {
		t := b.vType.Field(i).Type
		key := t.PkgPath() + "." + t.Name()
		if _, ok := m[key]; ok {
			return nil, fmt.Errorf("interface conflict: %s", key)
		}
		m[key] = struct{}{}
	}
	for i := 0; i < b.vType.NumField(); i++ {
		if b.vValue.Elem().Field(i).Kind() == reflect.Interface && b.vValue.Elem().Field(i).IsNil() {
			return nil, fmt.Errorf("dependency not enough: %s#%s", b.vType.Name(), b.vType.Field(i).Name)
		}
	}
	return b.value, nil
}

func (b *builder) MustBuild() interface{} {
	v, err := b.Build()
	if err != nil {
		panic(err)
	}
	return v
}
