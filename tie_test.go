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
		t.Errorf("expected error but got nil")
	}
	if got, expected := err.Error(), "dependency not enough: x1#y"; got != expected {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}
