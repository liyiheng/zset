[![Go Report Card](https://goreportcard.com/badge/github.com/XanthusL/zset)](https://goreportcard.com/report/github.com/XanthusL/zset)
# zset
Implementing sorted set in Redis with golang.

## TODO
Key type int64 to string, or just waiting for generics.

## Installation
```bash
go get -u github.com/XanthusL/zset
```

## Usage
Removed RWLock in the SortedSet. 
Just implement it yourself if needed.
```go
s := zset.New()
// add data
s.Set(66, 1001, "test1")
s.Set(77, 1002, "test2")
s.Set(88, 1003, "test3")
s.Set(100, 1004, "liyiheng")
s.Set(99, 1005, "test4")
s.Set(44, 1006, "test5")
// update data
s.Set(44, 1001, "test1")

// get rank by id
rank, score, extra := s.GetRank(1004, false)
// get data by rank
id, score, extra := s.GetDataByRank(0, true)
// get data by id
dat, ok := s.GetData(1001)

// delete data by id
s.Delete(1001)
```

## Benchmark

```bash
go test -test.bench=".*"
BenchmarkSortedSet_Add-4                 1000000              4121 ns/op
BenchmarkSortedSet_GetRank-4              500000              3592 ns/op
BenchmarkSortedSet_GetDataByRank-4       2000000               667 ns/op
PASS
ok      zset    11.365s
```
