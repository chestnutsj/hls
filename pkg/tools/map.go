package tools

import (
	"container/list"
	"sort"
	"strings"
	"sync"
)

type OrderMap interface {
	Set(key string, value interface{}) bool
	Get(key string) (interface{}, bool)
	Delete(key string) bool
	Keys() []string
	Values() []interface{}
	SortKeys()
	Fetch(f func(interface{}) error, force bool) error
}

type orderedMap struct {
	mu   sync.RWMutex
	data map[string]*list.Element
	list *list.List
}

type entry struct {
	key   string
	value interface{}
}

func NewOrderedMap() OrderMap {
	return &orderedMap{
		data: make(map[string]*list.Element),
		list: list.New(),
	}
}

func (om *orderedMap) Set(key string, value interface{}) bool {
	om.mu.Lock()
	defer om.mu.Unlock()

	if e, exists := om.data[key]; exists {
		e.Value.(*entry).value = value
		return true
	} else {
		e := om.list.PushBack(&entry{key, value})
		om.data[key] = e
		return false
	}
}

func (om *orderedMap) Get(key string) (interface{}, bool) {
	om.mu.RLock()
	defer om.mu.RUnlock()

	if e, exists := om.data[key]; exists {
		return e.Value.(*entry).value, true
	}
	return nil, false
}

// Delete 删除键值对
func (om *orderedMap) Delete(key string) bool {
	om.mu.Lock()
	defer om.mu.Unlock()

	if e, exists := om.data[key]; exists {
		om.list.Remove(e)
		delete(om.data, key)
		return true
	}
	return false
}

func (om *orderedMap) Keys() []string {
	om.mu.RLock()
	defer om.mu.RUnlock()

	keys := make([]string, 0, om.list.Len())
	for e := om.list.Front(); e != nil; e = e.Next() {
		keys = append(keys, e.Value.(*entry).key)
	}
	return keys
}

func (om *orderedMap) Values() []interface{} {
	om.mu.RLock()
	defer om.mu.RUnlock()

	values := make([]interface{}, 0, om.list.Len())
	for e := om.list.Front(); e != nil; e = e.Next() {

		values = append(values, e.Value.(*entry).value)
	}
	return values
}

func (om *orderedMap) SortKeys() {
	om.mu.Lock()
	defer om.mu.Unlock()
	type pair struct {
		key string
		el  *list.Element
	}
	pairs := make([]pair, 0, om.list.Len())
	for e := om.list.Front(); e != nil; e = e.Next() {
		pairs = append(pairs, pair{key: e.Value.(*entry).key, el: e})
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].key < pairs[j].key
	})
	om.list.Init()
	om.data = make(map[string]*list.Element)
	for _, p := range pairs {
		e := om.list.PushBack(&entry{key: p.key, value: p.el.Value.(*entry).value})
		om.data[p.key] = e
	}
}

func (om *orderedMap) IndexOf(key string) int {
	om.mu.RLock()
	defer om.mu.RUnlock()

	if e, exists := om.data[key]; exists {
		index := 0
		for el := om.list.Front(); el != nil; el = el.Next() {
			if el == e {
				return index
			}
			index++
		}
	}
	return -1
}

func (om *orderedMap) Fetch(f func(interface{}) error, force bool) error {
	om.mu.RLock()
	defer om.mu.RUnlock()
	var lastErr error
	for e := om.list.Front(); e != nil; e = e.Next() {
		err := f(e.Value.(*entry).value)
		if err != nil {
			if force {
				lastErr = err
			} else {
				return err
			}
		}
	}
	return lastErr
}

type CaseInsensitiveMap interface {
	Set(key string, value interface{})
	Get(key string) (interface{}, bool)
	Delete(key string)
	Range(f func(key string, value interface{}) bool)
}
type keyValue struct {
	key string
	val interface{}
}

type caseInsensitiveMap struct {
	m map[string]keyValue
}

func NewCaseInsensitiveMap() CaseInsensitiveMap {
	return &caseInsensitiveMap{
		m: make(map[string]keyValue),
	}
}

func (c *caseInsensitiveMap) Set(key string, value interface{}) {
	c.m[strings.ToLower(key)] = keyValue{
		key: key,
		val: value,
	}
}

func (c *caseInsensitiveMap) Get(key string) (interface{}, bool) {
	x, exist := c.m[strings.ToLower(key)]
	if exist {
		return x.val, true
	} else {
		return nil, false
	}
}

func (c *caseInsensitiveMap) Delete(key string) {
	delete(c.m, strings.ToLower(key))
}

func (c *caseInsensitiveMap) Range(f func(key string, value interface{}) bool) {
	for _, v := range c.m {
		if !f(v.key, v.val) {
			break
		}
	}
}
