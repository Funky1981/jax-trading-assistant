package triage

import (
	"sort"
	"time"
)

type Queue struct {
	items []Item
}

func NewQueue(items []Item) Queue {
	copied := append([]Item{}, items...)
	return Queue{items: copied}
}

func (q Queue) OpenItems(asOf time.Time) ([]Item, error) {
	var open []Item
	for _, item := range q.items {
		if item.Status != StatusOpen {
			continue
		}
		if !item.DueAt.IsZero() && item.DueAt.After(asOf) {
			continue
		}
		if err := ValidateItem(item); err != nil {
			return nil, err
		}
		open = append(open, item)
	}
	sort.SliceStable(open, func(i, j int) bool {
		leftRank := priorityRank(open[i].Priority)
		rightRank := priorityRank(open[j].Priority)
		if leftRank != rightRank {
			return leftRank > rightRank
		}
		return open[i].DueAt.Before(open[j].DueAt)
	})
	return open, nil
}
