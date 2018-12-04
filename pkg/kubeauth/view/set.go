package view

type set struct {
	index map[string]struct{}
}

func (o *set) has(value string) bool {
	_, exist := o.index[value]
	return exist
}

func (o *set) put(value string) {
	o.index[value] = struct{}{}
}

func (o *set) del(value string) {
	delete(o.index, value)
}

func (o *set) deepCopy() *set {
	if o == nil {
		return nil
	}

	ret := *o

	return &ret
}

func newSet() *set {
	return &set{
		index: make(map[string]struct{}, 8),
	}
}
