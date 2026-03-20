package main

import (
	"testing"
)

func TestProvider(t *testing.T) {
	p := Provider()

	if p == nil {
		t.Fatal("Provider() returned nil")
	}

	// Validate required provider schema attributes
	requiredSchema := map[string]bool{
		"host":  false,
		"token": false,
	}
	for name, attr := range p.Schema {
		if _, ok := requiredSchema[name]; ok {
			if !attr.Required {
				t.Errorf("Schema attribute %q should be required", name)
			}
			requiredSchema[name] = true
		}
	}
	for name, found := range requiredSchema {
		if !found {
			t.Errorf("Expected required schema attribute %q not found", name)
		}
	}

	// Validate required resources exist
	requiredResources := map[string]bool{
		"garage_key":        false,
		"garage_bucket":     false,
		"garage_bucket_key": false,
	}
	for name := range p.ResourcesMap {
		if _, ok := requiredResources[name]; ok {
			requiredResources[name] = true
		}
	}
	for name, found := range requiredResources {
		if !found {
			t.Errorf("Expected resource %q not found", name)
		}
	}
}
