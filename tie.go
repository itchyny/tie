package tie

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"unsafe"
)

// Builder is the dependency resolving builder.
type Builder []interface{}

// New creates a new Builder.
func New(v interface{}) Builder {
	return Builder{v}
}

// With appends a new component.
func (b Builder) With(v interface{}) Builder {
	return append(b, v)
}

// Build the components.
func (b Builder) Build() (interface{}, error) {
	n := len(b)
	values := make([]reflect.Value, n)
	types := make([]reflect.Type, n)
	funcs := make([]reflect.Type, n)
	used := make([]bool, n)

	// validate and collect types
	for i, v := range b {
		switch t := reflect.TypeOf(v); t.Kind() {
		case reflect.Ptr:
			if t.Elem().Kind() != reflect.Struct {
				return nil, fmt.Errorf("not a struct pointer nor a func: %s", stringify(t))
			}
			types[i] = t
			values[i] = reflect.ValueOf(v)
		case reflect.Func:
			switch t.NumOut() {
			case 2:
				if !t.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
					return nil, fmt.Errorf("second return value is not an error: %s", stringify(t))
				}
				fallthrough
			case 1:
				u := t.Out(0)
				if u.Kind() != reflect.Ptr || u.Elem().Kind() != reflect.Struct {
					return nil, fmt.Errorf("not a struct pointer: %s in %s", stringify(u), stringify(t))
				}
				types[i] = u
				funcs[i] = t
			default:
				return nil, fmt.Errorf("unexpected number of return values: %s", stringify(t))
			}
		default:
			return nil, fmt.Errorf("not a struct pointer nor a func: %s", stringify(t))
		}
	}

	// build dependency adjacent matrix
	xs := make([]bool, n*n)
	adj := make([][]bool, n)
	for i := 0; i < n; i++ {
		adj[i] = xs[i*n : (i+1)*n]
	}
	for i, t := range funcs {
		if t == nil {
			t = types[i].Elem()
			for j := 0; j < t.NumField(); j++ {
				u := t.Field(j).Type
				for k, t := range types {
					if i != k && t.AssignableTo(u) {
						adj[k][i] = true
					}
				}
			}
		} else {
			for j := 0; j < t.NumIn(); j++ {
				u := t.In(j)
				if k := u.Kind(); !(k == reflect.Ptr && u.Elem().Kind() == reflect.Struct || k == reflect.Interface) {
					return nil, fmt.Errorf("not a struct pointer nor an interface: %s for %s", stringify(u), stringify(t))
				}
				var found bool
				for k, t := range types {
					if i != k && t.AssignableTo(u) {
						adj[k][i] = true
						found = true
					}
				}
				if !found {
					return nil, fmt.Errorf("dependency not enough: %s for %s", stringify(u), stringify(t))
				}
			}
		}
	}

	// topological sort
	ls, err := tsort(n, adj)
	if err != nil {
		if err, ok := err.(cycleError); ok {
			var sb strings.Builder
			sb.WriteString(err.Error())
			var prev int
			for i, k := range []int(err) {
				if i == 0 {
					sb.WriteString(":\n  ")
					if funcs[k] == nil {
						sb.WriteString(stringify(types[k]))
					} else {
						sb.WriteString(stringify(funcs[k]))
					}
				} else {
					sb.WriteString("\n    -> ")
					if funcs[k] == nil {
						for t, j := types[k].Elem(), 0; j < t.NumField(); j++ {
							if types[prev].AssignableTo(t.Field(j).Type) {
								sb.WriteString(stringify(t.Field(j).Type))
								break
							}
						}
					} else {
						for t, j := funcs[k], 0; j < t.NumIn(); j++ {
							if types[prev].AssignableTo(t.In(j)) {
								sb.WriteString(stringify(t.In(j)))
								break
							}
						}
					}
					sb.WriteString(" for ")
					if funcs[k] == nil {
						sb.WriteString(stringify(types[k]))
					} else {
						sb.WriteString(stringify(funcs[k]))
					}
				}
				prev = k
			}
			return nil, errors.New(sb.String())
		}
		return nil, err
	}

	// initialize function dependencies and fill in struct fields
	for i, l := range ls {
		if u := funcs[l]; u == nil {
			v, t := values[l], types[l].Elem()
			for j := 0; j < t.NumField(); j++ {
				u := t.Field(j).Type
				for k := 0; k < i; k++ {
					if adj[ls[k]][l] && types[ls[k]].AssignableTo(u) {
						w := v.Elem().Field(j)
						reflect.NewAt(w.Type(), unsafe.Pointer(w.UnsafeAddr())).Elem().Set(values[ls[k]])
						used[ls[k]] = true
						break
					}
				}
			}
		} else {
			args := make([]reflect.Value, u.NumIn())
			for j := 0; j < u.NumIn(); j++ {
				u := u.In(j)
				for k := 0; k < i; k++ {
					if adj[ls[k]][l] && types[ls[k]].AssignableTo(u) {
						args[j] = values[ls[k]]
						used[ls[k]] = true
						break
					}
				}
			}
			ys := reflect.ValueOf(b[l]).Call(args)
			if len(ys) == 2 && !ys[1].IsNil() {
				return nil, ys[1].Interface().(error)
			}
			values[l] = ys[0]
		}
	}

	// check unused components
	for i := 1; i < n; i++ {
		if !used[i] {
			return nil, fmt.Errorf("unused component: %s", stringify(types[i]))
		}
	}

	// check not enough dependency
	for i, v := range values {
		e, t := v.Elem(), types[i].Elem()
		for i := 0; i < t.NumField(); i++ {
			if e.Field(i).Kind() == reflect.Interface && e.Field(i).IsNil() {
				return nil, fmt.Errorf("dependency not enough: %s#%s", t.Name(), t.Field(i).Name)
			}
		}
	}

	// return the first value as an interface
	return values[0].Interface(), nil
}

// MustBuild builds the components and panic on error.
func (b Builder) MustBuild() interface{} {
	v, err := b.Build()
	if err != nil {
		panic(err)
	}
	return v
}

func stringify(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Ptr:
		return "*" + stringify(t.Elem())
	case reflect.Func:
		return fmt.Sprint(t)
	}
	if t.PkgPath() == "" {
		return t.Name()
	}
	return filepath.Base(t.PkgPath()) + "." + t.Name()
}

func tsort(n int, adj [][]bool) ([]int, error) {
	ts := make([]int, 0, n) // result
	qs := make([]int, 0, n) // breadth-first search
	vs := make([]bool, n)   // visited
	deg := make([]int, n)   // indegree
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
	if len(ts) == n {
		return ts, nil
	}
	var ps [][2]int // linked list
	var ks []int    // depth-first search
	for i := 0; i < n; i++ {
		if !vs[i] {
			ks = append(ks, len(ps))
			ps = append(ps, [2]int{i, -1})
		}
	}
	for k := 0; len(ks) > 0; {
		k, ks = ks[len(ks)-1], ks[:len(ks)-1]
		for i, j := ps[k][0], 0; j < n; j++ {
			if !vs[j] && adj[i][j] {
				for l, m := k, 2; ; m++ {
					if j == ps[l][0] {
						xs := make([]int, m)
						xs[len(xs)-1] = j
						for m -= 2; k >= 0 && m >= 0; m-- {
							xs[m], k = ps[k][0], ps[k][1]
						}
						return nil, cycleError(xs)
					}
					if l = ps[l][1]; l < 0 {
						break
					}
				}
				ks = append(ks, len(ps))
				ps = append(ps, [2]int{j, k})
			}
		}
	}
	return nil, cycleError(nil)
}

type cycleError []int

func (err cycleError) Error() string {
	return "dependency has a cycle"
}
