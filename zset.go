package zset

import (
	"math/rand"
)

/*================================== Redis skipList APIs =====================================
 *	zslCreate
 *	创建一个新的跳跃表。	O(1)
 *	zslFree
 *	释放给定跳跃表，以及表中包含的所有节点。	O(N) ， N 为跳跃表的长度。
 *	zslInsert
 *	将包含给定成员和分值的新节点添加到跳跃表中。	平均 O(\log N) ，最坏 O(N) ， N 为跳跃表长度。
 *	zslDelete
 *	删除跳跃表中包含给定成员和分值的节点。	平均 O(\log N) ，最坏 O(N) ， N 为跳跃表长度。
 *	zslGetRank
 *	返回包含给定成员和分值的节点在跳跃表中的排位。	平均 O(\log N) ，最坏 O(N) ， N 为跳跃表长度。
 *	zslGetElementByRank
 *	返回跳跃表在给定排位上的节点。	平均 O(\log N) ，最坏 O(N) ， N 为跳跃表长度。
 *	zslIsInRange
 *	给定一个分值范围（range），
 *	比如 0 到 15 ， 20 到 28 ，诸如此类，
 *	如果给定的分值范围包含在跳跃表的分值范围之内， 那么返回 1 ，否则返回 0 。
 *	通过跳跃表的表头节点和表尾节点， 这个检测可以用 O(1) 复杂度完成。
 *	zslFirstInRange
 *	给定一个分值范围， 返回跳跃表中第一个符合这个范围的节点。	平均 O(\log N) ，最坏 O(N) 。 N 为跳跃表长度。
 *	zslLastInRange
 *	给定一个分值范围， 返回跳跃表中最后一个符合这个范围的节点。	平均 O(\log N) ，最坏 O(N) 。 N 为跳跃表长度。
 *	zslDeleteRangeByScore
 *	给定一个分值范围， 删除跳跃表中所有在这个范围之内的节点。	O(N) ， N 为被删除节点数量。
 *	zslDeleteRangeByRank
 *	给定一个排位范围， 删除跳跃表中所有在这个范围之内的节点。	O(N) ， N 为被删除节点数量。
 */
const zSkiplistMaxlevel = 32

/*
 * 1. Port redis zset to golang
 * 2. Refactor
 */
type (
	skipListLevel struct {
		forward *skipListNode // 前进指针
		span    uint64        // 跨度
	}

	skipListNode struct {
		objID    int64            // 成员id
		score    float64          // 分值
		backward *skipListNode    // 后退指针
		level    []*skipListLevel // 层
	}
	obj struct {
		key        int64       // 成员id
		attachment interface{} // 自定义数据
		score      float64     // 分值
	}

	skipList struct {
		header *skipListNode // 头节点
		tail   *skipListNode // 尾节点
		length int64         // 节点个数
		level  int16         // 层数最多的节点的层数
	}
	// SortedSet is the final exported sorted set we can use
	SortedSet struct {
		dict map[int64]*obj
		zsl  *skipList
	}
	zrangespec struct {
		min   float64
		max   float64
		minex int32
		maxex int32
	}
	/* Struct to hold an inclusive/exclusive range spec by lexicographic comparison. */
	zlexrangespec struct {
		minKey int64
		maxKey int64
		minex  int /* are min or max exclusive? */
		maxex  int /* are min or max exclusive? */
	}
)

/* Create a skiplist node with the specified number of levels.
 * The SDS string 'obj' is referenced by the node after the call. */
func zslCreateNode(level int16, score float64, id int64) *skipListNode {
	n := &skipListNode{
		score: score,
		objID: id,
		level: make([]*skipListLevel, level),
	}
	for i := range n.level {
		n.level[i] = new(skipListLevel)
	}
	return n
}

/* Create a new skiplist. */
func zslCreate() *skipList {
	return &skipList{
		level:  1,
		header: zslCreateNode(zSkiplistMaxlevel, 0, 0),
	}
}

const zSkiplistP = 0.25 /* Skiplist P = 1/4 */

/* Returns a random level for the new skiplist node we are going to create.
 * The return value of this function is between 1 and _ZSKIPLIST_MAXLEVEL
 * (both inclusive), with a powerlaw-alike distribution where higher
 * levels are less likely to be returned. */
func randomLevel() int16 {
	level := int16(1)
	for float32(rand.Int31()&0xFFFF) < (zSkiplistP * 0xFFFF) {
		level++
	}
	if level < zSkiplistMaxlevel {
		return level
	}
	return zSkiplistMaxlevel
}

/* zslInsert a new node in the skiplist. Assumes the element does not already
 * exist (up to the caller to enforce that). The skiplist takes ownership
 * of the passed SDS string 'obj'. */
func (zsl *skipList) zslInsert(score float64, id int64) *skipListNode {
	update := make([]*skipListNode, zSkiplistMaxlevel)
	rank := make([]uint64, zSkiplistMaxlevel)
	x := zsl.header
	for i := zsl.level - 1; i >= 0; i-- {
		/* store rank that is crossed to reach the insert position */
		if i == zsl.level-1 {
			rank[i] = 0
		} else {
			rank[i] = rank[i+1]
		}
		if x.level[i] != nil {
			for x.level[i].forward != nil &&
				(x.level[i].forward.score < score ||
					(x.level[i].forward.score == score && x.level[i].forward.objID < id)) {
				rank[i] += x.level[i].span
				x = x.level[i].forward
			}
		}
		update[i] = x
	}
	/* we assume the element is not already inside, since we allow duplicated
	 * scores, reinserting the same element should never happen since the
	 * caller of zslInsert() should test in the hash table if the element is
	 * already inside or not. */
	level := randomLevel()
	if level > zsl.level {
		for i := zsl.level; i < level; i++ {
			rank[i] = 0
			update[i] = zsl.header
			update[i].level[i].span = uint64(zsl.length)
		}
		zsl.level = level
	}
	x = zslCreateNode(level, score, id)
	for i := int16(0); i < level; i++ {
		x.level[i].forward = update[i].level[i].forward
		update[i].level[i].forward = x

		/* update span covered by update[i] as x is inserted here */
		x.level[i].span = update[i].level[i].span - (rank[0] - rank[i])
		update[i].level[i].span = (rank[0] - rank[i]) + 1
	}

	/* increment span for untouched levels */
	for i := level; i < zsl.level; i++ {
		update[i].level[i].span++
	}

	if update[0] == zsl.header {
		x.backward = nil
	} else {
		x.backward = update[0]

	}
	if x.level[0].forward != nil {
		x.level[0].forward.backward = x
	} else {
		zsl.tail = x
	}
	zsl.length++
	return x
}

/* Internal function used by zslDelete, zslDeleteByScore and zslDeleteByRank */
func (zsl *skipList) zslDeleteNode(x *skipListNode, update []*skipListNode) {
	for i := int16(0); i < zsl.level; i++ {
		if update[i].level[i].forward == x {
			update[i].level[i].span += x.level[i].span - 1
			update[i].level[i].forward = x.level[i].forward
		} else {
			update[i].level[i].span--
		}
	}
	if x.level[0].forward != nil {
		x.level[0].forward.backward = x.backward
	} else {
		zsl.tail = x.backward
	}
	for zsl.level > 1 && zsl.header.level[zsl.level-1].forward == nil {
		zsl.level--
	}
	zsl.length--
}

/* Delete an element with matching score/element from the skiplist.
 * The function returns 1 if the node was found and deleted, otherwise
 * 0 is returned.
 *
 * If 'node' is NULL the deleted node is freed by zslFreeNode(), otherwise
 * it is not freed (but just unlinked) and *node is set to the node pointer,
 * so that it is possible for the caller to reuse the node (including the
 * referenced SDS string at node->obj). */
func (zsl *skipList) zslDelete(score float64, id int64) int {
	update := make([]*skipListNode, zSkiplistMaxlevel)
	x := zsl.header
	for i := zsl.level - 1; i >= 0; i-- {
		for x.level[i].forward != nil &&
			(x.level[i].forward.score < score ||
				(x.level[i].forward.score == score &&
					x.level[i].forward.objID < id)) {
			x = x.level[i].forward
		}
		update[i] = x
	}
	/* We may have multiple elements with the same score, what we need
	 * is to find the element with both the right score and object. */
	x = x.level[0].forward
	if x != nil && score == x.score && x.objID == id {
		zsl.zslDeleteNode(x, update)
		return 1
	}
	return 0 /* not found */
}

func zslValueGteMin(value float64, spec *zrangespec) bool {
	if spec.minex != 0 {
		return value > spec.min
	}
	return value >= spec.min
}

func zslValueLteMax(value float64, spec *zrangespec) bool {
	if spec.maxex != 0 {
		return value < spec.max
	}
	return value <= spec.max
}

/* Returns if there is a part of the zset is in range. */
func (zsl *skipList) zslIsInRange(ran *zrangespec) bool {
	/* Test for ranges that will always be empty. */
	if ran.min > ran.max ||
		(ran.min == ran.max && (ran.minex != 0 || ran.maxex != 0)) {
		return false
	}
	x := zsl.tail
	if x == nil || !zslValueGteMin(x.score, ran) {
		return false
	}
	x = zsl.header.level[0].forward
	if x == nil || !zslValueLteMax(x.score, ran) {
		return false
	}
	return true
}

/* Find the first node that is contained in the specified range.
 * Returns NULL when no element is contained in the range. */
func (zsl *skipList) zslFirstInRange(ran *zrangespec) *skipListNode {
	/* If everything is out of range, return early. */
	if !zsl.zslIsInRange(ran) {
		return nil
	}

	x := zsl.header
	for i := zsl.level - 1; i >= 0; i-- {
		/* Go forward while *OUT* of range. */
		for x.level[i].forward != nil &&
			!zslValueGteMin(x.level[i].forward.score, ran) {
			x = x.level[i].forward
		}
	}
	/* This is an inner range, so the next node cannot be NULL. */
	x = x.level[0].forward
	//serverAssert(x != NULL);

	/* Check if score <= max. */
	if !zslValueLteMax(x.score, ran) {
		return nil
	}
	return x
}

/* Find the last node that is contained in the specified range.
 * Returns NULL when no element is contained in the range. */
func (zsl *skipList) zslLastInRange(ran *zrangespec) *skipListNode {

	/* If everything is out of range, return early. */
	if !zsl.zslIsInRange(ran) {
		return nil
	}
	x := zsl.header
	for i := zsl.level - 1; i >= 0; i-- {
		/* Go forward while *IN* range. */
		for x.level[i].forward != nil &&
			zslValueLteMax(x.level[i].forward.score, ran) {
			x = x.level[i].forward
		}
	}
	/* This is an inner range, so this node cannot be NULL. */
	//serverAssert(x != NULL);

	/* Check if score >= min. */
	if !zslValueGteMin(x.score, ran) {
		return nil
	}
	return x
}

/* Delete all the elements with score between min and max from the skiplist.
 * Min and max are inclusive, so a score >= min || score <= max is deleted.
 * Note that this function takes the reference to the hash table view of the
 * sorted set, in order to remove the elements from the hash table too. */
func (zsl *skipList) zslDeleteRangeByScore(ran *zrangespec, dict map[int64]*obj) uint64 {
	removed := uint64(0)
	update := make([]*skipListNode, zSkiplistMaxlevel)
	x := zsl.header
	for i := zsl.level - 1; i >= 0; i-- {
		for x.level[i].forward != nil {
			var condition bool
			if ran.minex != 0 {
				condition = x.level[i].forward.score <= ran.min
			} else {
				condition = x.level[i].forward.score < ran.min
			}
			if !condition {
				break
			}
			x = x.level[i].forward
		}
		update[i] = x
	}

	/* Current node is the last with score < or <= min. */
	x = x.level[0].forward

	/* Delete nodes while in range. */
	for x != nil {
		var condition bool
		if ran.maxex != 0 {
			condition = x.score < ran.max
		} else {
			condition = x.score <= ran.max
		}
		if !condition {
			break
		}
		next := x.level[0].forward
		zsl.zslDeleteNode(x, update)
		delete(dict, x.objID)
		// Here is where x->obj is actually released.
		// And golang has GC, don't need to free manually anymore
		//zslFreeNode(x)
		removed++
		x = next
	}
	return removed
}

func (zsl *skipList) zslDeleteRangeByLex(ran *zlexrangespec, dict map[int64]*obj) uint64 {
	removed := uint64(0)

	update := make([]*skipListNode, zSkiplistMaxlevel)
	x := zsl.header
	for i := zsl.level - 1; i >= 0; i-- {
		for x.level[i].forward != nil && !zslLexValueGteMin(x.level[i].forward.objID, ran) {
			x = x.level[i].forward
		}
		update[i] = x
	}

	/* Current node is the last with score < or <= min. */
	x = x.level[0].forward

	/* Delete nodes while in range. */
	for x != nil && zslLexValueLteMax(x.objID, ran) {
		next := x.level[0].forward
		zsl.zslDeleteNode(x, update)
		delete(dict, x.objID)
		removed++
		x = next
	}
	return removed
}

func zslLexValueGteMin(id int64, spec *zlexrangespec) bool {
	if spec.minex != 0 {
		return compareKey(id, spec.minKey) > 0
	}
	return compareKey(id, spec.minKey) >= 0
}

func compareKey(a, b int64) int8 {
	if a == b {
		return 0
	} else if a > b {
		return 1
	}
	return -1
}

func zslLexValueLteMax(id int64, spec *zlexrangespec) bool {
	if spec.maxex != 0 {
		return compareKey(id, spec.maxKey) < 0
	}
	return compareKey(id, spec.maxKey) <= 0
}

/* Delete all the elements with rank between start and end from the skiplist.
 * Start and end are inclusive. Note that start and end need to be 1-based */
func (zsl *skipList) zslDeleteRangeByRank(start, end uint64, dict map[int64]*obj) uint64 {
	update := make([]*skipListNode, zSkiplistMaxlevel)
	var traversed, removed uint64

	x := zsl.header
	for i := zsl.level - 1; i >= 0; i-- {
		for x.level[i].forward != nil && (traversed+x.level[i].span) < start {
			traversed += x.level[i].span
			x = x.level[i].forward
		}
		update[i] = x
	}

	traversed++
	x = x.level[0].forward
	for x != nil && traversed <= end {
		next := x.level[0].forward
		zsl.zslDeleteNode(x, update)
		delete(dict, x.objID)
		removed++
		traversed++
		x = next
	}
	return removed
}

/* Find the rank for an element by both score and obj.
 * Returns 0 when the element cannot be found, rank otherwise.
 * Note that the rank is 1-based due to the span of zsl->header to the
 * first element. */
func (zsl *skipList) zslGetRank(score float64, key int64) int64 {
	rank := uint64(0)
	x := zsl.header
	for i := zsl.level - 1; i >= 0; i-- {
		for x.level[i].forward != nil &&
			(x.level[i].forward.score < score ||
				(x.level[i].forward.score == score &&
					x.level[i].forward.objID <= key)) {
			rank += x.level[i].span
			x = x.level[i].forward
		}

		/* x might be equal to zsl->header, so test if obj is non-NULL */
		if x.objID == key {
			return int64(rank)
		}
	}
	return 0
}

/* Finds an element by its rank. The rank argument needs to be 1-based. */
func (zsl *skipList) zslGetElementByRank(rank uint64) *skipListNode {
	traversed := uint64(0)
	x := zsl.header
	for i := zsl.level - 1; i >= 0; i-- {
		for x.level[i].forward != nil && (traversed+x.level[i].span) <= rank {
			traversed += x.level[i].span
			x = x.level[i].forward
		}
		if traversed == rank {
			return x
		}
	}
	return nil
}

/*-----------------------------------------------------------------------------
 * Common sorted set API
 *----------------------------------------------------------------------------*/

// New creates a new SortedSet and return its pointer
func New() *SortedSet {
	s := &SortedSet{
		dict: make(map[int64]*obj),
		zsl:  zslCreate(),
	}
	return s
}

// Length returns counts of elements
func (z *SortedSet) Length() int64 {
	return z.zsl.length
}

// Set is used to add or update an element
func (z *SortedSet) Set(score float64, key int64, dat interface{}) {
	v, ok := z.dict[key]
	z.dict[key] = &obj{attachment: dat, key: key, score: score}
	if ok {
		/* Remove and re-insert when score changes. */
		if score != v.score {
			z.zsl.zslDelete(v.score, key)
			z.zsl.zslInsert(score, key)
		}
	} else {
		z.zsl.zslInsert(score, key)
	}
}

// IncrBy ..
func (z *SortedSet) IncrBy(score float64, key int64) (float64, interface{}) {
	v, ok := z.dict[key]
	if !ok {
		// use negative infinity ?
		return 0, nil
	}
	if score != 0 {
		z.zsl.zslDelete(v.score, key)
		v.score += score
		z.zsl.zslInsert(v.score, key)
	}
	return v.score, v.attachment
}

// Delete removes an element from the SortedSet
// by its key.
func (z *SortedSet) Delete(key int64) (ok bool) {
	v, ok := z.dict[key]
	if ok {
		z.zsl.zslDelete(v.score, key)
		delete(z.dict, key)
		return true
	}
	return false
}

// GetRank returns position,score and extra data of an element which
// found by the parameter key.
// The parameter reverse determines the rank is descent or ascend，
// true means descend and false means ascend.
func (z *SortedSet) GetRank(key int64, reverse bool) (rank int64, score float64, data interface{}) {
	v, ok := z.dict[key]
	if !ok {
		return -1, 0, nil
	}
	r := z.zsl.zslGetRank(v.score, key)
	if reverse {
		r = z.zsl.length - r
	} else {
		r--
	}
	return int64(r), v.score, v.attachment

}

// GetData returns data stored in the map by its key
func (z *SortedSet) GetData(key int64) (data interface{}, ok bool) {
	o, ok := z.dict[key]
	if !ok {
		return nil, false
	}
	return o.attachment, true
}

// GetDataByRank returns the id,score and extra data of an element which
// found by position in the rank.
// The parameter rank is the position, reverse says if in the descend rank.
func (z *SortedSet) GetDataByRank(rank int64, reverse bool) (key int64, score float64, data interface{}) {
	if rank < 0 || rank > z.zsl.length {
		return 0, 0, nil
	}
	if reverse {
		rank = z.zsl.length - rank
	} else {
		rank++
	}
	n := z.zsl.zslGetElementByRank(uint64(rank))
	if n == nil {
		return 0, 0, nil
	}
	dat, _ := z.dict[n.objID]
	if dat == nil {
		return 0, 0, nil
	}
	return dat.key, dat.score, dat.attachment
}
