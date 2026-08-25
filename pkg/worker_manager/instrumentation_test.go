package workermanager

import "testing"

func TestSetGetParam_RoundTripsAnyType(t *testing.T) {
	t.Parallel()

	params := NewTaskParams()
	SetParam(params, "name", "widget")
	SetParam(params, "quantity", 3)
	SetParam(params, "price", 9.99)
	SetParam(params, "tags", []string{"a", "b"})

	name, ok := GetParam[string](params, "name")
	if !ok || name != "widget" {
		t.Fatalf("GetParam[string](name) = %q, %v; want \"widget\", true", name, ok)
	}

	quantity, ok := GetParam[int](params, "quantity")
	if !ok || quantity != 3 {
		t.Fatalf("GetParam[int](quantity) = %d, %v; want 3, true", quantity, ok)
	}

	price, ok := GetParam[float64](params, "price")
	if !ok || price != 9.99 {
		t.Fatalf("GetParam[float64](price) = %v, %v; want 9.99, true", price, ok)
	}

	tags, ok := GetParam[[]string](params, "tags")
	if !ok || len(tags) != 2 || tags[0] != "a" {
		t.Fatalf("GetParam[[]string](tags) = %v, %v; want [a b], true", tags, ok)
	}
}

func TestGetParam_MissingKeyReturnsZeroValueAndFalse(t *testing.T) {
	t.Parallel()

	params := NewTaskParams()
	value, ok := GetParam[string](params, "missing")
	if ok || value != "" {
		t.Fatalf("GetParam(missing) = %q, %v; want \"\", false", value, ok)
	}
}

func TestGetParam_WrongTypeReturnsZeroValueAndFalse(t *testing.T) {
	t.Parallel()

	params := NewTaskParams()
	SetParam(params, "count", 42)

	value, ok := GetParam[string](params, "count")
	if ok || value != "" {
		t.Fatalf("GetParam[string](count) = %q, %v; want \"\", false (stored value is an int)", value, ok)
	}
}

func TestClone_IsIndependentOfTheOriginal(t *testing.T) {
	t.Parallel()

	original := NewTaskParams()
	SetParam(original, "key", "original")

	clone := original.Clone()
	SetParam(clone, "key", "mutated")
	SetParam(clone, "extra", "only-in-clone")

	originalValue, _ := GetParam[string](original, "key")
	if originalValue != "original" {
		t.Fatalf("original was mutated through its clone: got %q", originalValue)
	}

	if _, ok := GetParam[string](original, "extra"); ok {
		t.Fatalf("original gained a key set only on its clone")
	}
}

func TestParams_ReturnsACopyNotTheInternalMap(t *testing.T) {
	t.Parallel()

	params := NewTaskParams()
	SetParam(params, "key", "value")

	snapshot := params.Params()
	snapshot["key"] = "mutated-in-snapshot"
	snapshot["new"] = "added-in-snapshot"

	value, _ := GetParam[string](params, "key")
	if value != "value" {
		t.Fatalf("mutating the map returned by Params() affected the original: got %q", value)
	}
	if _, ok := GetParam[string](params, "new"); ok {
		t.Fatalf("original gained a key added only to the Params() snapshot")
	}
}
