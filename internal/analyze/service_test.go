package analyze

import (
	"reflect"
	"testing"
)

// TestMergeServicesRequestedFirst verifies that the requested service is always first.
func TestMergeServicesRequestedFirst(t *testing.T) {
	got := mergeServices("requested.example.com", []string{
		"api.example.com",
		"panel.example.com",
	})

	want := []string{
		"requested.example.com",
		"api.example.com",
		"panel.example.com",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

// TestMergeServicesRemovesRequestedDuplicate verifies that a requested service already present in config is not duplicated.
func TestMergeServicesRemovesRequestedDuplicate(t *testing.T) {
	got := mergeServices("panel.example.com", []string{
		"api.example.com",
		"panel.example.com",
		"cdn.example.com",
	})

	want := []string{
		"panel.example.com",
		"api.example.com",
		"cdn.example.com",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

// TestMergeServicesRemovesConfiguredDuplicates verifies duplicate configured services are removed while preserving order.
func TestMergeServicesRemovesConfiguredDuplicates(t *testing.T) {
	got := mergeServices("requested.example.com", []string{
		"api.example.com",
		"api.example.com",
		"cdn.example.com",
		"cdn.example.com",
	})

	want := []string{
		"requested.example.com",
		"api.example.com",
		"cdn.example.com",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}
