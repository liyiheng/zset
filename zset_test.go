package zset_test

import (
	"testing"
	"zset"
)

func TestNew(t *testing.T) {
	s := zset.New()
	if s == nil {
		t.Failed()
	}
	s.Add(66, 1001, "test1")
	s.Add(77, 1002, "test2")
	s.Add(88, 1003, "test3")
	s.Add(100, 1004, "liyiheng")
	s.Add(99, 1005, "test4")
	s.Add(44, 1006, "test5")
	s.Add(44, 1001, "test1")

	rank, score, extra := s.GetRank(1004, false)
	if rank == 5 {
		t.Log("Key:", 1004, "Rank:", rank, "Score:", score, "Extra:", extra)
	} else {
		t.Error("Key:", 1004, "Rank:", rank, "Score:", score, "Extra:", extra)
	}
	id, score, extra := s.GetDataByRank(0, true)
	t.Log("GetData[REVERSE] Rank:", 0, "ID:", id, "Score:", score, "Extra:", extra)
	id, score, extra = s.GetDataByRank(0, false)
	t.Log("GetData[UNREVERSE] Rank:", 0, "ID:", id, "Score:", score, "Extra:", extra)
	s.Delete(1001)

}
