package date

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	clockMu sync.RWMutex
	frozen  *time.Time

	weekStartMu sync.RWMutex
	weekStarts  = time.Monday

	humanLocaleMu sync.RWMutex
	humanLocale   = "en"
)

// SetWeekStartsAt configures which weekday StartOfWeek/EndOfWeek use (default Monday).
func SetWeekStartsAt(day time.Weekday) {
	weekStartMu.Lock()
	defer weekStartMu.Unlock()
	weekStarts = day
}

// WeekStartsAt returns the configured first day of the week.
func WeekStartsAt() time.Weekday {
	weekStartMu.RLock()
	defer weekStartMu.RUnlock()
	return weekStarts
}

// SetHumanLocale sets the locale used by HumanDiff ("en" or "tr").
func SetHumanLocale(locale string) {
	humanLocaleMu.Lock()
	defer humanLocaleMu.Unlock()
	humanLocale = strings.ToLower(strings.TrimSpace(locale))
}

// HumanLocale returns the active HumanDiff locale.
func HumanLocale() string {
	humanLocaleMu.RLock()
	defer humanLocaleMu.RUnlock()
	return humanLocale
}

// Pretend freezes the clock used by Now/UTC helpers.
func Pretend(t time.Time) {
	clockMu.Lock()
	defer clockMu.Unlock()
	cp := t
	frozen = &cp
}

// Travel moves the frozen clock by duration (or freezes now+duration).
func Travel(d time.Duration) {
	clockMu.Lock()
	defer clockMu.Unlock()
	base := time.Now()
	if frozen != nil {
		base = *frozen
	}
	next := base.Add(d)
	frozen = &next
}

// Restore clears a frozen clock.
func Restore() {
	clockMu.Lock()
	defer clockMu.Unlock()
	frozen = nil
}

func now() time.Time {
	clockMu.RLock()
	defer clockMu.RUnlock()
	if frozen != nil {
		return *frozen
	}
	return time.Now()
}

// Now returns the current time in Local (respects Pretend).
func Now() time.Time { return now().In(time.Local) }

// UTC returns the current UTC time (respects Pretend).
func UTC() time.Time { return now().UTC() }

// Parse parses a time using common layouts.
func Parse(value string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
		"01/02/2006",
		"02.01.2006",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t, nil
		}
		if t, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse date [%s]", value)
}

// Format formats t with layout (defaults to RFC3339).
func Format(t time.Time, layout ...string) string {
	l := time.RFC3339
	if len(layout) > 0 && layout[0] != "" {
		l = layout[0]
	}
	return t.Format(l)
}

// AddDays adds days to t.
func AddDays(t time.Time, days int) time.Time {
	return t.AddDate(0, 0, days)
}

// SubDays subtracts days from t.
func SubDays(t time.Time, days int) time.Time {
	return AddDays(t, -days)
}

// AddMonths adds months to t.
func AddMonths(t time.Time, months int) time.Time {
	return t.AddDate(0, months, 0)
}

// SubMonths subtracts months from t.
func SubMonths(t time.Time, months int) time.Time {
	return AddMonths(t, -months)
}

// AddQuarters adds calendar quarters (3 months each) to t.
func AddQuarters(t time.Time, quarters int) time.Time {
	return AddMonths(t, quarters*3)
}

// SubQuarters subtracts calendar quarters from t.
func SubQuarters(t time.Time, quarters int) time.Time {
	return AddQuarters(t, -quarters)
}

// AddYears adds years to t.
func AddYears(t time.Time, years int) time.Time {
	return t.AddDate(years, 0, 0)
}

// SubYears subtracts years from t.
func SubYears(t time.Time, years int) time.Time {
	return AddYears(t, -years)
}

// AddHours adds hours to t.
func AddHours(t time.Time, hours int) time.Time {
	return t.Add(time.Duration(hours) * time.Hour)
}

// SubHours subtracts hours from t.
func SubHours(t time.Time, hours int) time.Time {
	return AddHours(t, -hours)
}

// AddMinutes adds minutes to t.
func AddMinutes(t time.Time, minutes int) time.Time {
	return t.Add(time.Duration(minutes) * time.Minute)
}

// SubMinutes subtracts minutes from t.
func SubMinutes(t time.Time, minutes int) time.Time {
	return AddMinutes(t, -minutes)
}

// AddSeconds adds seconds to t.
func AddSeconds(t time.Time, seconds int) time.Time {
	return t.Add(time.Duration(seconds) * time.Second)
}

// SubSeconds subtracts seconds from t.
func SubSeconds(t time.Time, seconds int) time.Time {
	return AddSeconds(t, -seconds)
}

// Copy returns a copy of t (same instant and location).
func Copy(t time.Time) time.Time {
	return time.Unix(0, t.UnixNano()).In(t.Location())
}

// SetTime sets the clock on t's calendar day (optional nanoseconds).
func SetTime(t time.Time, hour, min, sec int, nsec ...int) time.Time {
	ns := 0
	if len(nsec) > 0 {
		ns = nsec[0]
	}
	y, m, d := t.Date()
	return time.Date(y, m, d, hour, min, sec, ns, t.Location())
}

// SetDate sets the calendar date on t, keeping the clock and location.
func SetDate(t time.Time, year int, month time.Month, day int) time.Time {
	hour, min, sec := t.Clock()
	return time.Date(year, month, day, hour, min, sec, t.Nanosecond(), t.Location())
}

// Equal reports whether a and b represent the same instant.
func Equal(a, b time.Time) bool {
	return a.Equal(b)
}

// Gt reports whether a is after b.
func Gt(a, b time.Time) bool {
	return a.After(b)
}

// Gte reports whether a is after or equal to b.
func Gte(a, b time.Time) bool {
	return a.After(b) || a.Equal(b)
}

// Lt reports whether a is before b.
func Lt(a, b time.Time) bool {
	return a.Before(b)
}

// Lte reports whether a is before or equal to b.
func Lte(a, b time.Time) bool {
	return a.Before(b) || a.Equal(b)
}

// Max returns the later of the given times (zero time when empty).
func Max(values ...time.Time) time.Time {
	if len(values) == 0 {
		return time.Time{}
	}
	best := values[0]
	for _, v := range values[1:] {
		if v.After(best) {
			best = v
		}
	}
	return best
}

// Min returns the earlier of the given times (zero time when empty).
func Min(values ...time.Time) time.Time {
	if len(values) == 0 {
		return time.Time{}
	}
	best := values[0]
	for _, v := range values[1:] {
		if v.Before(best) {
			best = v
		}
	}
	return best
}

// Average returns the mean instant of the given times (zero time when empty).
func Average(values ...time.Time) time.Time {
	if len(values) == 0 {
		return time.Time{}
	}
	var sum int64
	for _, v := range values {
		sum += v.UnixNano()
	}
	avg := sum / int64(len(values))
	return time.Unix(0, avg).In(values[0].Location())
}

// Closest returns the time in times nearest to needle (zero time when empty).
func Closest(needle time.Time, times ...time.Time) time.Time {
	if len(times) == 0 {
		return time.Time{}
	}
	best := times[0]
	bestDiff := absDuration(needle.Sub(best))
	for _, candidate := range times[1:] {
		diff := absDuration(needle.Sub(candidate))
		if diff < bestDiff {
			best = candidate
			bestDiff = diff
		}
	}
	return best
}

// Farthest returns the time in times farthest from needle (zero time when empty).
func Farthest(needle time.Time, times ...time.Time) time.Time {
	if len(times) == 0 {
		return time.Time{}
	}
	best := times[0]
	bestDiff := absDuration(needle.Sub(best))
	for _, candidate := range times[1:] {
		diff := absDuration(needle.Sub(candidate))
		if diff > bestDiff {
			best = candidate
			bestDiff = diff
		}
	}
	return best
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

// Timestamp returns Unix seconds for t.
func Timestamp(t time.Time) int64 {
	return t.Unix()
}

// FromTimestamp builds a local time from Unix seconds.
func FromTimestamp(sec int64) time.Time {
	return time.Unix(sec, 0)
}

// TimestampMilli returns Unix milliseconds for t.
func TimestampMilli(t time.Time) int64 {
	return t.UnixMilli()
}

// FromTimestampMilli builds a local time from Unix milliseconds.
func FromTimestampMilli(ms int64) time.Time {
	return time.UnixMilli(ms)
}

// TimestampMicro returns Unix microseconds for t.
func TimestampMicro(t time.Time) int64 {
	return t.UnixMicro()
}

// FromTimestampMicro builds a local time from Unix microseconds.
func FromTimestampMicro(us int64) time.Time {
	return time.UnixMicro(us)
}

// TimestampNano returns Unix nanoseconds for t.
func TimestampNano(t time.Time) int64 {
	return t.UnixNano()
}

// FromTimestampNano builds a local time from Unix nanoseconds.
func FromTimestampNano(ns int64) time.Time {
	return time.Unix(0, ns)
}

// StartOfDay returns midnight for t's location.
func StartOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// EndOfDay returns end of day for t's location.
func EndOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 23, 59, 59, int(time.Second-time.Nanosecond), t.Location())
}

// Noon returns 12:00:00 for t's calendar day.
func Noon(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 12, 0, 0, 0, t.Location())
}

// StartOfHour returns the beginning of t's hour.
func StartOfHour(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
}

// EndOfHour returns the last nanosecond of t's hour.
func EndOfHour(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 59, 59, int(time.Second-time.Nanosecond), t.Location())
}

// StartOfMinute returns the beginning of t's minute.
func StartOfMinute(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, t.Location())
}

// EndOfMinute returns the last nanosecond of t's minute.
func EndOfMinute(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 59, int(time.Second-time.Nanosecond), t.Location())
}

// StartOfSecond returns the beginning of t's second.
func StartOfSecond(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, t.Location())
}

// EndOfSecond returns the last nanosecond of t's second.
func EndOfSecond(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), int(time.Second-time.Nanosecond), t.Location())
}

// StartOfWeek returns the configured week-start day at 00:00 for t's location.
func StartOfWeek(t time.Time) time.Time {
	t = StartOfDay(t)
	start := WeekStartsAt()
	// Normalize both to Monday=0 .. Sunday=6 for distance calc.
	toIndex := func(d time.Weekday) int {
		return (int(d) - int(time.Monday) + 7) % 7
	}
	delta := (toIndex(t.Weekday()) - toIndex(start) + 7) % 7
	return t.AddDate(0, 0, -delta)
}

// EndOfWeek returns the last moment of the configured week containing t.
func EndOfWeek(t time.Time) time.Time {
	return EndOfDay(StartOfWeek(t).AddDate(0, 0, 6))
}

// StartOfMonth returns the first day of t's month at 00:00.
func StartOfMonth(t time.Time) time.Time {
	y, m, _ := t.Date()
	return time.Date(y, m, 1, 0, 0, 0, 0, t.Location())
}

// EndOfMonth returns the last moment of t's month.
func EndOfMonth(t time.Time) time.Time {
	return EndOfDay(StartOfMonth(t).AddDate(0, 1, -1))
}

// StartOfYear returns January 1 00:00 for t's year.
func StartOfYear(t time.Time) time.Time {
	y, _, _ := t.Date()
	return time.Date(y, time.January, 1, 0, 0, 0, 0, t.Location())
}

// EndOfYear returns December 31 end-of-day for t's year.
func EndOfYear(t time.Time) time.Time {
	return EndOfDay(time.Date(t.Year(), time.December, 31, 0, 0, 0, 0, t.Location()))
}

// StartOfQuarter returns the first day of t's calendar quarter at 00:00.
func StartOfQuarter(t time.Time) time.Time {
	month := time.Month((Quarter(t)-1)*3 + 1)
	return time.Date(t.Year(), month, 1, 0, 0, 0, 0, t.Location())
}

// EndOfQuarter returns the last moment of t's calendar quarter.
func EndOfQuarter(t time.Time) time.Time {
	return EndOfDay(StartOfQuarter(t).AddDate(0, 3, -1))
}

// IsToday reports whether t is today.
func IsToday(t time.Time) bool {
	now := Now()
	y1, m1, d1 := t.In(now.Location()).Date()
	y2, m2, d2 := now.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// IsYesterday reports whether t is yesterday.
func IsYesterday(t time.Time) bool {
	return IsToday(t.AddDate(0, 0, 1))
}

// IsTomorrow reports whether t is tomorrow.
func IsTomorrow(t time.Time) bool {
	return IsToday(t.AddDate(0, 0, -1))
}

// IsCurrentWeek reports whether t falls in the current Monday-start week.
func IsCurrentWeek(t time.Time) bool {
	now := Now()
	return StartOfWeek(t.In(now.Location())).Equal(StartOfWeek(now))
}

// IsCurrentMonth reports whether t falls in the current calendar month.
func IsCurrentMonth(t time.Time) bool {
	now := Now()
	tt := t.In(now.Location())
	return tt.Year() == now.Year() && tt.Month() == now.Month()
}

// IsCurrentYear reports whether t falls in the current calendar year.
func IsCurrentYear(t time.Time) bool {
	now := Now()
	return t.In(now.Location()).Year() == now.Year()
}

// IsCurrentHour reports whether t falls in the current clock hour.
func IsCurrentHour(t time.Time) bool {
	now := Now()
	tt := t.In(now.Location())
	return IsSameDay(tt, now) && tt.Hour() == now.Hour()
}

// IsCurrentMinute reports whether t falls in the current clock minute.
func IsCurrentMinute(t time.Time) bool {
	now := Now()
	tt := t.In(now.Location())
	return IsCurrentHour(tt) && tt.Minute() == now.Minute()
}

// IsCurrentSecond reports whether t falls in the current clock second.
func IsCurrentSecond(t time.Time) bool {
	now := Now()
	tt := t.In(now.Location())
	return IsCurrentMinute(tt) && tt.Second() == now.Second()
}

// IsCurrentQuarter reports whether t falls in the current calendar quarter.
func IsCurrentQuarter(t time.Time) bool {
	now := Now()
	tt := t.In(now.Location())
	return tt.Year() == now.Year() && Quarter(tt) == Quarter(now)
}

// IsWeekend reports whether t falls on Saturday or Sunday.
func IsWeekend(t time.Time) bool {
	w := t.Weekday()
	return w == time.Saturday || w == time.Sunday
}

// IsWeekday reports whether t is Monday–Friday.
func IsWeekday(t time.Time) bool {
	return !IsWeekend(t)
}

// IsMonday reports whether t is Monday.
func IsMonday(t time.Time) bool { return t.Weekday() == time.Monday }

// IsTuesday reports whether t is Tuesday.
func IsTuesday(t time.Time) bool { return t.Weekday() == time.Tuesday }

// IsWednesday reports whether t is Wednesday.
func IsWednesday(t time.Time) bool { return t.Weekday() == time.Wednesday }

// IsThursday reports whether t is Thursday.
func IsThursday(t time.Time) bool { return t.Weekday() == time.Thursday }

// IsFriday reports whether t is Friday.
func IsFriday(t time.Time) bool { return t.Weekday() == time.Friday }

// IsSaturday reports whether t is Saturday.
func IsSaturday(t time.Time) bool { return t.Weekday() == time.Saturday }

// IsSunday reports whether t is Sunday.
func IsSunday(t time.Time) bool { return t.Weekday() == time.Sunday }

// Previous returns the previous occurrence of day (always in the past).
func Previous(t time.Time, day time.Weekday) time.Time {
	diff := int(t.Weekday() - day)
	if diff <= 0 {
		diff += 7
	}
	return t.AddDate(0, 0, -diff)
}

// Next returns the next occurrence of day (always in the future).
func Next(t time.Time, day time.Weekday) time.Time {
	diff := int(day - t.Weekday())
	if diff <= 0 {
		diff += 7
	}
	return t.AddDate(0, 0, diff)
}

// AddWeekdays advances t by n weekdays (Mon–Fri), skipping weekends.
func AddWeekdays(t time.Time, n int) time.Time {
	if n == 0 {
		return t
	}
	step := 1
	if n < 0 {
		step = -1
		n = -n
	}
	for n > 0 {
		t = t.AddDate(0, 0, step)
		if IsWeekday(t) {
			n--
		}
	}
	return t
}

// SubWeekdays moves t backward by n weekdays (Mon–Fri).
func SubWeekdays(t time.Time, n int) time.Time {
	return AddWeekdays(t, -n)
}

// IsSameDay reports whether a and b share the same calendar day in a's location.
func IsSameDay(a, b time.Time) bool {
	y1, m1, d1 := a.Date()
	y2, m2, d2 := b.In(a.Location()).Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// IsSameMonth reports whether a and b share the same calendar month and year in a's location.
func IsSameMonth(a, b time.Time) bool {
	y1, m1, _ := a.Date()
	y2, m2, _ := b.In(a.Location()).Date()
	return y1 == y2 && m1 == m2
}

// IsSameYear reports whether a and b share the same calendar year in a's location.
func IsSameYear(a, b time.Time) bool {
	return a.Year() == b.In(a.Location()).Year()
}

// IsSameHour reports whether a and b share the same calendar hour in a's location.
func IsSameHour(a, b time.Time) bool {
	bb := b.In(a.Location())
	return IsSameDay(a, bb) && a.Hour() == bb.Hour()
}

// IsSameMinute reports whether a and b share the same calendar minute in a's location.
func IsSameMinute(a, b time.Time) bool {
	bb := b.In(a.Location())
	return IsSameHour(a, bb) && a.Minute() == bb.Minute()
}

// IsSameSecond reports whether a and b share the same calendar second in a's location.
func IsSameSecond(a, b time.Time) bool {
	bb := b.In(a.Location())
	return IsSameMinute(a, bb) && a.Second() == bb.Second()
}

// IsSameMillisecond reports whether a and b share the same Unix millisecond.
func IsSameMillisecond(a, b time.Time) bool {
	return a.UnixMilli() == b.UnixMilli()
}

// IsSameWeek reports whether a and b fall in the same Monday-start week in a's location.
func IsSameWeek(a, b time.Time) bool {
	return StartOfWeek(a).Equal(StartOfWeek(b.In(a.Location())))
}

// DaysInMonth returns the number of days in t's month.
func DaysInMonth(t time.Time) int {
	return EndOfMonth(t).Day()
}

// IsFirstOfMonth reports whether t is the first day of its month.
func IsFirstOfMonth(t time.Time) bool {
	return t.Day() == 1
}

// IsLastOfMonth reports whether t is the last day of its month.
func IsLastOfMonth(t time.Time) bool {
	return t.Day() == DaysInMonth(t)
}

// IsStartOfDay reports whether t is at midnight (00:00:00.000000000) in its location.
func IsStartOfDay(t time.Time) bool {
	return t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 && t.Nanosecond() == 0
}

// IsEndOfDay reports whether t is the last nanosecond of its calendar day.
func IsEndOfDay(t time.Time) bool {
	return t.Equal(EndOfDay(t))
}

// IsStartOfWeek reports whether t is Monday 00:00:00.000000000 in its location.
func IsStartOfWeek(t time.Time) bool {
	return t.Equal(StartOfWeek(t))
}

// IsEndOfWeek reports whether t is the last nanosecond of Sunday in its location.
func IsEndOfWeek(t time.Time) bool {
	return t.Equal(EndOfWeek(t))
}

// IsStartOfHour reports whether t is at minute 0, second 0, nanosecond 0 of its hour.
func IsStartOfHour(t time.Time) bool {
	return t.Equal(StartOfHour(t))
}

// IsEndOfHour reports whether t is the last nanosecond of its hour.
func IsEndOfHour(t time.Time) bool {
	return t.Equal(EndOfHour(t))
}

// IsStartOfMonth reports whether t is the first moment of its calendar month.
func IsStartOfMonth(t time.Time) bool {
	return t.Equal(StartOfMonth(t))
}

// IsEndOfMonth reports whether t is the last nanosecond of its calendar month.
func IsEndOfMonth(t time.Time) bool {
	return t.Equal(EndOfMonth(t))
}

// IsStartOfYear reports whether t is January 1 00:00:00.000000000 in its location.
func IsStartOfYear(t time.Time) bool {
	return t.Equal(StartOfYear(t))
}

// IsEndOfYear reports whether t is the last nanosecond of December 31 in its location.
func IsEndOfYear(t time.Time) bool {
	return t.Equal(EndOfYear(t))
}

// IsNoon reports whether t is at 12:00:00.000000000 in its location.
func IsNoon(t time.Time) bool {
	return t.Hour() == 12 && t.Minute() == 0 && t.Second() == 0 && t.Nanosecond() == 0
}

// Quarter returns the calendar quarter of t (1-4).
func Quarter(t time.Time) int {
	return (int(t.Month())-1)/3 + 1
}

// IsSameQuarter reports whether a and b share the same calendar quarter and year in a's location.
func IsSameQuarter(a, b time.Time) bool {
	bb := b.In(a.Location())
	return a.Year() == bb.Year() && Quarter(a) == Quarter(bb)
}

// IsLeapYear reports whether t's calendar year is a leap year.
func IsLeapYear(t time.Time) bool {
	y := t.Year()
	return y%4 == 0 && (y%100 != 0 || y%400 == 0)
}

// IsPast reports whether t is before now.
func IsPast(t time.Time) bool {
	return t.Before(Now())
}

// IsFuture reports whether t is after now.
func IsFuture(t time.Time) bool {
	return t.After(Now())
}

// DiffInDays returns whole-day difference (absolute).
func DiffInDays(a, b time.Time) int {
	if a.After(b) {
		a, b = b, a
	}
	return int(b.Sub(a).Hours() / 24)
}

// DiffInWeekdays returns absolute weekday (Mon–Fri) count between start-of-day a and b
// (exclusive of the later calendar day).
func DiffInWeekdays(a, b time.Time) int {
	if a.After(b) {
		a, b = b, a
	}
	a = StartOfDay(a)
	b = StartOfDay(b)
	count := 0
	for d := a; d.Before(b); d = d.AddDate(0, 0, 1) {
		if IsWeekday(d) {
			count++
		}
	}
	return count
}

// DiffInWeeks returns whole-week difference (absolute).
func DiffInWeeks(a, b time.Time) int {
	return DiffInDays(a, b) / 7
}

// DiffInMonths returns whole calendar-month difference (absolute).
func DiffInMonths(a, b time.Time) int {
	if a.After(b) {
		a, b = b, a
	}
	months := (b.Year()-a.Year())*12 + int(b.Month()-a.Month())
	if b.Day() < a.Day() {
		months--
	}
	if months < 0 {
		return 0
	}
	return months
}

// DiffInQuarters returns whole calendar-quarter difference (absolute).
func DiffInQuarters(a, b time.Time) int {
	if a.After(b) {
		a, b = b, a
	}
	return (b.Year()-a.Year())*4 + (Quarter(b) - Quarter(a))
}

// DiffInYears returns whole calendar-year difference (absolute).
func DiffInYears(a, b time.Time) int {
	return DiffInMonths(a, b) / 12
}

// Age returns whole years from birth to Now (respects Pretend).
func Age(birth time.Time) int {
	return DiffInYears(birth, Now())
}

// IsBirthday reports whether t's month/day matches today (respects Pretend).
func IsBirthday(t time.Time) bool {
	now := Now()
	_, m1, d1 := t.In(now.Location()).Date()
	_, m2, d2 := now.Date()
	return m1 == m2 && d1 == d2
}

// DiffInHours returns whole-hour difference (absolute).
func DiffInHours(a, b time.Time) int {
	if a.After(b) {
		a, b = b, a
	}
	return int(b.Sub(a).Hours())
}

// DiffInMinutes returns whole-minute difference (absolute).
func DiffInMinutes(a, b time.Time) int {
	if a.After(b) {
		a, b = b, a
	}
	return int(b.Sub(a).Minutes())
}

// DiffInSeconds returns whole-second difference (absolute).
func DiffInSeconds(a, b time.Time) int64 {
	if a.After(b) {
		a, b = b, a
	}
	return int64(b.Sub(a) / time.Second)
}

// DiffInMilliseconds returns whole-millisecond difference (absolute).
func DiffInMilliseconds(a, b time.Time) int64 {
	if a.After(b) {
		a, b = b, a
	}
	return b.Sub(a).Milliseconds()
}

// DiffInMicroseconds returns whole-microsecond difference (absolute).
func DiffInMicroseconds(a, b time.Time) int64 {
	if a.After(b) {
		a, b = b, a
	}
	return b.Sub(a).Microseconds()
}

// DiffInNanoseconds returns whole-nanosecond difference (absolute).
func DiffInNanoseconds(a, b time.Time) int64 {
	if a.After(b) {
		a, b = b, a
	}
	return b.Sub(a).Nanoseconds()
}

// HumanDiff returns a short relative phrase for t vs now.
func HumanDiff(t time.Time) string {
	now := Now()
	d := now.Sub(t)
	future := false
	if d < 0 {
		future = true
		d = -d
	}
	locale := HumanLocale()
	var phrase string
	switch {
	case d < time.Minute:
		if future {
			if locale == "tr" {
				return "birazdan"
			}
			return "in a moment"
		}
		if locale == "tr" {
			return "az önce"
		}
		return "just now"
	case d < time.Hour:
		mins := int(d.Minutes())
		if locale == "tr" {
			phrase = fmt.Sprintf("%d dakika", mins)
		} else {
			phrase = fmt.Sprintf("%d minutes", mins)
		}
	case d < 24*time.Hour:
		hours := int(d.Hours())
		if locale == "tr" {
			phrase = fmt.Sprintf("%d saat", hours)
		} else {
			phrase = fmt.Sprintf("%d hours", hours)
		}
	case d < 30*24*time.Hour:
		days := int(d.Hours() / 24)
		if locale == "tr" {
			phrase = fmt.Sprintf("%d gün", days)
		} else {
			phrase = fmt.Sprintf("%d days", days)
		}
	default:
		months := int(d.Hours() / 24 / 30)
		if months < 1 {
			months = 1
		}
		if locale == "tr" {
			phrase = fmt.Sprintf("%d ay", months)
		} else {
			phrase = fmt.Sprintf("%d months", months)
		}
	}
	if future {
		if locale == "tr" {
			return phrase + " sonra"
		}
		return "in " + phrase
	}
	if locale == "tr" {
		return phrase + " önce"
	}
	return phrase + " ago"
}

// Between reports whether t is within [start, end].
func Between(t, start, end time.Time) bool {
	return (t.Equal(start) || t.After(start)) && (t.Equal(end) || t.Before(end))
}

// BetweenExcluded reports whether t is within (start, end), exclusive of bounds.
func BetweenExcluded(t, start, end time.Time) bool {
	return t.After(start) && t.Before(end)
}

// IsMidnight reports whether t is at midnight (alias of IsStartOfDay).
func IsMidnight(t time.Time) bool {
	return IsStartOfDay(t)
}

// IsStartOfMinute reports whether t is at second 0, nanosecond 0 of its minute.
func IsStartOfMinute(t time.Time) bool {
	return t.Equal(StartOfMinute(t))
}

// IsEndOfMinute reports whether t is the last nanosecond of its minute.
func IsEndOfMinute(t time.Time) bool {
	return t.Equal(EndOfMinute(t))
}

// IsStartOfSecond reports whether t is at nanosecond 0 of its second.
func IsStartOfSecond(t time.Time) bool {
	return t.Equal(StartOfSecond(t))
}

// IsEndOfSecond reports whether t is the last nanosecond of its second.
func IsEndOfSecond(t time.Time) bool {
	return t.Equal(EndOfSecond(t))
}

// IsStartOfQuarter reports whether t is the first moment of its calendar quarter.
func IsStartOfQuarter(t time.Time) bool {
	return t.Equal(StartOfQuarter(t))
}

// IsEndOfQuarter reports whether t is the last nanosecond of its calendar quarter.
func IsEndOfQuarter(t time.Time) bool {
	return t.Equal(EndOfQuarter(t))
}

// IsFirstOfYear reports whether t is January 1 (any clock time).
func IsFirstOfYear(t time.Time) bool {
	return t.Month() == time.January && t.Day() == 1
}

// IsLastOfYear reports whether t is December 31 (any clock time).
func IsLastOfYear(t time.Time) bool {
	return t.Month() == time.December && t.Day() == 31
}

// IsFirstOfQuarter reports whether t is the first calendar day of its quarter.
func IsFirstOfQuarter(t time.Time) bool {
	return IsSameDay(t, StartOfQuarter(t))
}

// IsLastOfQuarter reports whether t is the last calendar day of its quarter.
func IsLastOfQuarter(t time.Time) bool {
	return IsSameDay(t, EndOfQuarter(t))
}

// Today returns start of today (respects Pretend).
func Today() time.Time {
	return StartOfDay(Now())
}

// Yesterday returns start of yesterday (respects Pretend).
func Yesterday() time.Time {
	return StartOfDay(Now().AddDate(0, 0, -1))
}

// Tomorrow returns start of tomorrow (respects Pretend).
func Tomorrow() time.Time {
	return StartOfDay(Now().AddDate(0, 0, 1))
}

// Create builds a time in the given location (Local when loc is nil).
func Create(year int, month time.Month, day, hour, min, sec, nsec int, loc ...*time.Location) time.Time {
	l := time.Local
	if len(loc) > 0 && loc[0] != nil {
		l = loc[0]
	}
	return time.Date(year, month, day, hour, min, sec, nsec, l)
}

// CreateFromDate builds a midnight time for the given calendar date.
func CreateFromDate(year int, month time.Month, day int, loc ...*time.Location) time.Time {
	return Create(year, month, day, 0, 0, 0, 0, loc...)
}

// CreateFromTime builds today's date with the given clock (respects Pretend).
func CreateFromTime(hour, min, sec, nsec int, loc ...*time.Location) time.Time {
	base := Now()
	if len(loc) > 0 && loc[0] != nil {
		base = base.In(loc[0])
	}
	y, m, d := base.Date()
	return Create(y, m, d, hour, min, sec, nsec, base.Location())
}

// MustParse parses value or panics.
func MustParse(value string) time.Time {
	t, err := Parse(value)
	if err != nil {
		panic(err)
	}
	return t
}

// ParseFormat parses value with an explicit layout.
func ParseFormat(layout, value string) (time.Time, error) {
	if t, err := time.Parse(layout, value); err == nil {
		return t, nil
	}
	return time.ParseInLocation(layout, value, time.Local)
}

// AddWeeks adds weeks (7-day units) to t.
func AddWeeks(t time.Time, weeks int) time.Time {
	return AddDays(t, weeks*7)
}

// SubWeeks subtracts weeks from t.
func SubWeeks(t time.Time, weeks int) time.Time {
	return AddWeeks(t, -weeks)
}

// AddMilliseconds adds milliseconds to t.
func AddMilliseconds(t time.Time, n int) time.Time {
	return t.Add(time.Duration(n) * time.Millisecond)
}

// SubMilliseconds subtracts milliseconds from t.
func SubMilliseconds(t time.Time, n int) time.Time {
	return AddMilliseconds(t, -n)
}

// AddMicroseconds adds microseconds to t.
func AddMicroseconds(t time.Time, n int) time.Time {
	return t.Add(time.Duration(n) * time.Microsecond)
}

// SubMicroseconds subtracts microseconds from t.
func SubMicroseconds(t time.Time, n int) time.Time {
	return AddMicroseconds(t, -n)
}

// AddNanoseconds adds nanoseconds to t.
func AddNanoseconds(t time.Time, n int) time.Time {
	return t.Add(time.Duration(n) * time.Nanosecond)
}

// SubNanoseconds subtracts nanoseconds from t.
func SubNanoseconds(t time.Time, n int) time.Time {
	return AddNanoseconds(t, -n)
}

// AddDuration adds an arbitrary duration to t.
func AddDuration(t time.Time, d time.Duration) time.Time {
	return t.Add(d)
}

// SubDuration subtracts an arbitrary duration from t.
func SubDuration(t time.Time, d time.Duration) time.Time {
	return t.Add(-d)
}

// SetYear sets the year, keeping month/day/clock when possible.
func SetYear(t time.Time, year int) time.Time {
	return time.Date(year, t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}

// SetMonth sets the month, keeping day/clock when possible.
func SetMonth(t time.Time, month time.Month) time.Time {
	return time.Date(t.Year(), month, t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}

// SetDay sets the day of month, keeping clock.
func SetDay(t time.Time, day int) time.Time {
	return time.Date(t.Year(), t.Month(), day, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}

// SetHour sets the hour.
func SetHour(t time.Time, hour int) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), hour, t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}

// SetMinute sets the minute.
func SetMinute(t time.Time, minute int) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), minute, t.Second(), t.Nanosecond(), t.Location())
}

// SetSecond sets the second.
func SetSecond(t time.Time, second int) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), second, t.Nanosecond(), t.Location())
}

// SetMillisecond sets the millisecond component (0–999), clearing finer precision.
func SetMillisecond(t time.Time, ms int) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), ms*int(time.Millisecond), t.Location())
}

// SetMicrosecond sets the microsecond component (0–999999), clearing finer precision.
func SetMicrosecond(t time.Time, us int) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), us*int(time.Microsecond), t.Location())
}

// Year returns t's year.
func Year(t time.Time) int { return t.Year() }

// Month returns t's month.
func Month(t time.Time) time.Month { return t.Month() }

// Day returns t's day of month.
func Day(t time.Time) int { return t.Day() }

// Hour returns t's hour.
func Hour(t time.Time) int { return t.Hour() }

// Minute returns t's minute.
func Minute(t time.Time) int { return t.Minute() }

// Second returns t's second.
func Second(t time.Time) int { return t.Second() }

// Millisecond returns t's millisecond component (0–999).
func Millisecond(t time.Time) int {
	return t.Nanosecond() / int(time.Millisecond)
}

// Microsecond returns t's microsecond component (0–999999).
func Microsecond(t time.Time) int {
	return t.Nanosecond() / int(time.Microsecond)
}

// Nanosecond returns t's nanosecond component (0–999999999).
func Nanosecond(t time.Time) int { return t.Nanosecond() }

// DayOfYear returns the 1–366 day-of-year.
func DayOfYear(t time.Time) int {
	return t.YearDay()
}

// WeekOfYear returns the ISO week number (1–53).
func WeekOfYear(t time.Time) int {
	_, w := t.ISOWeek()
	return w
}

// IsoWeekYear returns the ISO week-year.
func IsoWeekYear(t time.Time) int {
	y, _ := t.ISOWeek()
	return y
}

// DaysInYear returns 366 for leap years, otherwise 365.
func DaysInYear(t time.Time) int {
	if IsLeapYear(t) {
		return 366
	}
	return 365
}

// IsoWeekday returns ISO weekday (Monday=1 … Sunday=7).
func IsoWeekday(t time.Time) int {
	w := int(t.Weekday())
	if w == 0 {
		return 7
	}
	return w
}

// Decade returns the decade start year (e.g. 2026 → 2020).
func Decade(t time.Time) int {
	return (t.Year() / 10) * 10
}

// Century returns the century number (e.g. 2026 → 21).
func Century(t time.Time) int {
	y := t.Year()
	if y <= 0 {
		return y/100 - 1
	}
	return (y-1)/100 + 1
}

// StartOfDecade returns January 1 00:00 of the decade.
func StartOfDecade(t time.Time) time.Time {
	return time.Date(Decade(t), time.January, 1, 0, 0, 0, 0, t.Location())
}

// EndOfDecade returns December 31 end-of-day of the decade's last year.
func EndOfDecade(t time.Time) time.Time {
	return EndOfDay(time.Date(Decade(t)+9, time.December, 31, 0, 0, 0, 0, t.Location()))
}

// IsStartOfDecade reports whether t is the first moment of its decade.
func IsStartOfDecade(t time.Time) bool {
	return t.Equal(StartOfDecade(t))
}

// IsEndOfDecade reports whether t is the last nanosecond of its decade.
func IsEndOfDecade(t time.Time) bool {
	return t.Equal(EndOfDecade(t))
}

// ToDateString formats as YYYY-MM-DD.
func ToDateString(t time.Time) string {
	return t.Format("2006-01-02")
}

// ToDateTimeString formats as YYYY-MM-DD HH:MM:SS.
func ToDateTimeString(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// ToTimeString formats as HH:MM:SS.
func ToTimeString(t time.Time) string {
	return t.Format("15:04:05")
}

// ToIso8601String formats as RFC3339.
func ToIso8601String(t time.Time) string {
	return t.Format(time.RFC3339)
}

// IsLastWeek reports whether t falls in the previous Monday-start week (respects Pretend).
func IsLastWeek(t time.Time) bool {
	return IsSameWeek(t, Now().AddDate(0, 0, -7))
}

// IsNextWeek reports whether t falls in the next Monday-start week (respects Pretend).
func IsNextWeek(t time.Time) bool {
	return IsSameWeek(t, Now().AddDate(0, 0, 7))
}

// IsLastMonth reports whether t falls in the previous calendar month (respects Pretend).
func IsLastMonth(t time.Time) bool {
	now := Now()
	prev := AddMonths(StartOfMonth(now), -1)
	tt := t.In(now.Location())
	return tt.Year() == prev.Year() && tt.Month() == prev.Month()
}

// IsNextMonth reports whether t falls in the next calendar month (respects Pretend).
func IsNextMonth(t time.Time) bool {
	now := Now()
	next := AddMonths(StartOfMonth(now), 1)
	tt := t.In(now.Location())
	return tt.Year() == next.Year() && tt.Month() == next.Month()
}

// IsLastYear reports whether t falls in the previous calendar year (respects Pretend).
func IsLastYear(t time.Time) bool {
	return t.In(Now().Location()).Year() == Now().Year()-1
}

// IsNextYear reports whether t falls in the next calendar year (respects Pretend).
func IsNextYear(t time.Time) bool {
	return t.In(Now().Location()).Year() == Now().Year()+1
}

// IsLastQuarter reports whether t falls in the previous calendar quarter (respects Pretend).
func IsLastQuarter(t time.Time) bool {
	now := Now()
	prev := AddQuarters(StartOfQuarter(now), -1)
	tt := t.In(now.Location())
	return tt.Year() == prev.Year() && Quarter(tt) == Quarter(prev)
}

// IsNextQuarter reports whether t falls in the next calendar quarter (respects Pretend).
func IsNextQuarter(t time.Time) bool {
	now := Now()
	next := AddQuarters(StartOfQuarter(now), 1)
	tt := t.In(now.Location())
	return tt.Year() == next.Year() && Quarter(tt) == Quarter(next)
}

// NotEqual reports whether a and b are different instants.
func NotEqual(a, b time.Time) bool {
	return !a.Equal(b)
}

// IsZero reports whether t is the zero time.
func IsZero(t time.Time) bool {
	return t.IsZero()
}

// FloatDiffInDays returns absolute day difference as a float (Sub/24h).
func FloatDiffInDays(a, b time.Time) float64 {
	d := b.Sub(a)
	if d < 0 {
		d = -d
	}
	return float64(d) / float64(24*time.Hour)
}

// FloatDiffInHours returns absolute hour difference as a float.
func FloatDiffInHours(a, b time.Time) float64 {
	d := b.Sub(a)
	if d < 0 {
		d = -d
	}
	return d.Hours()
}

// FloatDiffInMinutes returns absolute minute difference as a float.
func FloatDiffInMinutes(a, b time.Time) float64 {
	d := b.Sub(a)
	if d < 0 {
		d = -d
	}
	return d.Minutes()
}

// FloatDiffInSeconds returns absolute second difference as a float.
func FloatDiffInSeconds(a, b time.Time) float64 {
	d := b.Sub(a)
	if d < 0 {
		d = -d
	}
	return d.Seconds()
}

// InTimezone returns t in the named location (Local on error).
func InTimezone(t time.Time, name string) time.Time {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return t.In(time.Local)
	}
	return t.In(loc)
}

// AsUTC returns t in UTC.
func AsUTC(t time.Time) time.Time {
	return t.UTC()
}

// AsLocal returns t in Local.
func AsLocal(t time.Time) time.Time {
	return t.In(time.Local)
}

// DaysUntil returns whole days from now until t (negative when past; respects Pretend).
func DaysUntil(t time.Time) int {
	now := Now()
	if t.Before(now) {
		return -DiffInDays(t, now)
	}
	return DiffInDays(now, t)
}

// DaysSince returns whole days from t until now (negative when future; respects Pretend).
func DaysSince(t time.Time) int {
	return -DaysUntil(t)
}

// IsDayOfWeek reports whether t falls on the given weekday.
func IsDayOfWeek(t time.Time, day time.Weekday) bool {
	return t.Weekday() == day
}

// Diff returns the signed duration b - a.
func Diff(a, b time.Time) time.Duration {
	return b.Sub(a)
}

// DiffInDaysSigned returns whole calendar-day difference b - a (may be negative).
func DiffInDaysSigned(a, b time.Time) int {
	if b.Before(a) {
		return -DiffInDays(b, a)
	}
	return DiffInDays(a, b)
}

// DiffInHoursSigned returns whole-hour difference b - a (may be negative).
func DiffInHoursSigned(a, b time.Time) int {
	return int(b.Sub(a).Hours())
}

// DiffInMinutesSigned returns whole-minute difference b - a (may be negative).
func DiffInMinutesSigned(a, b time.Time) int {
	return int(b.Sub(a).Minutes())
}

// DiffInSecondsSigned returns whole-second difference b - a (may be negative).
func DiffInSecondsSigned(a, b time.Time) int64 {
	return int64(b.Sub(a) / time.Second)
}

// StartOfCentury returns January 1 00:00 of the century's first year.
func StartOfCentury(t time.Time) time.Time {
	startYear := (Century(t)-1)*100 + 1
	return time.Date(startYear, time.January, 1, 0, 0, 0, 0, t.Location())
}

// EndOfCentury returns December 31 end-of-day of the century's last year.
func EndOfCentury(t time.Time) time.Time {
	endYear := Century(t) * 100
	return EndOfDay(time.Date(endYear, time.December, 31, 0, 0, 0, 0, t.Location()))
}

// IsStartOfCentury reports whether t is the first moment of its century.
func IsStartOfCentury(t time.Time) bool {
	return t.Equal(StartOfCentury(t))
}

// IsEndOfCentury reports whether t is the last nanosecond of its century.
func IsEndOfCentury(t time.Time) bool {
	return t.Equal(EndOfCentury(t))
}

// SetTimezone returns t with the named location, keeping the wall-clock fields.
func SetTimezone(t time.Time, name string) time.Time {
	loc, err := time.LoadLocation(name)
	if err != nil {
		loc = time.Local
	}
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
}

// IsUTC reports whether t's location is UTC.
func IsUTC(t time.Time) bool {
	return t.Location() == time.UTC || t.Location().String() == "UTC"
}

// IsLocal reports whether t's location is Local.
func IsLocal(t time.Time) bool {
	return t.Location() == time.Local || t.Location().String() == time.Local.String()
}

// DiffInWeekendDays returns absolute weekend-day count between start-of-day a and b
// (exclusive of the later calendar day).
func DiffInWeekendDays(a, b time.Time) int {
	if a.After(b) {
		a, b = b, a
	}
	a = StartOfDay(a)
	b = StartOfDay(b)
	count := 0
	for d := a; d.Before(b); d = d.AddDate(0, 0, 1) {
		if IsWeekend(d) {
			count++
		}
	}
	return count
}
