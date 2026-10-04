package planfiles

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ordered(id string, order float64) orderItem {
	return orderItem{ID: id, Key: newOrderKey(&order, id+".md", "r")}
}

func unordered(id string) orderItem {
	return orderItem{ID: id, Key: newOrderKey(nil, id+".md", "r")}
}

// sortedAfter applies orders to seq and returns the IDs in key order.
func sortedAfter(seq []orderItem, orders map[string]float64) []string {
	items := append([]orderItem(nil), seq...)
	for i := range items {
		if v, ok := orders[items[i].ID]; ok {
			value := v
			items[i].Key = newOrderKey(&value, items[i].Key.relPath, items[i].Key.repoID)
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].Key.less(items[j].Key) })
	ids := make([]string, len(items))
	for i, it := range items {
		ids[i] = it.ID
	}
	return ids
}

func idsOf(seq []orderItem) []string {
	ids := make([]string, len(seq))
	for i, it := range seq {
		ids[i] = it.ID
	}
	return ids
}

// @covers AC-TASKS-PLAN-FILES-003.2
func TestComputeOrders_SortedSequenceChangesNothing(t *testing.T) {
	seq := []orderItem{ordered("a", 10), ordered("b", 20), unordered("c"), unordered("d")}

	assert.Empty(t, computeOrders(seq))
}

// @covers AC-TASKS-PLAN-FILES-003.2
func TestComputeOrders_MovedTaskBetweenOrderedNeighboursGetsMidpoint(t *testing.T) {
	// c was dragged above b: only c changes, between a and b.
	seq := []orderItem{ordered("a", 10), ordered("c", 30), ordered("b", 20), ordered("d", 40)}

	got := computeOrders(seq)

	require.Len(t, got, 1)
	assert.Equal(t, sortedAfter(seq, got), idsOf(seq))
}

// @covers AC-TASKS-PLAN-FILES-003.2
func TestComputeOrders_MoveToTopUsesNeighbourMinusOne(t *testing.T) {
	seq := []orderItem{ordered("d", 40), ordered("a", 10), ordered("b", 20), ordered("c", 30)}

	got := computeOrders(seq)

	assert.Equal(t, map[string]float64{"d": 9}, got)
}

// @covers AC-TASKS-PLAN-FILES-003.2
func TestComputeOrders_MoveToBottomUsesNeighbourPlusOne(t *testing.T) {
	seq := []orderItem{ordered("b", 20), ordered("c", 30), ordered("d", 40), ordered("a", 10)}

	got := computeOrders(seq)

	assert.Equal(t, map[string]float64{"a": 41}, got)
}

// @covers AC-TASKS-PLAN-FILES-003.2
func TestComputeOrders_OrderedTaskBelowUnorderedNeighboursNumbersFromTop(t *testing.T) {
	// m has an order, so it sorts before every unordered task; to put it
	// below u1 and u2 the tasks from the top through m must all carry orders.
	seq := []orderItem{unordered("u1"), unordered("u2"), ordered("m", 5), unordered("u3")}

	got := computeOrders(seq)

	assert.Equal(t, map[string]float64{"u1": 10, "u2": 20, "m": 30}, got)
	assert.Equal(t, idsOf(seq), sortedAfter(seq, got))
}

// @covers AC-TASKS-PLAN-FILES-003.2
func TestComputeOrders_UnorderedMovedTaskNextToOrderedOne(t *testing.T) {
	seq := []orderItem{ordered("a", 10), unordered("z"), unordered("b")}

	got := computeOrders(seq)

	require.Len(t, got, 1)
	assert.Equal(t, idsOf(seq), sortedAfter(seq, got))
}

// @covers AC-TASKS-PLAN-FILES-003.2
func TestComputeOrders_EqualOrdersFallBackToFullRenumber(t *testing.T) {
	// a and b tie on order and path order decides; a task dropped between
	// them cannot take a value strictly between.
	a, b := ordered("a", 10), ordered("b", 10)
	x := orderItem{ID: "x", Key: newOrderKey(floatPtr(10), "zzz.md", "r")}
	seq := []orderItem{a, x, b}

	got := computeOrders(seq)

	assert.Equal(t, idsOf(seq), sortedAfter(seq, got))
}

func floatPtr(v float64) *float64 { return &v }

// @covers AC-TASKS-PLAN-FILES-003.2
func TestComputeOrders_PropertyApplyingOutputReproducesSequence(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	for trial := 0; trial < 2000; trial++ {
		n := 1 + rng.Intn(12)
		seq := make([]orderItem, n)
		for i := range seq {
			id := fmt.Sprintf("t%02d", i)
			switch rng.Intn(3) {
			case 0:
				seq[i] = unordered(id)
			default:
				seq[i] = ordered(id, float64(rng.Intn(6)*10))
			}
		}
		rng.Shuffle(n, func(i, j int) { seq[i], seq[j] = seq[j], seq[i] })

		got := computeOrders(seq)

		require.Equal(t, idsOf(seq), sortedAfter(seq, got), "trial %d seq=%v got=%v", trial, idsOf(seq), got)
	}
}

// @covers AC-TASKS-PLAN-FILES-003.2
func TestComputeOrders_PropertySingleMoveBetweenOrderedNeighboursChangesOneFile(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	for trial := 0; trial < 500; trial++ {
		n := 3 + rng.Intn(8)
		seq := make([]orderItem, n)
		for i := range seq {
			seq[i] = ordered(fmt.Sprintf("t%02d", i), float64((i+1)*10))
		}
		from := rng.Intn(n)
		to := rng.Intn(n)
		moved := seq[from]
		rest := append(append([]orderItem(nil), seq[:from]...), seq[from+1:]...)
		seq = append(append(append([]orderItem(nil), rest[:to]...), moved), rest[to:]...)

		got := computeOrders(seq)

		require.LessOrEqual(t, len(got), 1, "trial %d seq=%v got=%v", trial, idsOf(seq), got)
		require.Equal(t, idsOf(seq), sortedAfter(seq, got))
	}
}
