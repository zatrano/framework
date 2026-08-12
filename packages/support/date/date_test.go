package date_test

import (
	"testing"
	"time"

	"github.com/zatrano/framework/packages/support/date"
)

func TestPretendAndWeekHelpers(t *testing.T) {
	defer date.Restore()
	fixed := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	date.Pretend(fixed)
	if !date.Now().Equal(fixed.In(time.Local)) && date.Now().UTC().Format(time.RFC3339) != fixed.Format(time.RFC3339) {
		// compare via UTC unix
		if date.UTC().Unix() != fixed.Unix() {
			t.Fatalf("now=%v", date.Now())
		}
	}
	if !date.IsPast(fixed.Add(-time.Hour)) || !date.IsFuture(fixed.Add(time.Hour)) {
		t.Fatal("past/future")
	}
	start := date.StartOfWeek(fixed)
	if start.Weekday() != time.Monday {
		t.Fatalf("start weekday=%v", start.Weekday())
	}
	if date.AddYears(fixed, 1).Year() != 2027 {
		t.Fatal("add years")
	}
	if !date.IsToday(fixed) || !date.IsYesterday(fixed.AddDate(0, 0, -1)) || !date.IsTomorrow(fixed.AddDate(0, 0, 1)) {
		t.Fatal("today/yesterday/tomorrow")
	}
}

func TestMonthWeekendSameDayAndDiff(t *testing.T) {
	fixed := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC) // Monday
	start := date.StartOfMonth(fixed)
	end := date.EndOfMonth(fixed)
	if start.Day() != 1 || start.Hour() != 0 {
		t.Fatalf("start of month=%v", start)
	}
	if end.Day() != 31 || end.Hour() != 23 {
		t.Fatalf("end of month=%v", end)
	}
	if date.IsWeekend(fixed) || !date.IsWeekday(fixed) {
		t.Fatal("weekday monday")
	}
	saturday := time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC)
	if !date.IsWeekend(saturday) || date.IsWeekday(saturday) {
		t.Fatal("weekend saturday")
	}
}

func TestNamedWeekdays(t *testing.T) {
	// 2026-08-01 Saturday … 2026-08-07 Friday
	days := []struct {
		day time.Time
		ok  func(time.Time) bool
	}{
		{time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC), date.IsMonday},
		{time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC), date.IsTuesday},
		{time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC), date.IsWednesday},
		{time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC), date.IsThursday},
		{time.Date(2026, 8, 7, 0, 0, 0, 0, time.UTC), date.IsFriday},
		{time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), date.IsSaturday},
		{time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC), date.IsSunday},
	}
	for _, d := range days {
		if !d.ok(d.day) {
			t.Fatalf("expected true for %v", d.day.Weekday())
		}
	}
	monday := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	if date.IsTuesday(monday) || date.IsSunday(monday) {
		t.Fatal("monday mismatches")
	}
}

func TestPreviousNextWeekdays(t *testing.T) {
	monday := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	prevMon := date.Previous(monday, time.Monday)
	if prevMon.Format("2006-01-02") != "2026-07-27" {
		t.Fatalf("previous monday=%v", prevMon)
	}
	nextMon := date.Next(monday, time.Monday)
	if nextMon.Format("2006-01-02") != "2026-08-10" {
		t.Fatalf("next monday=%v", nextMon)
	}
	if got := date.Previous(monday, time.Friday).Format("2006-01-02"); got != "2026-07-31" {
		t.Fatalf("previous friday=%s", got)
	}
	if got := date.Next(monday, time.Friday).Format("2006-01-02"); got != "2026-08-07" {
		t.Fatalf("next friday=%s", got)
	}
	friday := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	if got := date.AddWeekdays(friday, 1).Format("2006-01-02"); got != "2026-08-10" {
		t.Fatalf("add weekdays=%s", got)
	}
	if got := date.SubWeekdays(monday, 1).Format("2006-01-02"); got != "2026-07-31" {
		t.Fatalf("sub weekdays=%s", got)
	}
	if got := date.AddWeekdays(monday, 0); !got.Equal(monday) {
		t.Fatal("add weekdays zero")
	}
}

func TestNoonHourAndDiffWeekdays(t *testing.T) {
	base := time.Date(2026, 8, 3, 15, 30, 45, 0, time.UTC) // Monday
	noon := date.Noon(base)
	if noon.Hour() != 12 || noon.Minute() != 0 || noon.Day() != 3 {
		t.Fatalf("noon=%v", noon)
	}
	start := date.StartOfHour(base)
	if start.Minute() != 0 || start.Second() != 0 || start.Hour() != 15 {
		t.Fatalf("start of hour=%v", start)
	}
	end := date.EndOfHour(base)
	if end.Minute() != 59 || end.Second() != 59 || end.Hour() != 15 {
		t.Fatalf("end of hour=%v", end)
	}
	friday := time.Date(2026, 8, 7, 9, 0, 0, 0, time.UTC)
	if got := date.DiffInWeekdays(base, friday); got != 4 {
		t.Fatalf("diff weekdays=%d", got)
	}
	nextMon := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	if got := date.DiffInWeekdays(base, nextMon); got != 5 {
		t.Fatalf("diff weekdays week=%d", got)
	}
	if date.DiffInWeekdays(friday, base) != 4 {
		t.Fatal("diff weekdays absolute")
	}
}

func TestStartEndOfMinuteAndSecond(t *testing.T) {
	base := time.Date(2026, 8, 3, 15, 30, 45, 123456789, time.UTC)
	sm := date.StartOfMinute(base)
	if sm.Second() != 0 || sm.Nanosecond() != 0 || sm.Minute() != 30 || sm.Hour() != 15 {
		t.Fatalf("start of minute=%v", sm)
	}
	em := date.EndOfMinute(base)
	if em.Second() != 59 || em.Nanosecond() != int(time.Second-time.Nanosecond) || em.Minute() != 30 {
		t.Fatalf("end of minute=%v", em)
	}
	ss := date.StartOfSecond(base)
	if ss.Nanosecond() != 0 || ss.Second() != 45 {
		t.Fatalf("start of second=%v", ss)
	}
	es := date.EndOfSecond(base)
	if es.Second() != 45 || es.Nanosecond() != int(time.Second-time.Nanosecond) {
		t.Fatalf("end of second=%v", es)
	}
	if !date.IsStartOfHour(date.StartOfHour(base)) || date.IsStartOfHour(base) {
		t.Fatal("IsStartOfHour")
	}
	if !date.IsEndOfHour(date.EndOfHour(base)) || date.IsEndOfHour(base) {
		t.Fatal("IsEndOfHour")
	}
	monday := date.StartOfWeek(base) // 2026-08-03 is Monday
	if !date.IsStartOfWeek(monday) || date.IsStartOfWeek(base) {
		t.Fatal("IsStartOfWeek")
	}
	sundayEnd := date.EndOfWeek(base)
	if !date.IsEndOfWeek(sundayEnd) || date.IsEndOfWeek(base) {
		t.Fatal("IsEndOfWeek")
	}
}

func TestClosestAndTimestamps(t *testing.T) {
	needle := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	a := time.Date(2026, 8, 3, 10, 0, 0, 0, time.UTC)
	b := time.Date(2026, 8, 3, 12, 30, 0, 0, time.UTC)
	c := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	if got := date.Closest(needle, a, b, c); !got.Equal(b) {
		t.Fatalf("closest=%v", got)
	}
	if !date.Closest(needle).IsZero() {
		t.Fatal("closest empty")
	}
	if got := date.Farthest(needle, a, b, c); !got.Equal(c) {
		t.Fatalf("farthest=%v", got)
	}
	if !date.Farthest(needle).IsZero() {
		t.Fatal("farthest empty")
	}
	sec := date.Timestamp(needle)
	if !date.FromTimestamp(sec).UTC().Equal(needle) {
		t.Fatalf("from timestamp=%v sec=%d", date.FromTimestamp(sec).UTC(), sec)
	}
	ms := date.TimestampMilli(needle)
	if !date.FromTimestampMilli(ms).UTC().Equal(needle) {
		t.Fatalf("from milli=%v ms=%d", date.FromTimestampMilli(ms).UTC(), ms)
	}
	us := date.TimestampMicro(needle)
	if !date.FromTimestampMicro(us).UTC().Equal(needle) {
		t.Fatalf("from micro=%v us=%d", date.FromTimestampMicro(us).UTC(), us)
	}
	ns := date.TimestampNano(needle)
	if !date.FromTimestampNano(ns).UTC().Equal(needle) {
		t.Fatalf("from nano=%v ns=%d", date.FromTimestampNano(ns).UTC(), ns)
	}
}

func TestIsCurrentPeriodHelpers(t *testing.T) {
	date.Pretend(time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC))
	defer date.Restore()

	sameWeek := time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC)   // Monday
	otherWeek := time.Date(2026, 8, 10, 9, 0, 0, 0, time.UTC) // next Monday
	if !date.IsCurrentWeek(sameWeek) || date.IsCurrentWeek(otherWeek) {
		t.Fatal("is current week")
	}
	if !date.IsCurrentMonth(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)) ||
		date.IsCurrentMonth(time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("is current month")
	}
	if !date.IsCurrentYear(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)) ||
		date.IsCurrentYear(time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("is current year")
	}
	if !date.IsCurrentHour(time.Date(2026, 8, 5, 12, 30, 0, 0, time.UTC)) ||
		date.IsCurrentHour(time.Date(2026, 8, 5, 11, 59, 0, 0, time.UTC)) {
		t.Fatal("is current hour")
	}
	if !date.IsCurrentMinute(time.Date(2026, 8, 5, 12, 0, 30, 0, time.UTC)) ||
		date.IsCurrentMinute(time.Date(2026, 8, 5, 12, 1, 0, 0, time.UTC)) {
		t.Fatal("is current minute")
	}
	if !date.IsCurrentSecond(time.Date(2026, 8, 5, 12, 0, 0, 500, time.UTC)) ||
		date.IsCurrentSecond(time.Date(2026, 8, 5, 12, 0, 1, 0, time.UTC)) {
		t.Fatal("is current second")
	}
	if !date.IsCurrentQuarter(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)) ||
		date.IsCurrentQuarter(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("is current quarter")
	}
	a := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	b := a.Add(1500 * time.Millisecond)
	if date.DiffInMilliseconds(a, b) != 1500 {
		t.Fatalf("diff ms=%d", date.DiffInMilliseconds(a, b))
	}
	c := a.Add(2500 * time.Microsecond)
	if date.DiffInMicroseconds(a, c) != 2500 {
		t.Fatalf("diff us=%d", date.DiffInMicroseconds(a, c))
	}
	d := a.Add(3500 * time.Nanosecond)
	if date.DiffInNanoseconds(a, d) != 3500 {
		t.Fatalf("diff ns=%d", date.DiffInNanoseconds(a, d))
	}
}

func TestIsSameDayHelpers(t *testing.T) {
	fixed := time.Date(2026, 8, 3, 10, 0, 0, 0, time.UTC)
	same := time.Date(2026, 8, 3, 23, 59, 0, 0, time.UTC)
	other := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	if !date.IsSameDay(fixed, same) || date.IsSameDay(fixed, other) {
		t.Fatal("is same day")
	}
	later := fixed.Add(2*time.Hour + 30*time.Minute)
	if date.DiffInHours(fixed, later) != 2 {
		t.Fatalf("diff hours=%d", date.DiffInHours(fixed, later))
	}
	if date.DiffInMinutes(fixed, later) != 150 {
		t.Fatalf("diff minutes=%d", date.DiffInMinutes(fixed, later))
	}
}

func TestIsSameHourMinuteSecondWeek(t *testing.T) {
	base := time.Date(2026, 8, 3, 12, 30, 45, 0, time.UTC) // Monday
	if !date.IsSameHour(base, base.Add(20*time.Minute)) || date.IsSameHour(base, base.Add(time.Hour)) {
		t.Fatal("is same hour")
	}
	if !date.IsSameMinute(base, base.Add(10*time.Second)) || date.IsSameMinute(base, base.Add(time.Minute)) {
		t.Fatal("is same minute")
	}
	if !date.IsSameSecond(base, base.Add(100*time.Millisecond)) || date.IsSameSecond(base, base.Add(time.Second)) {
		t.Fatal("is same second")
	}
	if !date.IsSameMillisecond(base, base.Add(100*time.Microsecond)) || date.IsSameMillisecond(base, base.Add(time.Millisecond)) {
		t.Fatal("is same millisecond")
	}
	sameWeek := time.Date(2026, 8, 7, 9, 0, 0, 0, time.UTC)   // Friday
	otherWeek := time.Date(2026, 8, 10, 9, 0, 0, 0, time.UTC) // next Monday
	if !date.IsSameWeek(base, sameWeek) || date.IsSameWeek(base, otherWeek) {
		t.Fatal("is same week")
	}
}

func TestYearLeapAndSeconds(t *testing.T) {
	fixed := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC) // leap year
	start := date.StartOfYear(fixed)
	end := date.EndOfYear(fixed)
	if start.Month() != time.January || start.Day() != 1 || start.Hour() != 0 {
		t.Fatalf("start of year=%v", start)
	}
	if end.Month() != time.December || end.Day() != 31 || end.Hour() != 23 {
		t.Fatalf("end of year=%v", end)
	}
	if !date.IsLeapYear(fixed) || date.IsLeapYear(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("leap year")
	}
	later := fixed.Add(90 * time.Second)
	if date.DiffInSeconds(fixed, later) != 90 {
		t.Fatalf("diff seconds=%d", date.DiffInSeconds(fixed, later))
	}
}

func TestSameMonthYearDaysAndQuarter(t *testing.T) {
	a := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	sameMonth := time.Date(2026, 8, 31, 1, 0, 0, 0, time.UTC)
	otherMonth := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	otherYear := time.Date(2025, 8, 3, 0, 0, 0, 0, time.UTC)
	if !date.IsSameMonth(a, sameMonth) || date.IsSameMonth(a, otherMonth) {
		t.Fatal("same month")
	}
	if !date.IsSameYear(a, otherMonth) || date.IsSameYear(a, otherYear) {
		t.Fatal("same year")
	}
	if date.DaysInMonth(a) != 31 || date.DaysInMonth(time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)) != 29 {
		t.Fatal("days in month")
	}
	if !date.IsFirstOfMonth(time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)) || date.IsFirstOfMonth(a) {
		t.Fatal("is first of month")
	}
	if !date.IsLastOfMonth(sameMonth) || date.IsLastOfMonth(a) {
		t.Fatal("is last of month")
	}
	startDay := date.StartOfDay(a)
	endDay := date.EndOfDay(a)
	if !date.IsStartOfDay(startDay) || date.IsStartOfDay(a) {
		t.Fatal("is start of day")
	}
	if !date.IsEndOfDay(endDay) || date.IsEndOfDay(a) {
		t.Fatal("is end of day")
	}
	firstNoon := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	if !date.IsStartOfMonth(date.StartOfMonth(a)) || date.IsStartOfMonth(firstNoon) {
		t.Fatal("IsStartOfMonth")
	}
	if !date.IsEndOfMonth(date.EndOfMonth(a)) || date.IsEndOfMonth(sameMonth) {
		t.Fatal("IsEndOfMonth")
	}
	if !date.IsStartOfYear(date.StartOfYear(a)) || date.IsStartOfYear(a) {
		t.Fatal("IsStartOfYear")
	}
	if !date.IsEndOfYear(date.EndOfYear(a)) || date.IsEndOfYear(a) {
		t.Fatal("IsEndOfYear")
	}
	if !date.IsNoon(date.Noon(a)) || date.IsNoon(a.Add(time.Minute)) || !date.IsNoon(firstNoon) {
		t.Fatal("IsNoon")
	}
	if date.Quarter(a) != 3 || date.Quarter(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)) != 1 {
		t.Fatal("quarter")
	}
}

func TestQuarterBoundsAndWeeks(t *testing.T) {
	a := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC) // Q3
	start := date.StartOfQuarter(a)
	end := date.EndOfQuarter(a)
	if start.Month() != time.July || start.Day() != 1 {
		t.Fatalf("start quarter=%v", start)
	}
	if end.Month() != time.September || end.Day() != 30 {
		t.Fatalf("end quarter=%v", end)
	}
	sameQ := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	otherQ := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	if !date.IsSameQuarter(a, sameQ) || date.IsSameQuarter(a, otherQ) {
		t.Fatal("same quarter")
	}
	if date.DiffInWeeks(a, a.AddDate(0, 0, 14)) != 2 {
		t.Fatalf("diff weeks=%d", date.DiffInWeeks(a, a.AddDate(0, 0, 14)))
	}
}

func TestDiffMonthsYearsAgeBirthday(t *testing.T) {
	defer date.Restore()
	fixed := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	date.Pretend(fixed)

	from := time.Date(2024, 5, 10, 0, 0, 0, 0, time.UTC)
	if date.DiffInMonths(from, fixed) != 26 { // May 10 2024 -> Aug 3 2026 = 26 months?
		// May->Aug is 3 months in same year span... (2026-2024)*12 + (8-5) = 24+3 = 27, day 3 < 10 so 26
		t.Fatalf("diff months=%d", date.DiffInMonths(from, fixed))
	}
	if date.DiffInYears(from, fixed) != 2 {
		t.Fatalf("diff years=%d", date.DiffInYears(from, fixed))
	}
	// Aug 2026 Q3 -> Feb 2027 Q1 = 2 quarters
	if date.DiffInQuarters(fixed, time.Date(2027, 2, 1, 0, 0, 0, 0, time.UTC)) != 2 {
		t.Fatalf("diff quarters=%d", date.DiffInQuarters(fixed, time.Date(2027, 2, 1, 0, 0, 0, 0, time.UTC)))
	}
	birth := time.Date(2000, 8, 3, 0, 0, 0, 0, time.UTC)
	if date.Age(birth) != 26 {
		t.Fatalf("age=%d", date.Age(birth))
	}
	if !date.IsBirthday(birth) || date.IsBirthday(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("birthday")
	}
}

func TestDateShiftHelpers(t *testing.T) {
	fixed := time.Date(2026, 8, 3, 12, 30, 45, 0, time.UTC)

	if date.AddHours(fixed, 3).Hour() != 15 || date.SubHours(fixed, 2).Hour() != 10 {
		t.Fatal("hours")
	}
	if date.AddMinutes(fixed, 40).Minute() != 10 || date.SubMinutes(fixed, 15).Minute() != 15 {
		t.Fatal("minutes")
	}
	if date.AddSeconds(fixed, 20).Second() != 5 || date.SubSeconds(fixed, 5).Second() != 40 {
		t.Fatal("seconds")
	}
	if date.SubDays(fixed, 2).Day() != 1 {
		t.Fatalf("sub days=%v", date.SubDays(fixed, 2))
	}
	if date.SubMonths(fixed, 1).Month() != time.July {
		t.Fatal("sub months")
	}
	if date.AddQuarters(fixed, 1).Month() != time.November || date.SubQuarters(fixed, 1).Month() != time.May {
		t.Fatal("quarters")
	}
	if date.SubYears(fixed, 1).Year() != 2025 {
		t.Fatal("sub years")
	}
}

func TestDateCopySetTimeMaxMin(t *testing.T) {
	fixed := time.Date(2026, 8, 3, 12, 30, 45, 0, time.UTC)
	copied := date.Copy(fixed)
	if !copied.Equal(fixed) || copied.Location() != fixed.Location() {
		t.Fatalf("copy=%v", copied)
	}
	set := date.SetTime(fixed, 9, 15, 0)
	if set.Hour() != 9 || set.Minute() != 15 || set.Second() != 0 || set.Day() != 3 {
		t.Fatalf("set time=%v", set)
	}
	earlier := fixed.Add(-time.Hour)
	later := fixed.Add(time.Hour)
	if !date.Max(earlier, later, fixed).Equal(later) {
		t.Fatal("max")
	}
	if !date.Min(earlier, later, fixed).Equal(earlier) {
		t.Fatal("min")
	}
	avg := date.Average(earlier, later)
	if !avg.Equal(fixed) {
		t.Fatalf("average=%v want=%v", avg, fixed)
	}
	if !date.Max().IsZero() || !date.Min().IsZero() || !date.Average().IsZero() {
		t.Fatal("empty max/min/average")
	}
}

func TestDateCompareAndSetDate(t *testing.T) {
	a := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	b := time.Date(2026, 8, 3, 15, 0, 0, 0, time.UTC)
	if !date.Equal(a, a) || date.Equal(a, b) {
		t.Fatal("equal")
	}
	if !date.Gt(b, a) || date.Gt(a, b) || !date.Gte(a, a) || !date.Gte(b, a) {
		t.Fatal("gt/gte")
	}
	if !date.Lt(a, b) || date.Lt(b, a) || !date.Lte(b, b) || !date.Lte(a, b) {
		t.Fatal("lt/lte")
	}
	set := date.SetDate(a, 2027, time.January, 10)
	if set.Year() != 2027 || set.Month() != time.January || set.Day() != 10 || set.Hour() != 12 {
		t.Fatalf("set date=%v", set)
	}
}

func TestDateCompletionHelpers(t *testing.T) {
	defer date.Restore()
	fixed := time.Date(2026, 8, 5, 12, 30, 45, 123456789, time.UTC)
	date.Pretend(fixed)

	if !date.IsMidnight(date.StartOfDay(fixed)) || date.IsMidnight(fixed) {
		t.Fatal("IsMidnight")
	}
	if !date.IsStartOfMinute(date.StartOfMinute(fixed)) || date.IsStartOfMinute(fixed) {
		t.Fatal("IsStartOfMinute")
	}
	if !date.IsEndOfMinute(date.EndOfMinute(fixed)) || date.IsEndOfMinute(fixed) {
		t.Fatal("IsEndOfMinute")
	}
	if !date.IsStartOfSecond(date.StartOfSecond(fixed)) || date.IsStartOfSecond(fixed) {
		t.Fatal("IsStartOfSecond")
	}
	if !date.IsEndOfSecond(date.EndOfSecond(fixed)) || date.IsEndOfSecond(fixed) {
		t.Fatal("IsEndOfSecond")
	}
	if !date.IsStartOfQuarter(date.StartOfQuarter(fixed)) || date.IsStartOfQuarter(fixed) {
		t.Fatal("IsStartOfQuarter")
	}
	if !date.IsEndOfQuarter(date.EndOfQuarter(fixed)) || date.IsEndOfQuarter(fixed) {
		t.Fatal("IsEndOfQuarter")
	}
	if !date.IsFirstOfYear(time.Date(2026, 1, 1, 15, 0, 0, 0, time.UTC)) || date.IsFirstOfYear(fixed) {
		t.Fatal("IsFirstOfYear")
	}
	if !date.IsLastOfYear(time.Date(2026, 12, 31, 1, 0, 0, 0, time.UTC)) || date.IsLastOfYear(fixed) {
		t.Fatal("IsLastOfYear")
	}
	if !date.IsFirstOfQuarter(time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)) || date.IsFirstOfQuarter(fixed) {
		t.Fatal("IsFirstOfQuarter")
	}
	if !date.IsLastOfQuarter(time.Date(2026, 9, 30, 9, 0, 0, 0, time.UTC)) || date.IsLastOfQuarter(fixed) {
		t.Fatal("IsLastOfQuarter")
	}

	if date.ToDateString(date.Today()) != "2026-08-05" || date.ToDateString(date.Yesterday()) != "2026-08-04" || date.ToDateString(date.Tomorrow()) != "2026-08-06" {
		t.Fatal("Today/Yesterday/Tomorrow")
	}
	created := date.Create(2026, time.August, 1, 8, 9, 10, 0, time.UTC)
	if date.ToDateTimeString(created) != "2026-08-01 08:09:10" {
		t.Fatalf("Create=%s", date.ToDateTimeString(created))
	}
	if date.ToDateString(date.CreateFromDate(2026, time.March, 15, time.UTC)) != "2026-03-15" {
		t.Fatal("CreateFromDate")
	}
	fromTime := date.CreateFromTime(1, 2, 3, 0, time.UTC)
	if date.ToTimeString(fromTime) != "01:02:03" || date.ToDateString(fromTime) != "2026-08-05" {
		t.Fatalf("CreateFromTime=%v", fromTime)
	}
	if got := date.MustParse("2026-08-03"); date.ToDateString(got) != "2026-08-03" {
		t.Fatalf("MustParse=%v", got)
	}
	parsed, err := date.ParseFormat("02.01.2006", "03.08.2026")
	if err != nil || date.ToDateString(parsed) != "2026-08-03" {
		t.Fatalf("ParseFormat=%v err=%v", parsed, err)
	}

	if date.ToDateString(date.AddWeeks(fixed, 1)) != "2026-08-12" || date.ToDateString(date.SubWeeks(fixed, 1)) != "2026-07-29" {
		t.Fatal("Add/SubWeeks")
	}
	if date.DiffInMilliseconds(fixed, date.AddMilliseconds(fixed, 5)) != 5 {
		t.Fatal("AddMilliseconds")
	}
	if date.DiffInMicroseconds(fixed, date.AddMicroseconds(fixed, 7)) != 7 {
		t.Fatal("AddMicroseconds")
	}
	if date.DiffInNanoseconds(fixed, date.AddNanoseconds(fixed, 9)) != 9 {
		t.Fatal("AddNanoseconds")
	}
	if date.DiffInSeconds(fixed, date.AddDuration(fixed, 2*time.Second)) != 2 {
		t.Fatal("AddDuration")
	}
	if date.DiffInSeconds(date.SubDuration(fixed, 2*time.Second), fixed) != 2 {
		t.Fatal("SubDuration")
	}

	if date.Year(date.SetYear(fixed, 2030)) != 2030 {
		t.Fatal("SetYear")
	}
	if date.Month(date.SetMonth(fixed, time.February)) != time.February {
		t.Fatal("SetMonth")
	}
	if date.Day(date.SetDay(fixed, 20)) != 20 {
		t.Fatal("SetDay")
	}
	if date.Hour(date.SetHour(fixed, 7)) != 7 || date.Minute(date.SetMinute(fixed, 11)) != 11 || date.Second(date.SetSecond(fixed, 22)) != 22 {
		t.Fatal("SetHour/Minute/Second")
	}
	msSet := date.SetMillisecond(fixed, 250)
	if date.Millisecond(msSet) != 250 || date.Nanosecond(msSet) != 250*int(time.Millisecond) {
		t.Fatal("SetMillisecond")
	}
	usSet := date.SetMicrosecond(fixed, 123456)
	if date.Microsecond(usSet) != 123456 {
		t.Fatal("SetMicrosecond")
	}
	if date.DayOfYear(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)) != 1 {
		t.Fatal("DayOfYear")
	}
	if date.WeekOfYear(fixed) < 1 || date.IsoWeekYear(fixed) != 2026 {
		t.Fatal("WeekOfYear/IsoWeekYear")
	}
	if date.DaysInYear(fixed) != 365 || date.DaysInYear(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)) != 366 {
		t.Fatal("DaysInYear")
	}
	if date.IsoWeekday(time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)) != 1 || date.IsoWeekday(time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC)) != 7 {
		t.Fatal("IsoWeekday")
	}
	if date.Decade(fixed) != 2020 || date.Century(fixed) != 21 {
		t.Fatal("Decade/Century")
	}
	if !date.IsStartOfDecade(date.StartOfDecade(fixed)) || date.IsStartOfDecade(fixed) {
		t.Fatal("IsStartOfDecade")
	}
	if !date.IsEndOfDecade(date.EndOfDecade(fixed)) || date.IsEndOfDecade(fixed) {
		t.Fatal("IsEndOfDecade")
	}
	if date.ToIso8601String(time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)) != "2026-08-03T12:00:00Z" {
		t.Fatal("ToIso8601String")
	}

	if !date.IsLastWeek(time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC)) || date.IsLastWeek(fixed) {
		t.Fatal("IsLastWeek")
	}
	if !date.IsNextWeek(time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)) || date.IsNextWeek(fixed) {
		t.Fatal("IsNextWeek")
	}
	if !date.IsLastMonth(time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)) || date.IsLastMonth(fixed) {
		t.Fatal("IsLastMonth")
	}
	if !date.IsNextMonth(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)) || date.IsNextMonth(fixed) {
		t.Fatal("IsNextMonth")
	}
	if !date.IsLastYear(time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)) || date.IsLastYear(fixed) {
		t.Fatal("IsLastYear")
	}
	if !date.IsNextYear(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)) || date.IsNextYear(fixed) {
		t.Fatal("IsNextYear")
	}
	if !date.IsLastQuarter(time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)) || date.IsLastQuarter(fixed) {
		t.Fatal("IsLastQuarter")
	}
	if !date.IsNextQuarter(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)) || date.IsNextQuarter(fixed) {
		t.Fatal("IsNextQuarter")
	}

	a := time.Date(2026, 8, 3, 10, 0, 0, 0, time.UTC)
	b := time.Date(2026, 8, 3, 14, 0, 0, 0, time.UTC)
	mid := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	if !date.Between(mid, a, b) || date.BetweenExcluded(a, a, b) || !date.BetweenExcluded(mid, a, b) {
		t.Fatal("Between/BetweenExcluded")
	}
	if !date.NotEqual(a, b) || date.NotEqual(a, a) || !date.IsZero(time.Time{}) || date.IsZero(a) {
		t.Fatal("NotEqual/IsZero")
	}
	if date.FloatDiffInDays(a, b) != 4.0/24.0 || date.FloatDiffInHours(a, b) != 4 || date.FloatDiffInMinutes(a, b) != 240 || date.FloatDiffInSeconds(a, b) != 14400 {
		t.Fatal("FloatDiff*")
	}
	if date.AsUTC(fixed).Location() != time.UTC {
		t.Fatal("AsUTC")
	}
	if date.DaysUntil(time.Date(2026, 8, 7, 12, 30, 45, 123456789, time.UTC)) != 2 {
		t.Fatalf("DaysUntil=%d", date.DaysUntil(time.Date(2026, 8, 7, 12, 30, 45, 123456789, time.UTC)))
	}
	if date.DaysSince(time.Date(2026, 8, 3, 12, 30, 45, 123456789, time.UTC)) != 2 {
		t.Fatalf("DaysSince=%d", date.DaysSince(time.Date(2026, 8, 3, 12, 30, 45, 123456789, time.UTC)))
	}

	if !date.IsDayOfWeek(fixed, time.Wednesday) || date.IsDayOfWeek(fixed, time.Monday) {
		t.Fatal("IsDayOfWeek")
	}
	if date.Diff(a, b) != 4*time.Hour {
		t.Fatal("Diff")
	}
	if date.DiffInHoursSigned(b, a) != -4 || date.DiffInMinutesSigned(a, b) != 240 || date.DiffInSecondsSigned(a, b) != 14400 {
		t.Fatal("signed diffs")
	}
	if date.DiffInDaysSigned(a, a.AddDate(0, 0, 3)) != 3 || date.DiffInDaysSigned(a.AddDate(0, 0, 3), a) != -3 {
		t.Fatal("DiffInDaysSigned")
	}
	if !date.IsStartOfCentury(date.StartOfCentury(fixed)) || date.ToDateString(date.StartOfCentury(fixed)) != "2001-01-01" {
		t.Fatal("StartOfCentury")
	}
	if !date.IsEndOfCentury(date.EndOfCentury(fixed)) || date.ToDateString(date.EndOfCentury(fixed)) != "2100-12-31" {
		t.Fatal("EndOfCentury")
	}
	wall := date.SetTimezone(fixed, "UTC")
	if !date.IsUTC(wall) || date.Hour(wall) != 12 {
		t.Fatal("SetTimezone/IsUTC")
	}
	local := date.AsLocal(fixed)
	if !date.IsLocal(local) {
		t.Fatal("IsLocal")
	}
	if date.DiffInWeekendDays(time.Date(2026, 8, 7, 0, 0, 0, 0, time.UTC), time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)) != 2 {
		t.Fatalf("DiffInWeekendDays=%d", date.DiffInWeekendDays(time.Date(2026, 8, 7, 0, 0, 0, 0, time.UTC), time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)))
	}
	if date.DiffInMilliseconds(fixed, date.SubMilliseconds(date.AddMilliseconds(fixed, 5), 5)) != 0 {
		t.Fatal("SubMilliseconds")
	}
}

func TestWeekStartAndHumanLocale(t *testing.T) {
	defer date.SetWeekStartsAt(time.Monday)
	defer date.SetHumanLocale("en")

	wed := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	date.SetWeekStartsAt(time.Sunday)
	start := date.StartOfWeek(wed)
	if start.Weekday() != time.Sunday || date.ToDateString(start) != "2026-08-02" {
		t.Fatalf("sunday start=%v", start)
	}
	date.SetWeekStartsAt(time.Monday)
	start = date.StartOfWeek(wed)
	if start.Weekday() != time.Monday || date.ToDateString(start) != "2026-08-03" {
		t.Fatalf("monday start=%v", start)
	}

	date.Pretend(time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC))
	defer date.Restore()
	past := time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)
	date.SetHumanLocale("tr")
	if got := date.HumanDiff(past); got != "2 saat önce" {
		t.Fatalf("tr human=%q", got)
	}
	date.SetHumanLocale("en")
	if got := date.HumanDiff(past); got != "2 hours ago" {
		t.Fatalf("en human=%q", got)
	}
}
