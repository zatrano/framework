package num

import (
	"cmp"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Format formats a number with thousand separators and decimals.
func Format(value float64, decimals int, decimalSep, thousandSep string) string {
	if decimals < 0 {
		decimals = 0
	}
	if decimalSep == "" {
		decimalSep = "."
	}
	if thousandSep == "" {
		thousandSep = ","
	}
	negative := value < 0
	if negative {
		value = -value
	}
	pow := math.Pow(10, float64(decimals))
	rounded := math.Round(value*pow) / pow
	raw := strconv.FormatFloat(rounded, 'f', decimals, 64)
	parts := strings.SplitN(raw, ".", 2)
	intPart := parts[0]
	var b strings.Builder
	n := len(intPart)
	for i, ch := range intPart {
		if i > 0 && (n-i)%3 == 0 {
			b.WriteString(thousandSep)
		}
		b.WriteRune(ch)
	}
	out := b.String()
	if decimals > 0 && len(parts) == 2 {
		out += decimalSep + parts[1]
	}
	if negative {
		out = "-" + out
	}
	return out
}

// Currency formats as currency-like string.
func Currency(value float64, symbol string, decimals ...int) string {
	d := 2
	if len(decimals) > 0 {
		d = decimals[0]
	}
	if symbol == "" {
		symbol = "$"
	}
	return symbol + Format(value, d, ".", ",")
}

// Percentage formats a ratio (0-1 or already percent) as percentage string.
func Percentage(value float64, decimals int, alreadyPercent ...bool) string {
	v := value
	if len(alreadyPercent) == 0 || !alreadyPercent[0] {
		v = value * 100
	}
	return Format(v, decimals, ".", ",") + "%"
}

// Clamp constrains value between min and max.
func Clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// ClampInt constrains value between min and max.
func ClampInt(value, min, max int64) int64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// NthRoot returns the n-th root of value (Pow(value, 1/n)).
func NthRoot(value, n float64) float64 {
	if n == 0 {
		return math.NaN()
	}
	return math.Pow(value, 1/n)
}

// PercentChange returns the percentage change from from to to ((to-from)/from*100).
// When from is zero, returns 0.
func PercentChange(from, to float64) float64 {
	if from == 0 {
		return 0
	}
	return ((to - from) / from) * 100
}

// ApproxEqual reports whether |a-b| is within epsilon (absolute).
func ApproxEqual(a, b, epsilon float64) bool {
	if epsilon < 0 {
		epsilon = -epsilon
	}
	return math.Abs(a-b) <= epsilon
}

// Ordinal returns an English ordinal string (1st, 2nd, 3rd, 4th...).
func Ordinal(n int64) string {
	abs := n
	if abs < 0 {
		abs = -abs
	}
	mod100 := abs % 100
	suffix := "th"
	if mod100 < 11 || mod100 > 13 {
		switch abs % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return strconv.FormatInt(n, 10) + suffix
}

// Between reports whether value is within inclusive bounds.
func Between(value, min, max float64) bool {
	return value >= min && value <= max
}

// ToInt converts common numeric types to int.
func ToInt(value any, fallback ...int) int {
	fb := 0
	if len(fallback) > 0 {
		fb = fallback[0]
	}
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return fb
		}
		return n
	default:
		n, err := strconv.Atoi(fmt.Sprint(value))
		if err != nil {
			return fb
		}
		return n
	}
}

// ToFloat converts common numeric types to float64.
func ToFloat(value any, fallback ...float64) float64 {
	fb := 0.0
	if len(fallback) > 0 {
		fb = fallback[0]
	}
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		n, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return fb
		}
		return n
	default:
		n, err := strconv.ParseFloat(fmt.Sprint(value), 64)
		if err != nil {
			return fb
		}
		return n
	}
}

// FileSize formats bytes into a human-readable size.
func FileSize(bytes int64, precision int) string {
	if precision < 0 {
		precision = 0
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	size := float64(bytes)
	unit := 0
	for size >= 1024 && unit < len(units)-1 {
		size /= 1024
		unit++
	}
	return Format(size, precision, ".", ",") + " " + units[unit]
}

// Abs returns the absolute value.
func Abs(value float64) float64 {
	return math.Abs(value)
}

// Floor returns the greatest integer value less than or equal to value.
func Floor(value float64) float64 {
	return math.Floor(value)
}

// Ceil returns the least integer value greater than or equal to value.
func Ceil(value float64) float64 {
	return math.Ceil(value)
}

// Truncate returns the integer part of value (toward zero).
func Truncate(value float64) float64 {
	return math.Trunc(value)
}

// Sqrt returns the square root of value (NaN if value < 0).
func Sqrt(value float64) float64 {
	return math.Sqrt(value)
}

// Pow returns base**exp.
func Pow(base, exp float64) float64 {
	return math.Pow(base, exp)
}

// Log returns the natural logarithm of value.
func Log(value float64) float64 {
	return math.Log(value)
}

// Log10 returns the base-10 logarithm of value.
func Log10(value float64) float64 {
	return math.Log10(value)
}

// Exp returns e**value.
func Exp(value float64) float64 {
	return math.Exp(value)
}

// Cbrt returns the cube root of value.
func Cbrt(value float64) float64 {
	return math.Cbrt(value)
}

// Sin returns the sine of value (radians).
func Sin(value float64) float64 {
	return math.Sin(value)
}

// Cos returns the cosine of value (radians).
func Cos(value float64) float64 {
	return math.Cos(value)
}

// Tan returns the tangent of value (radians).
func Tan(value float64) float64 {
	return math.Tan(value)
}

// ToRadians converts degrees to radians.
func ToRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}

// ToDegrees converts radians to degrees.
func ToDegrees(radians float64) float64 {
	return radians * 180 / math.Pi
}

// Hypot returns Sqrt(x*x + y*y).
func Hypot(x, y float64) float64 {
	return math.Hypot(x, y)
}

// Log2 returns the base-2 logarithm of value.
func Log2(value float64) float64 {
	return math.Log2(value)
}

// Asin returns the arcsine of value (radians).
func Asin(value float64) float64 {
	return math.Asin(value)
}

// Acos returns the arccosine of value (radians).
func Acos(value float64) float64 {
	return math.Acos(value)
}

// Atan returns the arctangent of value (radians).
func Atan(value float64) float64 {
	return math.Atan(value)
}

// Atan2 returns the arctangent of y/x (radians), using the signs of both to determine the quadrant.
func Atan2(y, x float64) float64 {
	return math.Atan2(y, x)
}

// Mod returns the floating-point remainder of a/b (sign follows a).
func Mod(a, b float64) float64 {
	return math.Mod(a, b)
}

// Remainder returns the IEEE 754 floating-point remainder of a/b.
func Remainder(a, b float64) float64 {
	return math.Remainder(a, b)
}

// CopySign returns a value with the magnitude of mag and the sign of sign.
func CopySign(mag, sign float64) float64 {
	return math.Copysign(mag, sign)
}

// MaxAbs returns the largest absolute value among values.
func MaxAbs(values ...float64) float64 {
	if len(values) == 0 {
		return 0
	}
	best := math.Abs(values[0])
	for _, v := range values[1:] {
		if a := math.Abs(v); a > best {
			best = a
		}
	}
	return best
}

// MinAbs returns the smallest absolute value among values.
func MinAbs(values ...float64) float64 {
	if len(values) == 0 {
		return 0
	}
	best := math.Abs(values[0])
	for _, v := range values[1:] {
		if a := math.Abs(v); a < best {
			best = a
		}
	}
	return best
}

// SafeDiv returns a/b, or fallback (default 0) when b is zero.
func SafeDiv(a, b float64, fallback ...float64) float64 {
	if b == 0 {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return 0
	}
	return a / b
}

// Sign returns -1, 0, or 1 depending on the sign of value.
func Sign(value float64) float64 {
	switch {
	case value > 0:
		return 1
	case value < 0:
		return -1
	default:
		return 0
	}
}

// IsPositive reports whether value is greater than zero.
func IsPositive(value float64) bool {
	return value > 0
}

// IsNegative reports whether value is less than zero.
func IsNegative(value float64) bool {
	return value < 0
}

// IsZero reports whether value is exactly zero.
func IsZero(value float64) bool {
	return value == 0
}

// Lerp linearly interpolates between a and b by t (t=0 → a, t=1 → b).
func Lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

// ToHex returns the hexadecimal representation of n (lowercase, no 0x prefix).
func ToHex(n int64) string {
	return strconv.FormatInt(n, 16)
}

// FromHex parses a hex string (optional 0x prefix). Uses fallback on error.
func FromHex(value string, fallback ...int64) int64 {
	raw := strings.TrimSpace(value)
	raw = strings.TrimPrefix(strings.ToLower(raw), "0x")
	n, err := strconv.ParseInt(raw, 16, 64)
	if err != nil {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return 0
	}
	return n
}

// ToBinary returns the binary representation of n (no 0b prefix).
func ToBinary(n int64) string {
	return strconv.FormatInt(n, 2)
}

// FromBinary parses a binary string (optional 0b prefix). Uses fallback on error.
func FromBinary(value string, fallback ...int64) int64 {
	raw := strings.TrimSpace(value)
	raw = strings.TrimPrefix(strings.ToLower(raw), "0b")
	n, err := strconv.ParseInt(raw, 2, 64)
	if err != nil {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return 0
	}
	return n
}

// ToOctal returns the octal representation of n (no 0o prefix).
func ToOctal(n int64) string {
	return strconv.FormatInt(n, 8)
}

// FromOctal parses an octal string (optional 0o prefix). Uses fallback on error.
func FromOctal(value string, fallback ...int64) int64 {
	raw := strings.TrimSpace(value)
	raw = strings.TrimPrefix(strings.ToLower(raw), "0o")
	n, err := strconv.ParseInt(raw, 8, 64)
	if err != nil {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return 0
	}
	return n
}

// IsEven reports whether n is even.
func IsEven(n int64) bool {
	return n%2 == 0
}

// IsOdd reports whether n is odd.
func IsOdd(n int64) bool {
	return n%2 != 0
}

// IsPowerOfTwo reports whether n is a positive power of two.
func IsPowerOfTwo(n int64) bool {
	return n > 0 && n&(n-1) == 0
}

// Digits returns the number of decimal digits in n (0 has 1 digit).
func Digits(n int64) int {
	if n < 0 {
		n = -n
	}
	if n == 0 {
		return 1
	}
	count := 0
	for n > 0 {
		n /= 10
		count++
	}
	return count
}

// FloorInt returns the greatest int64 less than or equal to value.
func FloorInt(value float64) int64 {
	return int64(math.Floor(value))
}

// CeilInt returns the least int64 greater than or equal to value.
func CeilInt(value float64) int64 {
	return int64(math.Ceil(value))
}

// Factorial returns n! (0 when n < 0).
func Factorial(n int64) int64 {
	if n < 0 {
		return 0
	}
	var result int64 = 1
	for i := int64(2); i <= n; i++ {
		result *= i
	}
	return result
}

// IsPrime reports whether n is a prime number.
func IsPrime(n int64) bool {
	if n <= 1 {
		return false
	}
	if n <= 3 {
		return true
	}
	if n%2 == 0 || n%3 == 0 {
		return false
	}
	for i := int64(5); i*i <= n; i += 6 {
		if n%i == 0 || n%(i+2) == 0 {
			return false
		}
	}
	return true
}

// IsInteger reports whether value is a finite whole number.
func IsInteger(value float64) bool {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return false
	}
	return value == math.Trunc(value)
}

// IsFinite reports whether value is neither NaN nor ±Inf.
func IsFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

// IsNaN reports whether value is an IEEE 754 “not-a-number” value.
func IsNaN(value float64) bool {
	return math.IsNaN(value)
}

// IsInf reports whether value is ±Inf. When sign is 0, either infinity matches;
// +1 matches +Inf only; -1 matches -Inf only.
func IsInf(value float64, sign ...int) bool {
	s := 0
	if len(sign) > 0 {
		s = sign[0]
	}
	return math.IsInf(value, s)
}

// Gcd returns the greatest common divisor of a and b (always non-negative).
func Gcd(a, b int64) int64 {
	if a < 0 {
		a = -a
	}
	if b < 0 {
		b = -b
	}
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// Lcm returns the least common multiple of a and b (always non-negative).
func Lcm(a, b int64) int64 {
	if a == 0 || b == 0 {
		return 0
	}
	g := Gcd(a, b)
	if a < 0 {
		a = -a
	}
	if b < 0 {
		b = -b
	}
	return (a / g) * b
}

// Round rounds to the given precision.
func Round(value float64, precision int) float64 {
	if precision < 0 {
		precision = 0
	}
	pow := math.Pow(10, float64(precision))
	return math.Round(value*pow) / pow
}

// InRange reports whether value is within [min, max].
func InRange(value, min, max float64) bool {
	if min > max {
		min, max = max, min
	}
	return value >= min && value <= max
}

// Min returns the smallest value (zero value when empty).
func Min[T cmp.Ordered](values ...T) T {
	var zero T
	if len(values) == 0 {
		return zero
	}
	m := values[0]
	for _, v := range values[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

// Max returns the largest value (zero value when empty).
func Max[T cmp.Ordered](values ...T) T {
	var zero T
	if len(values) == 0 {
		return zero
	}
	m := values[0]
	for _, v := range values[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

// Sum returns the sum of values.
func Sum(values ...float64) float64 {
	total := 0.0
	for _, v := range values {
		total += v
	}
	return total
}

// Avg returns the arithmetic mean (0 when empty).
func Avg(values ...float64) float64 {
	if len(values) == 0 {
		return 0
	}
	return Sum(values...) / float64(len(values))
}

// Variance returns the population variance (0 when empty).
func Variance(values ...float64) float64 {
	n := len(values)
	if n == 0 {
		return 0
	}
	mean := Avg(values...)
	var sum float64
	for _, v := range values {
		d := v - mean
		sum += d * d
	}
	return sum / float64(n)
}

// StdDev returns the population standard deviation (0 when empty).
func StdDev(values ...float64) float64 {
	return math.Sqrt(Variance(values...))
}

// PercentageOf returns part as a percent of total (0 when total is 0).
func PercentageOf(part, total float64) float64 {
	if total == 0 {
		return 0
	}
	return (part / total) * 100
}
