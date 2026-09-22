package onlyoffice

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalResponseObjectMap(t *testing.T) {
	raw := json.RawMessage(`{"response":{"id":"1","isAdmin":true}}`)
	out, err := unmarshalResponseObject(raw)
	if err != nil {
		t.Fatal(err)
	}
	if out["id"] != "1" || out["isAdmin"] != true {
		t.Fatalf("unexpected map: %#v", out)
	}
}

func TestUnmarshalResponseObjectArray(t *testing.T) {
	raw := json.RawMessage(`{"response":[{"id":"a"},{"id":"b"}]}`)
	out, err := unmarshalResponseObject(raw)
	if err != nil {
		t.Fatal(err)
	}
	if out["id"] != "a" {
		t.Fatalf("expected first array element, got %#v", out)
	}
}

func TestUnmarshalResponseObjectEmptyArray(t *testing.T) {
	raw := json.RawMessage(`{"response":[]}`)
	out, err := unmarshalResponseObject(raw)
	if err != nil {
		t.Fatal(err)
	}
	if out != nil {
		t.Fatalf("expected nil for empty array, got %#v", out)
	}
}

func TestUnmarshalResponseObjectNull(t *testing.T) {
	raw := json.RawMessage(`{"response":null}`)
	out, err := unmarshalResponseObject(raw)
	if err != nil {
		t.Fatal(err)
	}
	if out != nil {
		t.Fatalf("expected nil, got %#v", out)
	}
}

func TestUnmarshalResponseArrayList(t *testing.T) {
	raw := json.RawMessage(`{"response":[{"id":"a","displayName":"A"},{"id":"b","displayName":"B"}]}`)
	out, err := unmarshalResponseArray(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0]["id"] != "a" || out[1]["displayName"] != "B" {
		t.Fatalf("unexpected list: %#v", out)
	}
}

func TestUnmarshalResponseArrayNullAndScalar(t *testing.T) {
	for _, raw := range []string{`{"response":null}`, `{"response":{}}`, `{"response":"x"}`} {
		out, err := unmarshalResponseArray(json.RawMessage(raw))
		if err != nil {
			t.Fatalf("%s: %v", raw, err)
		}
		if out != nil {
			t.Fatalf("%s: expected nil, got %#v", raw, out)
		}
	}
}
