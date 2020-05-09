package tie

import (
	"errors"
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
	fs := make([]reflect.Type, n)

	// validate and collect types
	for i, v := range b {
		unused[i] = true
		switch t := reflect.TypeOf(v); t.Kind() {
		case reflect.Ptr:
			ts[i] = t
			switch t := t.Elem(); t.Kind() {
			case reflect.Struct:
				vs[i] = reflect.ValueOf(v)
			default:
				return nil, fmt.Errorf("not a struct pointer nor a func: %s", stringify(t))
			}
		case reflect.Func:
			switch t.NumOut() {
			case 2:
				if !t.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
					return nil, fmt.Errorf("second return value is not an error: %s", stringify(t))
				}
				fallthrough
			case 1:
				switch t := t.Out(0); t.Kind() {
				case reflect.Ptr:
					ts[i] = t
					switch t := t.Elem(); t.Kind() {
					case reflect.Struct:
					default:
						return nil, fmt.Errorf("not a struct pointer nor a func: %s", stringify(t))
					}
				default:
					return nil, fmt.Errorf("not a struct pointer nor a func: %s", stringify(t))
				}
			default:
				return nil, fmt.Errorf("unexpected number of return values: %s", stringify(t))
			}
			fs[i] = t
		default:
			return nil, fmt.Errorf("not a struct pointer nor a func: %s", stringify(t))
		}
	}

	// check duplicate interface in each struct just in case
	for _, t := range ts {
		m := make(map[string]struct{}, t.Elem().NumField())
		for i := 0; i < t.Elem().NumField(); i++ {
			key := stringify(t.Elem().Field(i).Type)
			if _, ok := m[key]; ok {
				return nil, fmt.Errorf("interface conflict in %s: %s", t.Elem().Name(), key)
			}
			m[key] = struct{}{}
		}
	}

	// check function arguments and build dependency adjacent matrix
	xs := make([]bool, n*n)
	adj := make([][]bool, n)
	for i := 0; i < n; i++ {
		adj[i] = xs[i*n : (i+1)*n]
	}
	for i, u := range fs {
		if u != nil {
			for j := 0; j < u.NumIn(); j++ {
				u := u.In(j)
				switch u.Kind() {
				case reflect.Interface:
				case reflect.Ptr:
					if u.Elem().Kind() != reflect.Struct {
						return nil, fmt.Errorf("not a struct pointer nor an interface: %s for %s", stringify(u), stringify(fs[i]))
					}
				default:
					return nil, fmt.Errorf("not a struct pointer nor an interface: %s for %s", stringify(u), stringify(fs[i]))
				}
				var found bool
				for k, t := range ts {
					if t.AssignableTo(u) {
						adj[k][i] = true
						found = true
					}
				}
				if !found {
					return nil, fmt.Errorf("dependency not enough: %s for %s", stringify(u), stringify(fs[i]))
				}
			}
		}
	}

	// topological sort
	ls, err := tsort(n, adj)
	if err != nil {
		return nil, err
	}

	// initialize function dependencies
	for _, i := range ls {
		if u := fs[i]; u != nil {
			args := make([]reflect.Value, u.NumIn())
			for j := 0; j < u.NumIn(); j++ {
				u := u.In(j)
				for k, t := range ts {
					if t.AssignableTo(u) {
						args[j] = vs[k]
						unused[k] = false
					}
				}
			}
			ys := reflect.ValueOf(b[i]).Call(args)
			if len(ys) == 2 && !ys[1].IsNil() {
				return nil, ys[1].Interface().(error)
			}
			vs[i] = ys[0]
		}
	}

	// fill in struct fields
	for i := 1; i < n; i++ {
		v, t := vs[i], ts[i]
		for j, w := range vs {
			u := ts[j]
			for k := 0; k < u.Elem().NumField(); k++ {
				if t.AssignableTo(u.Elem().Field(k).Type) {
					w := w.Elem().Field(k)
					reflect.NewAt(w.Type(), unsafe.Pointer(w.UnsafeAddr())).Elem().Set(v)
					unused[i] = false
					break
				}
			}
		}
	}

	// check unused components
	for i := 1; i < n; i++ {
		if unused[i] {
			return nil, fmt.Errorf("unused component: %s", stringify(ts[i]))
		}
	}

	// check not enough dependency
	for i, v := range vs {
		t := ts[i]
		for i := 0; i < t.Elem().NumField(); i++ {
			if v.Elem().Field(i).Kind() == reflect.Interface && v.Elem().Field(i).IsNil() {
				return nil, fmt.Errorf("dependency not enough: %s#%s", t.Elem().Name(), t.Elem().Field(i).Name)
			}
		}
	}
	return vs[0].Interface(), nil
}

func (b builder) MustBuild() interface{} {
	v, err := b.Build()
	if err != nil {
		panic(err)
	}
	return v
}

func stringify(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Ptr:
		return stringify(t.Elem())
	case reflect.Func:
		return fmt.Sprint(t)
	}
	if t.PkgPath() == "" {
		return t.Name()
	}
	return t.PkgPath() + "." + t.Name()
}

func tsort(n int, adj [][]bool) ([]int, error) {
	ts := make([]int, 0, n)
	qs := make([]int, 0, n)
	vs := make([]bool, n)
	deg := make([]int, n)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if adj[i][j] {
				deg[j]++
			}
		}
	}
	for i := 0; i < n; i++ {
		if deg[i] == 0 {
			qs = append(qs, i)
			vs[i] = true
		}
	}
	if len(qs) == 0 && n > 0 {
		return nil, errors.New("dependency has a cycle")
	}
	var x int
	for len(qs) > 0 {
		x, qs = qs[0], qs[1:]
		ts = append(ts, x)
		for i := 0; i < n; i++ {
			if adj[x][i] && !vs[i] {
				deg[i]--
				if deg[i] == 0 {
					qs = append(qs, i)
					vs[i] = true
				}
			}
		}
	}
	return ts, nil
}
