package tie

import "testing"

type x1 struct {
	y Y1
}

func (x *x1) FooX() int { return x.y.FooY() }

type y1 struct{}

func (*y1) FooY() int { return 42 }

type Y1 interface {
	FooY() int
}

type z1 struct{}

func (z *z1) FooZ() {}

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

type x2 struct {
	y Y2
	z Z2
}

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

func (y *y2) FooY() int { return y.w.FooW() }

type z2 struct {
	w W2
}

func (z *z2) FooZ() int { return z.w.FooW() }

type w2 struct {
	v int
}

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

type x3 struct {
	y Y3
	z Z3
}

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

func (y *y3) FooY1() int { return y.z.FooZ2() }

func (y *y3) FooY2() int { return 12 }

type z3 struct {
	y Y3
}

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

type xConflict struct {
	y, z Y1
}

func TestBuilderInterfaceConflictError(t *testing.T) {
	_, err := New(&xConflict{}).Build()
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if got, expected := err.Error(), "interface conflict: github.com/itchyny/tie.Y1"; got != expected {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}
