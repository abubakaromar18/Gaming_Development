package fixed

// Angle measures rotation in "binary degrees": 256 steps per full turn, the
// way 8-bit arcade games stored headings. Overflow wraps for free, so turning
// past 255 lands back on 0 with no modulo, and the value indexes a lookup
// table directly.
type Angle uint8

// Quarter turns, handy for building headings and in tests.
const (
	Angle0   Angle = 0
	Angle90  Angle = 64
	Angle180 Angle = 128
	Angle270 Angle = 192
)

// sinTable holds sin for every Angle. It is computed once from integer
// arithmetic (no math.Sin), so every platform builds the same table.
var sinTable = buildSinTable()

// buildSinTable uses Bhaskara I's 7th-century rational approximation,
//
//	sin(πx/P) ≈ 16x(P−x) / (5P² − 4x(P−x))   for 0 ≤ x ≤ P,
//
// with P = 128 (half a turn). Its error is under 0.002, invisible in a game,
// and it needs nothing but integer multiply and divide.
func buildSinTable() [256]F {
	const half = 128
	var t [256]F
	for i := 0; i < 256; i++ {
		x := int64(i % half)
		p := x * (half - x)
		num := 16 * p << Shift
		den := 5*half*half - 4*p
		v := F(num / den)
		if i >= half {
			v = -v
		}
		t[i] = v
	}
	return t
}

// Sin returns the sine of a.
func Sin(a Angle) F { return sinTable[a] }

// Cos returns the cosine of a, which is the sine a quarter turn ahead.
func Cos(a Angle) F { return sinTable[a+Angle90] }

// Radians converts to radians for the renderer. Never use it in the simulation.
func (a Angle) Radians() float64 { return float64(a) * (2 * 3.141592653589793 / 256) }
