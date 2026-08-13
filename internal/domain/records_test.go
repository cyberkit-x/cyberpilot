package domain

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestIDRoundTripAndVersion(t *testing.T) {
	id := MustNewID()
	parsed, err := ParseID(string(id))
	if err != nil {
		t.Fatal(err)
	}
	if parsed != id {
		t.Fatalf("ParseID() = %q, want %q", parsed, id)
	}
	if _, err := ParseID("not-an-id"); err == nil {
		t.Fatal("expected malformed ID to fail")
	}
}

func TestEventJSONRoundTrip(t *testing.T) {
	sessionID := MustNewID()
	want, err := NewEvent(sessionID, 7, "session.created", time.Date(2026, 8, 12, 9, 1, 2, 3, time.FixedZone("test", 3600)), map[string]string{"objective": "test"})
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var got Event
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.SchemaVersion != EventSchemaVersion || got.OccurredAt.Location() != time.UTC {
		t.Fatalf("unexpected normalized event: %#v", got)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestContractJSONCompatibility(t *testing.T) {
	proposal := ActionProposal{
		ID: MustNewID(), SessionID: MustNewID(), HypothesisID: MustNewID(),
		Target: "https://example.test", Purpose: "read health endpoint", Capability: "http.request",
		Arguments: json.RawMessage(`{"method":"GET"}`), Risk: Risk{Level: "low", TrafficClass: "single"},
		ExpectedEvidence: []string{"status", "body hash"}, TimeoutSeconds: 10,
	}
	data, err := json.Marshal(proposal)
	if err != nil {
		t.Fatal(err)
	}
	var got ActionProposal
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, proposal) {
		t.Fatalf("contract round trip mismatch: %#v != %#v", got, proposal)
	}
}
