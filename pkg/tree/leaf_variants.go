package tree

import (
	"iter"
	"math"

	"github.com/sdcio/data-server/pkg/utils"
)

type LeafVariants struct {
	les []*LeafEntry
	tc  *TreeContext
}

func newLeafVariants(tc *TreeContext) *LeafVariants {
	return &LeafVariants{
		les: make([]*LeafEntry, 0, 2),
		tc:  tc,
	}
}

func (lv *LeafVariants) Append(le *LeafEntry) {
	lv.les = append(lv.les, le)
}

// Items iterator for the LeafVariants
func (lv *LeafVariants) Items() iter.Seq[*LeafEntry] {
	return func(yield func(*LeafEntry) bool) {
		for _, v := range lv.les {
			if !yield(v) {
				return
			}
		}
	}
}

func (lv *LeafVariants) Length() int {
	return len(lv.les)
}

// ShouldDelete indicates if the entry should be deleted,
// since it is an entry that represents LeafsVariants but non
// of these are still valid.
func (lv *LeafVariants) shouldDelete() bool {
	// only procede if we have leave variants
	if len(lv.les) == 0 {
		return false
	}

	// if only running exists return false
	if lv.les[0].Update.Owner() == RunningIntentName && len(lv.les) == 1 {
		return false
	}

	// go through all variants
	for _, l := range lv.les {
		// if not running is set and not the owner is running then
		// it should not be deleted
		if !(l.Delete || l.Update.Owner() == RunningIntentName) {
			return false
		}
	}

	return true
}

func (lv *LeafVariants) GetHighestPrecedenceValue() int32 {
	result := int32(math.MaxInt32)
	for _, e := range lv.les {
		if !e.Delete && e.Owner() != DefaultsIntentName && e.Update.Priority() < result {
			result = e.Update.Priority()
		}
	}
	return result
}

// GetHighesNewUpdated returns the LeafEntry with the highes priority
// nil if no leaf entry exists.
func (lv *LeafVariants) GetHighestPrecedence(onlyNewOrUpdated bool, includeDefaults bool) *LeafEntry {
	if len(lv.les) == 0 {
		return nil
	}
	if lv.shouldDelete() {
		return nil
	}

	var highest *LeafEntry
	var secondHighest *LeafEntry
	for _, e := range lv.les {
		// first entry set result to it
		// if it is not marked for deletion
		if highest == nil {
			highest = e
			continue
		}
		// on a result != nil that is then not marked for deletion
		// start comparing priorities and choose the one with the
		// higher prio (lower number)
		if highest.Priority() > e.Priority() {
			secondHighest = highest
			highest = e
		} else {
			// check if the update is at least higher prio (lower number) then the secondHighest
			if secondHighest == nil || secondHighest.Priority() > e.Priority() {
				secondHighest = e
			}
		}
	}

	// do not include defaults loaded at validation time
	if !includeDefaults && highest.Update.Owner() == DefaultsIntentName {
		return nil
	}

	// if it does not matter if the highes update is also
	// New or Updated return it
	if !onlyNewOrUpdated {
		if !highest.Delete {
			return highest
		}
		return secondHighest
	}

	// if the highes is not marked for deletion and new or updated (=PrioChanged) return it
	if !highest.Delete {
		if highest.IsNew || highest.IsUpdated || (lv.tc.actualOwner != "" && highest.Update.Owner() == lv.tc.actualOwner && lv.highestNotRunning(highest)) {
			return highest
		}
		return nil
	}
	// otherwise if the secondhighest is not marked for deletion return it
	if secondHighest != nil && !secondHighest.Delete && secondHighest.Update.Owner() != RunningIntentName {
		return secondHighest
	}

	// otherwise return nil
	return nil
}

func (lv *LeafVariants) highestNotRunning(highest *LeafEntry) bool {
	// if highes is already running or even default, return false
	if highest.Update.Owner() == RunningIntentName {
		return false
	}

	runVal := lv.GetByOwner(RunningIntentName)
	if runVal == nil {
		return true
	}

	// ignore errors, they should not happen :-P I know... should...
	rval, _ := runVal.Value()
	hval, _ := highest.Value()

	return !utils.EqualTypedValues(rval, hval)
}

// GetByOwner returns the entry that is owned by the given owner,
// returns nil if no entry exists.
func (lv *LeafVariants) GetByOwner(owner string) *LeafEntry {
	for _, e := range lv.les {
		if e.Owner() == owner {
			return e
		}
	}
	return nil
}
