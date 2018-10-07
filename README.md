lruish
==========

This provides the `lrurish` package which implements a fixed-size lru-flavoured cache.
It can be used either thread safe or thread-unsafe. Heavily inspired by Hashicorp [golang-lru](https://github.com/hashicorp/golang-lru).

This package is called `lru:ish`, because while it behaves somewhat like a least-recently-used cache,
it does not strictly conform to that. This is indended for usecases where a very fast fixed-size cache is needed,
and where the strict guarantees of LRU are less important.

How does it work
===============
It maintains one map of values, and another 'ring'. The ring models a circular buffer, and the cache
always consider it to be at capacity.
Appending new items simply moves the head counter-clockwise, inserting the new item at the 'top'.
When an existing item is accessed, it is move upwards, switching position with another item.

For example, if an item at position `100` is accessed, it switches position with item `50`. Next time it is
accessed, it would switch with item at position `25`, then `12`, then `6`, `3`,`1` and finally `0`.

Speeds
=============

`RandUnsynched` tests the speed and hit/miss ratio of an non threadsafe cache. Higher numbers for
ratio is better
```
BenchmarkLRU_RandUnsynched-4   	 5000000	       363 ns/op	      60 B/op	       3 allocs/op
--- BENCH: BenchmarkLRU_RandUnsynched-4
	lruish_test.go:33: hit: 0 miss: 1 ratio: 0.000000
	lruish_test.go:33: hit: 0 miss: 100 ratio: 0.000000
	lruish_test.go:33: hit: 1322 miss: 8678 ratio: 0.152339
	lruish_test.go:33: hit: 248092 miss: 751908 ratio: 0.329950
	lruish_test.go:33: hit: 1249566 miss: 3750434 ratio: 0.333179
```

`RandSynched` tests the speed and hit/miss ratio of the threadsave cache.

```
BenchmarkLRU_RandSynched-4     	 3000000	       505 ns/op	      60 B/op	       3 allocs/op
--- BENCH: BenchmarkLRU_RandSynched-4
	lruish_test.go:62: hit: 0 miss: 1 ratio: 0.000000
	lruish_test.go:62: hit: 0 miss: 100 ratio: 0.000000
	lruish_test.go:62: hit: 1362 miss: 8638 ratio: 0.157675
	lruish_test.go:62: hit: 248266 miss: 751734 ratio: 0.330258
	lruish_test.go:62: hit: 748345 miss: 2251655 ratio: 0.332353
```

Corresponding benchmark for `golang-lru` (only threadsafe version exposed)
```
BenchmarkLRU_Rand-4   	 3000000	       544 ns/op	      84 B/op	       4 allocs/op
--- BENCH: BenchmarkLRU_Rand-4
	lru_test.go:34: hit: 0 miss: 1 ratio: 0.000000
	lru_test.go:34: hit: 0 miss: 100 ratio: 0.000000
	lru_test.go:34: hit: 1374 miss: 8626 ratio: 0.159286
	lru_test.go:34: hit: 249047 miss: 750953 ratio: 0.331641
	lru_test.go:34: hit: 748720 miss: 2251280 ratio: 0.332575
```

Example
=======

Using the LRU is very simple:

```go
l, _ := NewUnsynced(128)
for i := 0; i < 256; i++ {
    l.Add(i, nil)
}
if l.Len() != 128 {
    panic(fmt.Sprintf("bad len: %v", l.Len()))
}
```
