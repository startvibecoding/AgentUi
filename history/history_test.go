package history

import "testing"

func TestHistoryNavigation(t *testing.T) {
	m := New(3)
	m = m.Record("one")
	m = m.Record("two")
	m = m.Record("three")

	var got string
	var ok bool
	m, got, ok = m.Prev("draft")
	if !ok || got != "three" {
		t.Fatalf("prev1 = %q, %v", got, ok)
	}
	m, got, ok = m.Prev(got)
	if !ok || got != "two" {
		t.Fatalf("prev2 = %q, %v", got, ok)
	}
	m, got, ok = m.Next(got)
	if !ok || got != "three" {
		t.Fatalf("next1 = %q, %v", got, ok)
	}
	m, got, ok = m.Next(got)
	if !ok || got != "draft" {
		t.Fatalf("next2 = %q, %v", got, ok)
	}
}

func TestHistoryLimitAndDedup(t *testing.T) {
	m := New(2)
	m = m.Record("one")
	m = m.Record("one")
	m = m.Record("two")
	m = m.Record("three")
	items := m.Items()
	if len(items) != 2 || items[0] != "two" || items[1] != "three" {
		t.Fatalf("items = %#v", items)
	}
}
