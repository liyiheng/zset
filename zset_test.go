package zset_test

import (
	"math/rand"
	"testing"
	"zset"
)

var s *zset.SortedSet

func init() {
	s = zset.New()
}
func TestNew(t *testing.T) {
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

func BenchmarkSortedSet_Add(b *testing.B) {
	b.StopTimer()
	// data initialization
	scores := make([]float64, b.N)
	IDs := make([]int64, b.N)
	for i := range IDs {
		scores[i] = rand.Float64() + float64(rand.Int31n(99))
		IDs[i] = int64(i) + 100000
	}
	// BCE
	_ = scores[:b.N]
	_ = IDs[:b.N]

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		s.Add(scores[i], IDs[i], nil)
	}
}

func BenchmarkSortedSet_GetRank(b *testing.B) {
	l := s.Length()
	for i := 0; i < b.N; i++ {
		s.GetRank(100000+int64(i)%l, true)
	}
}

func BenchmarkSortedSet_GetDataByRank(b *testing.B) {
	l := s.Length()
	for i := 0; i < b.N; i++ {
		s.GetDataByRank(int64(i)%l, true)
	}
}
