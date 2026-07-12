package wmi

import (
	"reflect"
	"testing"
)

func TestParseList(t *testing.T) {
	raw := "Name=Test\r\nValue=123\r\n\r\nName=Test2\r\nValue=456\r\n\r\n"
	records := ParseList(raw)

	if len(records) != 2 {
		t.Fatalf("Expected 2 records, got %d", len(records))
	}

	expected1 := map[string]string{"Name": "Test", "Value": "123"}
	if !reflect.DeepEqual(records[0], expected1) {
		t.Errorf("First record mismatch: expected %v, got %v", expected1, records[0])
	}

	expected2 := map[string]string{"Name": "Test2", "Value": "456"}
	if !reflect.DeepEqual(records[1], expected2) {
		t.Errorf("Second record mismatch: expected %v, got %v", expected2, records[1])
	}
}

func TestParseListEmpty(t *testing.T) {
	raw := "\r\n\r\n"
	records := ParseList(raw)
	if len(records) != 0 {
		t.Errorf("Expected 0 records, got %d", len(records))
	}
}
