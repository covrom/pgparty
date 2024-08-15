package pgparty

import (
	"fmt"
	"sync"
)

var uoPool = sync.Pool{}

func getMOSlice(c int) []ModelObject {
	sl := uoPool.Get()
	if sl != nil {
		vsl := sl.([]ModelObject)
		if cap(vsl) >= c {
			return vsl
		}
	}
	return make([]ModelObject, 0, c)
}

func putMOSlice(sl []ModelObject) {
	if sl == nil {
		return
	}
	uoPool.Put(sl[:0])
}

type UniqueObjects struct {
	objs []ModelObject
	uniq map[any]int
}

func NewUniqueObjects() *UniqueObjects {
	return &UniqueObjects{
		objs: getMOSlice(16),
		uniq: make(map[any]int),
	}
}

func (uo *UniqueObjects) Object(id any) (ModelObject, bool) {
	if i, ok := uo.uniq[id]; ok {
		return uo.objs[i], true
	}
	return ModelObject{}, false
}

func (uo *UniqueObjects) Objects() []ModelObject {
	return uo.objs
}

func (uo *UniqueObjects) CopyObjects() (res []ModelObject) {
	res = make([]ModelObject, len(uo.objs))
	copy(res, uo.objs)
	return
}

func (uo *UniqueObjects) PutObject(data ModelObject) error {
	id := data.FieldID()
	if id == nil {
		return fmt.Errorf("UniqueObjects.AddObject: data doesn't have id value")
	}
	if i, ok := uo.uniq[id]; ok {
		uo.objs[i] = data
		return nil
	}
	i := len(uo.objs)
	uo.objs = append(uo.objs, data)
	uo.uniq[id] = i
	return nil
}

func (uo *UniqueObjects) Close() {
	putMOSlice(uo.objs)
	uo.objs = nil
	uo.uniq = nil
}

func (uo *UniqueObjects) Reset() {
	for i := range uo.objs {
		uo.objs[i] = ModelObject{}
	}
	uo.objs = uo.objs[:0]
	for k := range uo.uniq {
		delete(uo.uniq, k)
	}
}
