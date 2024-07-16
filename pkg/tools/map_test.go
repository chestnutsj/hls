package tools

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

func TestOrderedMap_Set(t *testing.T) {
	om := NewOrderedMap()
	key := "testKey"
	value := "testValue"

	setResult := om.Set(key, value)
	if setResult {
		t.Errorf("Set should return true when setting a new key-value pair")
	}

	_, exists := om.Get(key)
	if !exists {
		t.Errorf("Get should return true and the correct value when the key exists")
	}
}

func TestOrderedMap_Get(t *testing.T) {
	om := NewOrderedMap()
	key := "testKey"
	value := "testValue"
	om.Set(key, value)

	retrievedValue, exists := om.Get(key)
	if !exists || retrievedValue != value {
		t.Errorf("Get should return the correct value and true if it exists")
	}
}

func TestOrderedMap_Delete(t *testing.T) {
	om := NewOrderedMap()
	key := "testKey"
	value := "testValue"
	om.Set(key, value)

	deleteResult := om.Delete(key)
	if !deleteResult {
		t.Errorf("Delete should return true when deleting an existing key")
	}

	_, exists := om.Get(key)
	if exists {
		t.Errorf("The key should not exist after deletion")
	}
}

func TestOrderedMap_Keys(t *testing.T) {
	om := NewOrderedMap()
	keys := []string{"a", "b", "c"}
	for _, key := range keys {
		om.Set(key, key)
	}

	retrievedKeys := om.Keys()
	if !reflect.DeepEqual(retrievedKeys, keys) {
		t.Errorf("Keys should return the correct ordered keys")
	}
}

func TestOrderedMap_Values(t *testing.T) {
	om := NewOrderedMap()
	values := []interface{}{"a", "b", "c"}
	for i, value := range values {
		om.Set(fmt.Sprintf("%d", i), value)
	}

	retrievedValues := om.Values()
	if !reflect.DeepEqual(retrievedValues, values) {
		t.Errorf("Values should return the correct ordered values")
	}
}

func TestOrderedMap_SortKeys(t *testing.T) {
	om := NewOrderedMap()
	keys := []string{"c", "b", "a"}
	for _, key := range keys {
		om.Set(key, key)
	}

	om.SortKeys()

	sortedKeys := om.Keys()
	sort.Strings(keys) // 对预期结果进行排序以比较
	if !reflect.DeepEqual(sortedKeys, keys) {
		t.Errorf("SortKeys should sort the keys in ascending order")
	}
}
