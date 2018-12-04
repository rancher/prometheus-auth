package view

import (
	"sort"
)

type SetView interface {
	Put(value string)
	Del(value string)
	Has(value string) bool
	GetAll() []string
}

type setView struct {
	set *set
}

func (o *setView) Put(value string) {
	o.set.put(value)
}

func (o *setView) Del(value string) {
	o.set.del(value)
}

func (o *setView) Has(value string) bool {
	set := o.set.deepCopy()

	return set.has(value)
}

func (o *setView) GetAll() []string {
	ret := make([]string, 0, 16)

	set := o.set.deepCopy()
	for key := range set.index {
		ret = append(ret, key)
	}

	sort.Strings(ret)

	return ret
}

func NewSetView() SetView {
	return &setView{
		set: newSet(),
	}
}
