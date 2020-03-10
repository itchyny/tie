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
	value    interface{}
	rv       reflect.Value
	rt       reflect.Type
	builders []Builder
}

// New creates a new Builder.
func New(v interface{}) Builder {
	rv := reflect.ValueOf(v)
	return &builder{v, rv, rv.Type().Elem(), nil}
}

func (b *builder) With(v interface{}) Builder {
	t := reflect.TypeOf(v)
	for i := 0; i < b.rt.NumField(); i++ {
		if t.ConvertibleTo(b.rt.Field(i).Type) {
			w := b.rv.Elem().Field(i)
			reflect.NewAt(w.Type(), unsafe.Pointer(w.UnsafeAddr())).Elem().Set(reflect.ValueOf(v))
			b.builders = append(b.builders, New(v))
			break
		}
	}
	for _, b := range b.builders {
		b.With(v)
	}
	return b
}

func (b *builder) Build() (interface{}, error) {
	m := make(map[string]struct{}, b.rt.NumField())
	for i := 0; i < b.rt.NumField(); i++ {
		t := b.rt.Field(i).Type
		key := t.PkgPath() + "." + t.Name()
		if _, ok := m[key]; ok {
			return nil, fmt.Errorf("interface conflict: %s", key)
		}
		m[key] = struct{}{}
	}
	for i := 0; i < b.rt.NumField(); i++ {
		if b.rv.Elem().Field(i).Kind() == reflect.Interface && b.rv.Elem().Field(i).IsNil() {
			return nil, fmt.Errorf("dependency not enough: %s#%s", b.rt.Name(), b.rt.Field(i).Name)
		}
	}
	for _, b := range b.builders {
		if _, err := b.Build(); err != nil {
			return nil, err
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
