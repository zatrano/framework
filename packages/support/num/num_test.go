package num_test

import (
	"math"
	"testing"

	"github.com/zatrano/framework/packages/support/num"
)

func TestNumHelpers(t *testing.T) {
	if num.Currency(1234.5, "$") != "$1,234.50" {
		t.Fatalf("currency=%q", num.Currency(1234.5, "$"))
	}
	if num.Percentage(0.25, 0) != "25%" {
		t.Fatalf("pct=%q", num.Percentage(0.25, 0))
	}
	if num.Clamp(150, 0, 100) != 100 {
		t.Fatal("clamp")
	}
	if num.FileSize(1536, 1) != "1.5 KB" {
		t.Fatalf("size=%q", num.FileSize(1536, 1))
	}
	if num.Ordinal(1) != "1st" || num.Ordinal(2) != "2nd" || num.Ordinal(3) != "3rd" || num.Ordinal(11) != "11th" {
		t.Fatalf("ordinal failed")
	}
	if num.Abs(-3) != 3 || num.Round(1.234, 2) != 1.23 || !num.InRange(5, 1, 10) {
		t.Fatal("abs/round/inrange")
	}
}

func TestNumMathHelpers(t *testing.T) {
	if num.Floor(1.9) != 1 || num.Floor(-1.1) != -2 {
		t.Fatal("floor")
	}
	if num.Ceil(1.1) != 2 || num.Ceil(-1.9) != -1 {
		t.Fatal("ceil")
	}
	if num.Truncate(1.9) != 1 || num.Truncate(-1.9) != -1 {
		t.Fatal("truncate")
	}
	if num.Sqrt(9) != 3 || num.Pow(2, 3) != 8 {
		t.Fatal("sqrt/pow")
	}
	if num.Sign(5) != 1 || num.Sign(-3) != -1 || num.Sign(0) != 0 {
		t.Fatal("sign")
	}
	if got := num.NthRoot(27, 3); math.Abs(got-3) > 1e-9 {
		t.Fatalf("nthRoot=%v", got)
	}
	if got := num.PercentChange(50, 75); math.Abs(got-50) > 1e-9 {
		t.Fatalf("percentChange=%v", got)
	}
	if num.PercentChange(0, 10) != 0 {
		t.Fatal("percentChange zero")
	}
	if num.ClampInt(15, 0, 10) != 10 || num.ClampInt(-3, 0, 10) != 0 || num.ClampInt(5, 0, 10) != 5 {
		t.Fatal("clampInt")
	}
	if !num.ApproxEqual(1.0001, 1.0, 0.001) || num.ApproxEqual(1.1, 1.0, 0.01) {
		t.Fatal("approxEqual")
	}
}

func TestNumSignAndLerp(t *testing.T) {
	if !num.IsPositive(1.5) || num.IsPositive(0) || num.IsPositive(-1) {
		t.Fatal("IsPositive")
	}
	if !num.IsNegative(-0.1) || num.IsNegative(0) || num.IsNegative(2) {
		t.Fatal("IsNegative")
	}
	if !num.IsZero(0) || num.IsZero(0.0001) {
		t.Fatal("IsZero")
	}
	if got := num.Lerp(0, 10, 0.5); got != 5 {
		t.Fatalf("lerp mid=%v", got)
	}
	if got := num.Lerp(2, 8, 0); got != 2 || num.Lerp(2, 8, 1) != 8 {
		t.Fatalf("lerp ends=%v", got)
	}
}

func TestNumBaseConversion(t *testing.T) {
	if got := num.ToHex(255); got != "ff" {
		t.Fatalf("to hex=%q", got)
	}
	if num.FromHex("ff") != 255 || num.FromHex("0x10") != 16 {
		t.Fatal("from hex")
	}
	if num.FromHex("zz", 7) != 7 || num.FromHex("zz") != 0 {
		t.Fatal("from hex fallback")
	}
	if got := num.ToBinary(10); got != "1010" {
		t.Fatalf("to binary=%q", got)
	}
	if num.FromBinary("1010") != 10 || num.FromBinary("0b111") != 7 {
		t.Fatal("from binary")
	}
	if num.FromBinary("2", 3) != 3 {
		t.Fatal("from binary fallback")
	}
	if got := num.ToOctal(64); got != "100" {
		t.Fatalf("to octal=%q", got)
	}
	if num.FromOctal("100") != 64 || num.FromOctal("0o17") != 15 {
		t.Fatal("from octal")
	}
	if num.FromOctal("9", 4) != 4 || num.FromOctal("9") != 0 {
		t.Fatal("from octal fallback")
	}
}

func TestNumFactorialPrimeFinite(t *testing.T) {
	if num.Factorial(5) != 120 || num.Factorial(0) != 1 || num.Factorial(-1) != 0 {
		t.Fatalf("factorial=%d", num.Factorial(5))
	}
	if !num.IsPrime(17) || num.IsPrime(1) || num.IsPrime(15) || num.IsPrime(2) == false {
		t.Fatal("IsPrime")
	}
	if !num.IsInteger(4) || !num.IsInteger(4.0) || num.IsInteger(4.2) || num.IsInteger(math.NaN()) {
		t.Fatal("IsInteger")
	}
	if !num.IsFinite(1.5) || num.IsFinite(math.Inf(1)) || num.IsFinite(math.NaN()) {
		t.Fatal("IsFinite")
	}
	if !num.IsNaN(math.NaN()) || num.IsNaN(1.5) {
		t.Fatal("IsNaN")
	}
	if !num.IsInf(math.Inf(1)) || !num.IsInf(math.Inf(-1), -1) || num.IsInf(1.5) {
		t.Fatal("IsInf")
	}
	if num.IsInf(math.Inf(1), -1) || !num.IsInf(math.Inf(1), 1) {
		t.Fatal("IsInf sign")
	}
}

func TestNumLogExpCbrt(t *testing.T) {
	if got := num.Log(math.E); math.Abs(got-1) > 1e-9 {
		t.Fatalf("log=%v", got)
	}
	if got := num.Log10(1000); math.Abs(got-3) > 1e-9 {
		t.Fatalf("log10=%v", got)
	}
	if got := num.Exp(0); got != 1 {
		t.Fatalf("exp=%v", got)
	}
	if got := num.Cbrt(27); math.Abs(got-3) > 1e-9 {
		t.Fatalf("cbrt=%v", got)
	}
	if got := num.Cbrt(-8); math.Abs(got-(-2)) > 1e-9 {
		t.Fatalf("cbrt neg=%v", got)
	}
}

func TestNumTrig(t *testing.T) {
	if got := num.Sin(0); got != 0 {
		t.Fatalf("sin=%v", got)
	}
	if got := num.Cos(0); got != 1 {
		t.Fatalf("cos=%v", got)
	}
	if got := num.Tan(0); got != 0 {
		t.Fatalf("tan=%v", got)
	}
	rad := num.ToRadians(180)
	if math.Abs(rad-math.Pi) > 1e-9 {
		t.Fatalf("to radians=%v", rad)
	}
	if got := num.ToDegrees(math.Pi); math.Abs(got-180) > 1e-9 {
		t.Fatalf("to degrees=%v", got)
	}
	if got := num.Sin(num.ToRadians(90)); math.Abs(got-1) > 1e-9 {
		t.Fatalf("sin 90=%v", got)
	}
	if got := num.Hypot(3, 4); math.Abs(got-5) > 1e-9 {
		t.Fatalf("hypot=%v", got)
	}
	if got := num.Log2(8); math.Abs(got-3) > 1e-9 {
		t.Fatalf("log2=%v", got)
	}
	if got := num.Asin(1); math.Abs(got-(math.Pi/2)) > 1e-9 {
		t.Fatalf("asin=%v", got)
	}
	if got := num.Acos(1); math.Abs(got) > 1e-9 {
		t.Fatalf("acos=%v", got)
	}
	if got := num.Atan(0); got != 0 {
		t.Fatalf("atan=%v", got)
	}
	if got := num.Atan2(0, -1); math.Abs(got-math.Pi) > 1e-9 {
		t.Fatalf("atan2=%v", got)
	}
	if got := num.Mod(5.5, 2); math.Abs(got-1.5) > 1e-9 {
		t.Fatalf("mod=%v", got)
	}
	if got := num.Remainder(5.5, 2); math.Abs(got-(-0.5)) > 1e-9 {
		t.Fatalf("remainder=%v", got)
	}
	if got := num.CopySign(3.5, -1); got != -3.5 {
		t.Fatalf("copysign=%v", got)
	}
	if got := num.MaxAbs(-7, 3, -2); got != 7 {
		t.Fatalf("maxAbs=%v", got)
	}
	if got := num.MinAbs(-7, 3, -2); got != 2 {
		t.Fatalf("minAbs=%v", got)
	}
	if got := num.SafeDiv(10, 4); math.Abs(got-2.5) > 1e-9 {
		t.Fatalf("safeDiv=%v", got)
	}
	if got := num.SafeDiv(10, 0, -1); got != -1 || num.SafeDiv(10, 0) != 0 {
		t.Fatal("safeDiv zero")
	}
}

func TestNumIntegerHelpers(t *testing.T) {
	if !num.IsEven(4) || num.IsEven(3) || !num.IsOdd(3) || num.IsOdd(2) {
		t.Fatal("even/odd")
	}
	if num.Gcd(48, 18) != 6 || num.Gcd(-48, 18) != 6 || num.Gcd(0, 5) != 5 {
		t.Fatalf("gcd=%d", num.Gcd(48, 18))
	}
	if num.Lcm(4, 6) != 12 || num.Lcm(-4, 6) != 12 || num.Lcm(0, 5) != 0 {
		t.Fatalf("lcm=%d", num.Lcm(4, 6))
	}
	if !num.IsPowerOfTwo(8) || num.IsPowerOfTwo(6) || num.IsPowerOfTwo(0) || num.IsPowerOfTwo(-2) {
		t.Fatal("IsPowerOfTwo")
	}
	if num.Digits(0) != 1 || num.Digits(123) != 3 || num.Digits(-45) != 2 {
		t.Fatalf("Digits=%d/%d/%d", num.Digits(0), num.Digits(123), num.Digits(-45))
	}
	if num.FloorInt(1.9) != 1 || num.FloorInt(-1.1) != -2 {
		t.Fatalf("FloorInt=%d/%d", num.FloorInt(1.9), num.FloorInt(-1.1))
	}
	if num.CeilInt(1.1) != 2 || num.CeilInt(-1.9) != -1 {
		t.Fatalf("CeilInt=%d/%d", num.CeilInt(1.1), num.CeilInt(-1.9))
	}
}

func TestNumAggregates(t *testing.T) {
	if num.Min(3, 1, 4, 1, 5) != 1 || num.Max(3, 1, 4, 1, 5) != 5 {
		t.Fatal("min/max")
	}
	if num.Min[int]() != 0 || num.Max("a", "z", "m") != "z" {
		t.Fatal("min/max empty/string")
	}
	if num.Sum(1, 2, 3, 4) != 10 {
		t.Fatalf("sum=%v", num.Sum(1, 2, 3, 4))
	}
	if num.Avg(2, 4, 6) != 4 || num.Avg() != 0 {
		t.Fatalf("avg=%v", num.Avg(2, 4, 6))
	}
	if math.Abs(num.Variance(1, 2, 3, 4, 5)-2) > 1e-9 || num.Variance() != 0 {
		t.Fatalf("variance=%v", num.Variance(1, 2, 3, 4, 5))
	}
	if math.Abs(num.StdDev(1, 2, 3, 4, 5)-math.Sqrt(2)) > 1e-9 || num.StdDev() != 0 {
		t.Fatalf("stddev=%v", num.StdDev(1, 2, 3, 4, 5))
	}
	if num.PercentageOf(25, 200) != 12.5 || num.PercentageOf(1, 0) != 0 {
		t.Fatalf("pctof=%v", num.PercentageOf(25, 200))
	}
}
