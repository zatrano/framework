package arr_test

import (
	"fmt"
	"testing"

	"github.com/zatrano/framework/packages/support/arr"
)

func TestArrHelpers(t *testing.T) {
	if arr.First([]int{1, 2}) != 1 || arr.Last([]int{1, 2}) != 2 {
		t.Fatal("first/last")
	}
	if !arr.Contains([]string{"a", "b"}, "b") {
		t.Fatal("contains")
	}
	u := arr.Unique([]string{"a", "a", "b"})
	if len(u) != 2 {
		t.Fatalf("unique=%v", u)
	}
	only := arr.Only(map[string]any{"a": 1, "b": 2}, "a")
	if only["a"] != 1 || len(only) != 1 {
		t.Fatalf("only=%v", only)
	}
	if got := arr.Get(map[string]any{"user": map[string]any{"name": "ada"}}, "user.name"); got != "ada" {
		t.Fatalf("get=%v", got)
	}
	rev := arr.Reverse([]int{1, 2, 3})
	if len(rev) != 3 || rev[0] != 3 || rev[2] != 1 {
		t.Fatalf("reverse=%v", rev)
	}
	diff := arr.Diff([]string{"a", "b", "c"}, []string{"b"})
	if len(diff) != 2 || diff[0] != "a" || diff[1] != "c" {
		t.Fatalf("diff=%v", diff)
	}
	inter := arr.Intersect([]string{"a", "b", "c"}, []string{"b", "c", "d"})
	if len(inter) != 2 || inter[0] != "b" || inter[1] != "c" {
		t.Fatalf("intersect=%v", inter)
	}
	union := arr.Union([]string{"a", "b"}, []string{"b", "c"}, []string{"c", "d"})
	if len(union) != 4 || union[0] != "a" || union[3] != "d" {
		t.Fatalf("union=%v", union)
	}
	concat := arr.Concat([]int{1, 2}, []int{2, 3})
	if len(concat) != 4 || concat[0] != 1 || concat[2] != 2 || concat[3] != 3 {
		t.Fatalf("concat=%v", concat)
	}
	if !arr.Every([]int{2, 4, 6}, func(n int) bool { return n%2 == 0 }) {
		t.Fatal("every true")
	}
	if arr.Every([]int{2, 3, 4}, func(n int) bool { return n%2 == 0 }) {
		t.Fatal("every false")
	}
	if !arr.Every([]int{}, func(n int) bool { return false }) {
		t.Fatal("every empty")
	}
	if !arr.Some([]int{1, 2, 3}, func(n int) bool { return n == 2 }) || arr.Some([]int{1, 3}, func(n int) bool { return n == 2 }) {
		t.Fatal("some")
	}
	if !arr.None([]int{2, 4, 6}, func(n int) bool { return n%2 == 1 }) || arr.None([]int{2, 3}, func(n int) bool { return n%2 == 1 }) {
		t.Fatal("none")
	}
	odds := []int{1, 3, 5, 7}
	if got := arr.FirstWhere(odds, func(n int) bool { return n > 4 }); got != 5 {
		t.Fatalf("first where=%d", got)
	}
	if got := arr.FirstWhere(odds, func(n int) bool { return n > 10 }, -1); got != -1 {
		t.Fatalf("first where fallback=%d", got)
	}
	if got := arr.LastWhere(odds, func(n int) bool { return n < 6 }); got != 5 {
		t.Fatalf("last where=%d", got)
	}
	if got := arr.LastWhere(odds, func(n int) bool { return n < 0 }, -1); got != -1 {
		t.Fatalf("last where fallback=%d", got)
	}
	flat := arr.Flatten([][]int{{1, 2}, {3}})
	if len(flat) != 3 || flat[2] != 3 {
		t.Fatalf("flatten=%v", flat)
	}
	sum := arr.Reduce([]int{1, 2, 3, 4}, 0, func(acc, n int) int { return acc + n })
	if sum != 10 {
		t.Fatalf("reduce=%d", sum)
	}
	shuffled := arr.Shuffle([]string{"a", "b", "c", "d"})
	if len(shuffled) != 4 {
		t.Fatalf("shuffle=%v", shuffled)
	}
}

func TestHasWhereDotSetCollapse(t *testing.T) {
	nested := map[string]any{
		"user": map[string]any{
			"name": "ada",
			"meta": map[string]any{"role": "admin"},
		},
	}
	if !arr.Has(nested, "user.name") || arr.Has(nested, "user.email") {
		t.Fatal("has")
	}
	arr.Set(nested, "user.meta.city", "Istanbul")
	if got := arr.Get(nested, "user.meta.city"); got != "Istanbul" {
		t.Fatalf("set/get=%v", got)
	}
	dotted := arr.Dot(nested)
	if dotted["user.name"] != "ada" || dotted["user.meta.role"] != "admin" || dotted["user.meta.city"] != "Istanbul" {
		t.Fatalf("dot=%v", dotted)
	}
	undotted := arr.Undot(dotted)
	if arr.Get(undotted, "user.name") != "ada" || arr.Get(undotted, "user.meta.city") != "Istanbul" {
		t.Fatalf("undot=%v", undotted)
	}
	flipped := arr.Flip(map[string]string{"a": "1", "b": "2"})
	if flipped["1"] != "a" || flipped["2"] != "b" {
		t.Fatalf("flip=%v", flipped)
	}
	if _, ok := arr.Random([]int{}); ok {
		t.Fatal("random empty")
	}
	if v, ok := arr.Random([]int{7}); !ok || v != 7 {
		t.Fatal("random single")
	}
	taken := arr.TakeRandom([]int{1, 2, 3, 4, 5}, 3)
	if len(taken) != 3 {
		t.Fatalf("take random=%v", taken)
	}
	if got := arr.TakeRandom([]int{1, 2}, 5); len(got) != 2 {
		t.Fatalf("take random all=%v", got)
	}
	rows := []map[string]any{
		{"name": "a", "ok": true},
		{"name": "b", "ok": false},
		{"name": "c", "ok": true},
	}
	filtered := arr.Where(rows, "ok", true)
	if len(filtered) != 2 || filtered[0]["name"] != "a" {
		t.Fatalf("where=%v", filtered)
	}
	rejected := arr.WhereNot(rows, "ok", true)
	if len(rejected) != 1 || rejected[0]["name"] != "b" {
		t.Fatalf("where not=%v", rejected)
	}
	collapsed := arr.Collapse([][]int{{1, 2}, {3}})
	if len(collapsed) != 3 || collapsed[2] != 3 {
		t.Fatalf("collapse=%v", collapsed)
	}
}

func TestGroupByKeyBySortByPartition(t *testing.T) {
	rows := []map[string]any{
		{"name": "c", "role": "admin"},
		{"name": "a", "role": "user"},
		{"name": "b", "role": "admin"},
	}
	grouped := arr.GroupBy(rows, "role")
	if len(grouped["admin"]) != 2 || len(grouped["user"]) != 1 {
		t.Fatalf("groupby=%v", grouped)
	}
	keyed := arr.KeyBy(rows, "name")
	if keyed["a"]["role"] != "user" || keyed["b"]["role"] != "admin" {
		t.Fatalf("keyby=%v", keyed)
	}
	sorted := arr.SortBy(rows, func(a, b map[string]any) bool {
		return fmt.Sprint(a["name"]) < fmt.Sprint(b["name"])
	})
	if sorted[0]["name"] != "a" || sorted[2]["name"] != "c" {
		t.Fatalf("sortby=%v", sorted)
	}
	matched, rejected := arr.Partition(rows, func(row map[string]any) bool {
		return row["role"] == "admin"
	})
	if len(matched) != 2 || len(rejected) != 1 {
		t.Fatalf("partition matched=%d rejected=%d", len(matched), len(rejected))
	}
}

func TestTakeSkipNthSliding(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6}
	if got := arr.Take(items, 3); len(got) != 3 || got[2] != 3 {
		t.Fatalf("take=%v", got)
	}
	if got := arr.Skip(items, 4); len(got) != 2 || got[0] != 5 {
		t.Fatalf("skip=%v", got)
	}
	if got := arr.Nth(items, 2); len(got) != 3 || got[0] != 1 || got[2] != 5 {
		t.Fatalf("nth=%v", got)
	}
	if got := arr.Nth(items, 2, 1); len(got) != 3 || got[0] != 2 {
		t.Fatalf("nth offset=%v", got)
	}
	windows := arr.Sliding(items, 3)
	if len(windows) != 4 || windows[0][0] != 1 || windows[3][2] != 6 {
		t.Fatalf("sliding=%v", windows)
	}
	stepped := arr.Sliding(items, 2, 2)
	if len(stepped) != 3 || stepped[1][0] != 3 {
		t.Fatalf("sliding step=%v", stepped)
	}
	if got := arr.TakeWhile(items, func(n int) bool { return n < 4 }); len(got) != 3 || got[2] != 3 {
		t.Fatalf("takeWhile=%v", got)
	}
	if got := arr.SkipWhile(items, func(n int) bool { return n < 4 }); len(got) != 3 || got[0] != 4 {
		t.Fatalf("skipWhile=%v", got)
	}
	if got := arr.TakeUntil(items, func(n int) bool { return n == 4 }); len(got) != 3 || got[2] != 3 {
		t.Fatalf("takeUntil=%v", got)
	}
	if got := arr.SkipUntil(items, func(n int) bool { return n == 4 }); len(got) != 3 || got[0] != 4 {
		t.Fatalf("skipUntil=%v", got)
	}
}

func TestRejectPrependPushPopShiftForgetPull(t *testing.T) {
	items := []int{1, 2, 3, 4}
	rejected := arr.Reject(items, func(n int) bool { return n%2 == 0 })
	if len(rejected) != 2 || rejected[0] != 1 || rejected[1] != 3 {
		t.Fatalf("reject=%v", rejected)
	}
	if got := arr.Prepend(items, 0, -1); len(got) != 6 || got[0] != 0 || got[1] != -1 || got[2] != 1 {
		t.Fatalf("prepend=%v", got)
	}
	if got := arr.Push(items, 5, 6); len(got) != 6 || got[4] != 5 || got[5] != 6 {
		t.Fatalf("push=%v", got)
	}
	last, rest := arr.Pop(items)
	if last != 4 || len(rest) != 3 || rest[2] != 3 {
		t.Fatalf("pop last=%d rest=%v", last, rest)
	}
	first, rest := arr.Shift(items)
	if first != 1 || len(rest) != 3 || rest[0] != 2 {
		t.Fatalf("shift first=%d rest=%v", first, rest)
	}

	data := map[string]any{
		"name": "Ada",
		"meta": map[string]any{"city": "Ankara", "role": "admin"},
	}
	pulled := arr.Pull(data, "meta.city", "")
	if pulled != "Ankara" || arr.Has(data, "meta.city") || !arr.Has(data, "meta.role") {
		t.Fatalf("pull=%v data=%v", pulled, data)
	}
	arr.Forget(data, "name", "meta.role")
	if arr.Has(data, "name") || arr.Has(data, "meta.role") {
		t.Fatalf("forget failed: %v", data)
	}
}

func TestRangeTimesZipPad(t *testing.T) {
	if got := arr.Range(1, 5); len(got) != 5 || got[0] != 1 || got[4] != 5 {
		t.Fatalf("range=%v", got)
	}
	if got := arr.Range(5, 1); len(got) != 5 || got[0] != 5 || got[4] != 1 {
		t.Fatalf("range desc=%v", got)
	}
	if got := arr.Range(1, 10, 3); len(got) != 4 || got[3] != 10 {
		t.Fatalf("range step=%v", got)
	}
	times := arr.Times(3, func(i int) string { return string(rune('a' + i)) })
	if len(times) != 3 || times[0] != "a" || times[2] != "c" {
		t.Fatalf("times=%v", times)
	}
	zipped := arr.Zip([]string{"a", "b", "c"}, []int{1, 2})
	if len(zipped) != 2 || zipped[0].First != "a" || zipped[0].Second != 1 || zipped[1].First != "b" {
		t.Fatalf("zip=%v", zipped)
	}
	padded := arr.Pad([]int{1, 2}, 4, 0)
	if len(padded) != 4 || padded[2] != 0 || padded[3] != 0 {
		t.Fatalf("pad=%v", padded)
	}
	if got := arr.Pad([]int{1, 2, 3}, 2, 9); len(got) != 3 {
		t.Fatalf("pad no truncate=%v", got)
	}
}

func TestDoesntContainSoleDuplicatesCrossJoin(t *testing.T) {
	items := []int{1, 2, 2, 3, 1}
	if !arr.DoesntContain(items, 9) || arr.DoesntContain(items, 2) {
		t.Fatal("DoesntContain")
	}
	if v, ok := arr.Sole([]string{"only"}); !ok || v != "only" {
		t.Fatal("Sole ok")
	}
	if _, ok := arr.Sole([]int{1, 2}); ok {
		t.Fatal("Sole multi")
	}
	if _, ok := arr.Sole([]int{}); ok {
		t.Fatal("Sole empty")
	}
	dups := arr.Duplicates(items)
	if len(dups) != 2 || dups[0] != 1 || dups[1] != 2 {
		t.Fatalf("duplicates=%v", dups)
	}
	joined := arr.CrossJoin([]int{1, 2}, []int{10, 20})
	if len(joined) != 4 || joined[0][0] != 1 || joined[0][1] != 10 || joined[3][0] != 2 || joined[3][1] != 20 {
		t.Fatalf("crossJoin=%v", joined)
	}
	if got := arr.CrossJoin([]int{1}, []int{}); len(got) != 0 {
		t.Fatalf("crossJoin empty set=%v", got)
	}
}

func TestIndexOfFindIndexHasKey(t *testing.T) {
	items := []int{1, 2, 2, 3, 1}
	if arr.IndexOf(items, 2) != 1 || arr.IndexOf(items, 9) != -1 {
		t.Fatal("IndexOf")
	}
	if arr.LastIndexOf(items, 1) != 4 || arr.LastIndexOf(items, 9) != -1 {
		t.Fatal("LastIndexOf")
	}
	if arr.FindIndex(items, func(n int) bool { return n > 2 }) != 3 {
		t.Fatal("FindIndex")
	}
	if arr.FindIndex(items, func(n int) bool { return n > 9 }) != -1 {
		t.Fatal("FindIndex miss")
	}
	if arr.FindLastIndex(items, func(n int) bool { return n == 2 }) != 2 {
		t.Fatal("FindLastIndex")
	}
	if arr.FindLastIndex(items, func(n int) bool { return n > 9 }) != -1 {
		t.Fatal("FindLastIndex miss")
	}
	if !arr.ContainsAll(items, 1, 2, 3) || arr.ContainsAll(items, 1, 9) {
		t.Fatal("ContainsAll")
	}
	if !arr.ContainsAny(items, 9, 2) || arr.ContainsAny(items, 8, 9) {
		t.Fatal("ContainsAny")
	}
	if !arr.DoesntContainAny(items, 8, 9) || arr.DoesntContainAny(items, 8, 2) {
		t.Fatal("DoesntContainAny")
	}
	if !arr.DoesntContainAll(items, 1, 9) || arr.DoesntContainAll(items, 1, 2, 3) {
		t.Fatal("DoesntContainAll")
	}
	flat := arr.FlatMap([]int{1, 2}, func(n int) []int { return []int{n, n * 10} })
	if len(flat) != 4 || flat[0] != 1 || flat[1] != 10 || flat[2] != 2 || flat[3] != 20 {
		t.Fatalf("FlatMap=%v", flat)
	}
	uniqueBy := arr.UniqueBy([]string{"aa", "b", "cc", "d"}, func(s string) int { return len(s) })
	if len(uniqueBy) != 2 || uniqueBy[0] != "aa" || uniqueBy[1] != "b" {
		t.Fatalf("UniqueBy=%v", uniqueBy)
	}
	filtered := arr.FilterMap([]int{1, 2, 3, 4}, func(n int) (int, bool) {
		if n%2 == 0 {
			return n * 10, true
		}
		return 0, false
	})
	if len(filtered) != 2 || filtered[0] != 20 || filtered[1] != 40 {
		t.Fatalf("FilterMap=%v", filtered)
	}
	data := map[string]int{"a": 1, "b": 2}
	if !arr.HasKey(data, "a") || arr.HasKey(data, "c") {
		t.Fatal("HasKey")
	}
	if !arr.HasAnyKey(data, "c", "a") || arr.HasAnyKey(data, "c", "d") {
		t.Fatal("HasAnyKey")
	}
	if !arr.HasAllKeys(data, "a", "b") || arr.HasAllKeys(data, "a", "c") {
		t.Fatal("HasAllKeys")
	}
	if !arr.MissingKey(data, "c") || arr.MissingKey(data, "a") {
		t.Fatal("MissingKey")
	}
	if !arr.MissingAnyKey(data, "a", "c") || arr.MissingAnyKey(data, "a", "b") {
		t.Fatal("MissingAnyKey")
	}
	if !arr.MissingAllKeys(data, "c", "d") || arr.MissingAllKeys(data, "a", "c") {
		t.Fatal("MissingAllKeys")
	}
	if !arr.IsEmpty([]int{}) || arr.IsEmpty([]int{1}) || !arr.IsNotEmpty([]string{"a"}) || arr.IsNotEmpty([]string{}) {
		t.Fatal("IsEmpty/IsNotEmpty")
	}
	combined := arr.Combine([]string{"a", "b"}, []int{1, 2})
	if combined["a"] != 1 || combined["b"] != 2 || len(combined) != 2 {
		t.Fatalf("Combine=%v", combined)
	}
	if got := arr.Combine([]string{"a", "b", "c"}, []int{1}); len(got) != 1 || got["a"] != 1 {
		t.Fatalf("Combine short=%v", got)
	}
	if got := arr.Compact([]string{"a", "", "b", "", "c"}); len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("Compact strings=%v", got)
	}
	if got := arr.Compact([]int{0, 1, 0, 2, 0}); len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("Compact ints=%v", got)
	}
}

func TestMinMaxSumAvg(t *testing.T) {
	items := []int{3, 1, 4, 1, 5}
	min, ok := arr.Min(items)
	if !ok || min != 1 {
		t.Fatalf("min=%v ok=%v", min, ok)
	}
	max, ok := arr.Max(items)
	if !ok || max != 5 {
		t.Fatalf("max=%v ok=%v", max, ok)
	}
	if _, ok := arr.Min([]int{}); ok {
		t.Fatal("min empty")
	}
	if arr.Sum(items) != 14 {
		t.Fatalf("sum=%d", arr.Sum(items))
	}
	if got := arr.Avg([]float64{2, 4, 6}); got != 4 {
		t.Fatalf("avg=%v", got)
	}
	if arr.Avg([]int{}) != 0 {
		t.Fatal("avg empty")
	}
}

func TestMedianModeCountByProduct(t *testing.T) {
	items := []int{3, 1, 4, 1, 5}
	median, ok := arr.Median(items)
	if !ok || median != 3 {
		t.Fatalf("median=%v ok=%v", median, ok)
	}
	even, ok := arr.Median([]float64{1, 2, 3, 4})
	if !ok || even != 2.5 {
		t.Fatalf("even median=%v ok=%v", even, ok)
	}
	if _, ok := arr.Median([]int{}); ok {
		t.Fatal("median empty")
	}

	mode := arr.Mode(items)
	if len(mode) != 1 || mode[0] != 1 {
		t.Fatalf("mode=%v", mode)
	}
	tie := arr.Mode([]string{"a", "b", "a", "b", "c"})
	if len(tie) != 2 || tie[0] != "a" || tie[1] != "b" {
		t.Fatalf("mode tie=%v", tie)
	}
	if got := arr.Mode([]int{}); got != nil {
		t.Fatalf("mode empty=%v", got)
	}

	counts := arr.CountBy([]string{"a", "b", "a"})
	if counts["a"] != 2 || counts["b"] != 1 {
		t.Fatalf("countBy=%v", counts)
	}
	if arr.Product(items) != 60 {
		t.Fatalf("product=%d", arr.Product(items))
	}
	if arr.Product([]int{}) != 1 {
		t.Fatal("product empty")
	}
}
