package planfiles

import "math"

// orderStride is the spacing of freshly numbered `order` values.
const orderStride = 10

// orderItem is one plan task of a step in board sequence with the order key
// its file currently carries.
type orderItem struct {
	ID  string
	Key orderKey
}

// computeOrders returns the new `order` value of every task whose file must
// change so that sorting the tasks by their order keys reproduces seq, the
// board sequence of one step.
//
// The keys of the longest run of tasks that is already sorted stay as they
// are. Every other task takes a value between its kept neighbours, or the
// nearest kept neighbour plus or minus one at an edge. A task that must follow
// a kept task without `order` cannot be placed by value alone, because every
// task with an `order` sorts before every task without one; the tasks from the
// top of the step through that task are then numbered 10, 20, and so on.
// The result always reproduces seq: when a local assignment cannot, the whole
// step is renumbered.
func computeOrders(seq []orderItem) map[string]float64 {
	if len(seq) == 0 {
		return map[string]float64{}
	}
	values := assignOrders(seq, longestSortedRun(seq))
	if !reproducesSequence(seq, values) {
		values = renumber(seq)
	}
	return changedOnly(seq, values)
}

// longestSortedRun marks the members of a longest subsequence of seq that is
// strictly ascending by key. Among equally long runs it prefers later tasks.
func longestSortedRun(seq []orderItem) []bool {
	n := len(seq)
	length, prev := make([]int, n), make([]int, n)
	best := 0
	for i := range seq {
		length[i], prev[i] = 1, -1
		for j := 0; j < i; j++ {
			if seq[j].Key.less(seq[i].Key) && length[j]+1 >= length[i] {
				length[i], prev[i] = length[j]+1, j
			}
		}
		if length[i] >= length[best] {
			best = i
		}
	}
	kept := make([]bool, n)
	for i := best; i >= 0; i = prev[i] {
		kept[i] = true
	}
	return kept
}

func assignOrders(seq []orderItem, kept []bool) map[string]float64 {
	values := map[string]float64{}
	prefixEnd := -1
	for i := 0; i < len(seq); {
		if kept[i] {
			i++
			continue
		}
		j := i
		for j < len(seq) && !kept[j] {
			j++
		}
		if i > 0 && seq[i-1].Key.unordered {
			prefixEnd = j - 1
		} else {
			fillRun(values, seq, i, j)
		}
		i = j
	}
	for i := 0; i <= prefixEnd; i++ {
		values[seq[i].ID] = float64(orderStride * (i + 1))
	}
	return values
}

// fillRun gives the moved tasks seq[from:to] values between their kept
// neighbours seq[from-1] and seq[to]. The task above, when there is one, is
// known to carry an order.
func fillRun(values map[string]float64, seq []orderItem, from, to int) {
	count := to - from
	hasAbove := from > 0
	hasBelowOrder := to < len(seq) && !seq[to].Key.unordered
	for t := 0; t < count; t++ {
		var v float64
		switch {
		case hasAbove && hasBelowOrder:
			above, below := seq[from-1].Key.order, seq[to].Key.order
			v = above + (below-above)*float64(t+1)/float64(count+1)
		case hasAbove:
			v = seq[from-1].Key.order + float64(t+1)
		case hasBelowOrder:
			v = seq[to].Key.order - float64(count-t)
		default:
			v = float64(orderStride * (t + 1))
		}
		values[seq[from+t].ID] = v
	}
}

func renumber(seq []orderItem) map[string]float64 {
	values := make(map[string]float64, len(seq))
	for i, it := range seq {
		values[it.ID] = float64(orderStride * (i + 1))
	}
	return values
}

// reproducesSequence reports that applying values to seq and sorting by key
// yields seq again, with every value finite.
func reproducesSequence(seq []orderItem, values map[string]float64) bool {
	items := make([]orderItem, len(seq))
	for i, it := range seq {
		if v, ok := values[it.ID]; ok {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return false
			}
			it.Key = newOrderKey(&v, it.Key.relPath, it.Key.repoID)
		}
		items[i] = it
	}
	for i := 1; i < len(items); i++ {
		if !items[i-1].Key.less(items[i].Key) {
			return false
		}
	}
	return true
}

// changedOnly drops the values a file already carries.
func changedOnly(seq []orderItem, values map[string]float64) map[string]float64 {
	changed := make(map[string]float64, len(values))
	for _, it := range seq {
		v, ok := values[it.ID]
		if ok && (it.Key.unordered || it.Key.order != v) {
			changed[it.ID] = v
		}
	}
	return changed
}
