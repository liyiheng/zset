# zset
Implementing sorted set in Redis with golang.

## Usage

```go
s := zset.New()
// add data
s.Add(66, 1001, "test1")
s.Add(77, 1002, "test2")
s.Add(88, 1003, "test3")
s.Add(100, 1004, "liyiheng")
s.Add(99, 1005, "test4")
s.Add(44, 1006, "test5")
s.Add(44, 1001, "test1")

// get rank by id
rank, score, extra := s.GetRank(1004, false)
// get data by rank
id, score, extra := s.GetDataByRank(0, true)

// delete data by id
s.Delete(1001)
```
