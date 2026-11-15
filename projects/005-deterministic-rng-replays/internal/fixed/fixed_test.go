package fixed

import (
	"math"
	"testing"
)

func TestMulDiv(t *testing.T) {
	a := FromRatio(3, 2) // 1.5
	b := FromInt(4)
	if got := a.Mul(b); got != FromInt(6) {
		t.Fatalf("1.5*4 = %v, want 6", got)
	}
	if got := b.Div(a); (got - FromRatio(8, 3)).Abs() > 1 {
		t.Fatalf("4/1.5 = %v, want 2.6667", got)
	}
	// The 64-bit intermediate keeps products of large values from overflowing
	// before the shift: 20000 * 0.5 needs 47 bits along the way.
	if got := FromInt(20000).Mul(FromRatio(1, 2)); got != FromInt(10000) {
		t.Fatalf("20000*0.5 = %v", got)
	}
}

func TestWrap(t *testing.T) {
	size := FromInt(100)
	cases := []struct{ in, want int }{{0, 0}, {50, 50}, {100, 0}, {130, 30}, {-1, 99}, {-250, 50}}
	for _, c := range cases {
		if got := Wrap(FromInt(c.in), size); got != FromInt(c.want) {
			t.Errorf("Wrap(%d) = %v, want %d", c.in, got, c.want)
		}
	}
}

func TestWrapDeltaTakesShortestWayAround(t *testing.T) {
	size := FromInt(100)
	if got := WrapDelta(FromInt(95), FromInt(5), size); got != FromInt(10) {
		t.Fatalf("95→5 = %v, want +10 (across the seam)", got)
	}
	if got := WrapDelta(FromInt(5), FromInt(95), size); got != FromInt(-10) {
		t.Fatalf("5→95 = %v, want -10", got)
	}
	if got := WrapDelta(FromInt(20), FromInt(30), size); got != FromInt(10) {
		t.Fatalf("20→30 = %v, want 10", got)
	}
}

func TestSqrt(t *testing.T) {
	for _, n := range []int{0, 1, 2, 9, 100, 12345} {
		got := Sqrt(FromInt(n)).Float64()
		if math.Abs(got-math.Sqrt(float64(n))) > 1e-3 {
			t.Errorf("Sqrt(%d) = %f", n, got)
		}
	}
	if Sqrt(-One) != 0 {
		t.Error("Sqrt of a negative value should be 0")
	}
	if got := (Vec{FromInt(3), FromInt(4)}).Len(); got != FromInt(5) {
		t.Errorf("|(3,4)| = %v, want 5", got)
	}
}

func TestSinTableIsAccurateAndExactAtQuarters(t *testing.T) {
	if Sin(Angle0) != 0 || Sin(Angle90) != One || Sin(Angle180) != 0 || Sin(Angle270) != -One {
		t.Fatalf("quarter turns not exact: %v %v %v %v", Sin(0), Sin(64), Sin(128), Sin(192))
	}
	worst := 0.0
	for i := 0; i < 256; i++ {
		a := Angle(i)
		worst = math.Max(worst, math.Abs(Sin(a).Float64()-math.Sin(a.Radians())))
		worst = math.Max(worst, math.Abs(Cos(a).Float64()-math.Cos(a.Radians())))
	}
	if worst > 0.002 {
		t.Fatalf("max trig error %f exceeds Bhaskara bound", worst)
	}
}

func TestAngleOverflowWraps(t *testing.T) {
	a := Angle(250)
	a += 10
	if a != 4 {
		t.Fatalf("250+10 = %d, want 4", a)
	}
}

func TestVectorOps(t *testing.T) {
	a, b := V(1, 2), V(3, 5)
	if a.Add(b) != V(4, 7) || b.Sub(a) != V(2, 3) || a.Scale(FromInt(3)) != V(3, 6) {
		t.Fatal("basic vector ops wrong")
	}
	if got := FromAngle(Angle90); got != (Vec{0, One}) {
		t.Fatalf("90° heading = %v, want straight down (0,1)", got)
	}
	w, h := FromInt(100), FromInt(50)
	if got := WrapVec(V(-10, 60), w, h); got != V(90, 10) {
		t.Fatalf("WrapVec = %v", got)
	}
	if got := WrapDeltaVec(V(98, 1), V(2, 49), w, h); got != V(4, -2) {
		t.Fatalf("WrapDeltaVec = %v", got)
	}
}
