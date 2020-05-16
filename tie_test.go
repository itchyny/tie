package tie

import (
	"errors"
	"testing"
)

type x1 struct {
	y Y1
}

func newX1(y Y1) *x1 { return &x1{y} }

func (x *x1) FooX() int { return x.y.FooY() }

type y1 struct{}

func newY1() *y1 { return &y1{} }

func (*y1) FooY() int { return 42 }

type Y1 interface {
	FooY() int
}

type z1 struct{}

func (z *z1) FooZ() {}

type w1 struct{}

func (w *w1) FooY() int { return 24 }

func TestBuilder(t *testing.T) {
	got, err := New(&x1{}).With(&y1{}).Build()
	if err != nil {
		t.Fatal(err)
	}
	x := got.(*x1)
	if got, expected := x.FooX(), 42; got != expected {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

func TestBuilderOverwrite(t *testing.T) {
	got, err := New(&x1{}).With(&y1{}).With(&w1{}).Build()
	if err != nil {
		t.Fatal(err)
	}
	x := got.(*x1)
	if got, expected := x.FooX(), 24; got != expected {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

func TestBuilderFunc(t *testing.T) {
	got, err := New(newX1).With(newY1).Build()
	if err != nil {
		t.Fatal(err)
	}
	x := got.(*x1)
	if got, expected := x.FooX(), 42; got != expected {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

func TestBuilderFuncMixed(t *testing.T) {
	got, err := New(newX1).With(&y1{}).Build()
	if err != nil {
		t.Fatal(err)
	}
	x := got.(*x1)
	if got, expected := x.FooX(), 42; got != expected {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

func TestBuilderDependencyNotEnoughError(t *testing.T) {
	_, err := New(&x1{}).Build()
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if got, expected := err.Error(), "dependency not enough: x1#y"; got != expected {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

func TestBuilderUnusedComponentError(t *testing.T) {
	_, err := New(&x1{}).With(&y1{}).With(&z1{}).Build()
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if got, expected := err.Error(), "unused component: github.com/itchyny/tie.z1"; got != expected {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

func TestBuilderStructError(t *testing.T) {
	_, err := New(&x1{}).With(y1{}).Build()
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if got, expected := err.Error(), "not a struct pointer nor a func: github.com/itchyny/tie.y1"; got != expected {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

func TestBuilderStructError2(t *testing.T) {
	_, err := New(&x1{}).With(new(int)).Build()
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if got, expected := err.Error(), "not a struct pointer nor a func: int"; got != expected {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

func TestBuilderFuncDependency(t *testing.T) {
	_, err := New(newX1).Build()
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if got, expected := err.Error(), "dependency not enough: github.com/itchyny/tie.Y1 for func(tie.Y1) *tie.x1"; got != expected {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

func TestBuilderFuncArgs(t *testing.T) {
	_, err := New(func(int) *x1 { return nil }).Build()
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if got, expected := err.Error(), "not a struct pointer nor an interface: int for func(int) *tie.x1"; got != expected {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

func TestBuilderFuncReturnValues(t *testing.T) {
	_, err := New(func() {}).Build()
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if got, expected := err.Error(), "unexpected number of return values: func()"; got != expected {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

type x2 struct {
	y Y2
	z Z2
}

func newX2(y Y2, z Z2) *x2 { return &x2{y, z} }

func newX2E(y Y2, z Z2) (*x2, error) { return &x2{y, z}, nil }

func (x *x2) FooX() int { return x.y.FooY() + x.z.FooZ() }

type Y2 interface {
	FooY() int
}

type Z2 interface {
	FooZ() int
}

type W2 interface {
	FooW() int
}

type y2 struct {
	w W2
}

func newY2(w W2) *y2 { return &y2{w} }

func newY2E(w W2) (*y2, error) { return &y2{w}, nil }

func (y *y2) FooY() int { return y.w.FooW() }

type z2 struct {
	w W2
}

func newZ2(w W2) *z2 { return &z2{w} }

func newZ2E(w W2) (*z2, error) { return &z2{w}, nil }

func (z *z2) FooZ() int { return z.w.FooW() }

type w2 struct {
	v int
}

func newW2() *w2 { return &w2{} }

func newW2E() (*w2, error) { return &w2{}, nil }

func (w *w2) FooW() int {
	w.v++
	return w.v
}

func TestBuilderDiamond(t *testing.T) {
	got, err := New(&x2{}).With(&y2{}).With(&z2{}).With(&w2{24}).Build()
	if err != nil {
		t.Fatal(err)
	}
	x := got.(*x2)
	if got, expected := x.FooX(), 51; got != expected {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

func TestBuilderDiamond2(t *testing.T) {
	got, err := New(&x2{}).With(&w2{24}).With(&y2{}).With(&z2{}).Build()
	if err != nil {
		t.Fatal(err)
	}
	x := got.(*x2)
	if got, expected := x.FooX(), 51; got != expected {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

func TestBuilderDiamondDependencyNotEnoughError(t *testing.T) {
	_, err := New(&x2{}).With(&y2{}).With(&z2{}).Build()
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if got, expected := err.Error(), "dependency not enough: y2#w"; got != expected {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

func TestBuilderFuncDiamond(t *testing.T) {
	got, err := New(newX2).With(newY2).With(newZ2).With(newW2).Build()
	if err != nil {
		t.Fatal(err)
	}
	x := got.(*x2)
	if got, expected := x.FooX(), 3; got != expected {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

func TestBuilderFuncError(t *testing.T) {
	got, err := New(newX2E).With(newY2E).With(newZ2E).With(newW2E).Build()
	if err != nil {
		t.Fatal(err)
	}
	x := got.(*x2)
	if got, expected := x.FooX(), 3; got != expected {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

func TestBuilderFuncError2(t *testing.T) {
	_, err := New(newX2E).With(newY2E).With(newZ2E).
		With(func() (*w2, error) { return nil, errors.New("error") }).Build()
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if got, expected := err.Error(), "error"; got != expected {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

func TestBuilderFuncError3(t *testing.T) {
	_, err := New(newX2E).With(newY2E).With(newZ2E).
		With(func() (*w2, int) { return nil, 1 }).Build()
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if got, expected := err.Error(), "second return value is not an error: func() (*tie.w2, int)"; got != expected {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

func TestBuilderFuncError4(t *testing.T) {
	_, err := New(newX2E).With(newY2E).With(newZ2E).
		With(func() (int, error) { return 1, nil }).Build()
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if got, expected := err.Error(), "not a struct pointer: int in func() (int, error)"; got != expected {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

type x3 struct {
	y Y3
	z Z3
}

func newX3(y Y3, z Z3) *x3 { return &x3{y, z} }

func (x *x3) FooX() int { return x.y.FooY1() + x.z.FooZ1() }

type Y3 interface {
	FooY1() int
	FooY2() int
}

type Z3 interface {
	FooZ1() int
	FooZ2() int
}

type y3 struct {
	z Z3
}

func newY3(z Z3) *y3 { return &y3{z} }

func (y *y3) FooY1() int { return y.z.FooZ2() }

func (y *y3) FooY2() int { return 12 }

type z3 struct {
	y Y3
}

func newZ3(y Y3) *z3 { return &z3{y} }

func (z *z3) FooZ1() int { return z.y.FooY2() }

func (z *z3) FooZ2() int { return 18 }

func TestBuilderCyclic(t *testing.T) {
	got, err := New(&x3{}).With(&y3{}).With(&z3{}).Build()
	if err != nil {
		t.Fatal(err)
	}
	x := got.(*x3)
	if got, expected := x.FooX(), 30; got != expected {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

func TestBuilderFuncCyclicError(t *testing.T) {
	_, err := New(newX3).With(newY3).With(newZ3).Build()
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if got, expected := err.Error(), "dependency has a cycle"; got != expected {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

type x4 struct {
	y *y4
}

func (x *x4) FooX() int { return x.y.FooY() }

type y4 struct{}

func (*y4) FooY() int { return 42 }

func TestBuilderStructPtr(t *testing.T) {
	got, err := New(&x4{}).With(&y4{}).Build()
	if err != nil {
		t.Fatal(err)
	}
	x := got.(*x4)
	if got, expected := x.FooX(), 42; got != expected {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

type xConflict struct {
	y, z Y1
}

func TestBuilderInterfaceConflictError(t *testing.T) {
	_, err := New(&xConflict{}).Build()
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if got, expected := err.Error(), "interface conflict in xConflict: github.com/itchyny/tie.Y1"; got != expected {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

func TestBuilderFuncInterfaceConflictError(t *testing.T) {
	_, err := New(func() *xConflict { return &xConflict{} }).Build()
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if got, expected := err.Error(), "interface conflict in xConflict: github.com/itchyny/tie.Y1"; got != expected {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

type yConflict struct {
	x *xConflict
}

func TestBuilderInterfaceConflictError2(t *testing.T) {
	_, err := New(&yConflict{}).With(&xConflict{}).Build()
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if got, expected := err.Error(), "interface conflict in xConflict: github.com/itchyny/tie.Y1"; got != expected {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}
