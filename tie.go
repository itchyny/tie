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
	builders []*builder
	unused   reflect.Type
}

// New creates a new Builder.
func New(v interface{}) Builder {
	rv := reflect.ValueOf(v)
	return &builder{value: v, rv: rv, rt: rv.Type().Elem()}
}

func (b *builder) With(v interface{}) Builder {
	if _, ok := b.with(v); !ok {
		b.unused = reflect.TypeOf(v)
	}
	return b
}

func (b *builder) with(v interface{}) (Builder, bool) {
	var ok bool
	t := reflect.TypeOf(v)
	for i := 0; i < b.rt.NumField(); i++ {
		if t.ConvertibleTo(b.rt.Field(i).Type) {
			w := b.rv.Elem().Field(i)
			reflect.NewAt(w.Type(), unsafe.Pointer(w.UnsafeAddr())).Elem().Set(reflect.ValueOf(v))
			b.builders = append(b.builders, New(v).(*builder))
			ok = true
			break
		}
	}
	for _, b := range b.builders {
		if _, k := b.with(v); k {
			ok = true
		}
	}
	return b, ok
}

func (b *builder) Build() (interface{}, error) {
	if t := b.unused; t != nil {
		return nil, fmt.Errorf("unused component: %s", stringify(t))
	}
	return b.build()
}

func (b *builder) build() (interface{}, error) {
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
		if _, err := b.build(); err != nil {
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

func stringify(t reflect.Type) string {
	if t.Kind() == reflect.Ptr {
		return stringify(t.Elem())
	}
	return t.PkgPath() + "." + t.Name()
}
