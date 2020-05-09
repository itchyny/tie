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

// New creates a new Builder.
func New(v interface{}) Builder {
	return builder{v}
}

type builder []interface{}

func (b builder) With(v interface{}) Builder {
	return append(b, v)
}

func (b builder) Build() (interface{}, error) {
	n := len(b)
	unused := make([]bool, n)
	vs := make([]reflect.Value, n)
	ts := make([]reflect.Type, n)
	for i, v := range b {
		unused[i] = true
		vs[i] = reflect.ValueOf(v)
		switch t := reflect.TypeOf(v); t.Kind() {
		case reflect.Ptr:
			switch t := t.Elem(); t.Kind() {
			case reflect.Struct:
				ts[i] = t
			default:
				return nil, fmt.Errorf("not a struct pointer: %s", stringify(t))
			}
		default:
			return nil, fmt.Errorf("not a struct pointer: %s", stringify(t))
		}
	}
	for _, t := range ts {
		m := make(map[string]struct{}, t.NumField())
		for i := 0; i < t.NumField(); i++ {
			key := stringify(t.Field(i).Type)
			if _, ok := m[key]; ok {
				return nil, fmt.Errorf("interface conflict in %s: %s", t.Name(), key)
			}
			m[key] = struct{}{}
		}
	}
	for i := 1; i < n; i++ {
		v, t := vs[i], reflect.TypeOf(b[i])
		for j, w := range vs {
			u := ts[j]
			for k := 0; k < u.NumField(); k++ {
				if t.AssignableTo(u.Field(k).Type) {
					w := w.Elem().Field(k)
					reflect.NewAt(w.Type(), unsafe.Pointer(w.UnsafeAddr())).Elem().Set(v)
					unused[i] = false
					break
				}
			}
		}
	}
	for i := 1; i < n; i++ {
		if unused[i] {
			return nil, fmt.Errorf("unused component: %s", stringify(ts[i]))
		}
	}
	for i, v := range vs {
		t := ts[i]
		for i := 0; i < t.NumField(); i++ {
			if v.Elem().Field(i).Kind() == reflect.Interface && v.Elem().Field(i).IsNil() {
				return nil, fmt.Errorf("dependency not enough: %s#%s", t.Name(), t.Field(i).Name)
			}
		}
	}
	return b[0], nil
}

func (b builder) MustBuild() interface{} {
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
	if t.PkgPath() == "" {
		return t.Name()
	}
	return t.PkgPath() + "." + t.Name()
}
