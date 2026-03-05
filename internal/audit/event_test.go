package audit

import (
	"encoding/json"
	"testing"
)

func TestBuildEvent(t *testing.T) {
	ev := BuildEvent(EventOpts{
		Action:     "user-login",
		Category:   []string{"authentication"},
		Type:       []string{"access"},
		Outcome:    "success",
		Message:    "login success",
		User:       &UserFields{Name: "alice"},
		URL:        "http://localhost/api/login",
		ClientIP:   "192.168.1.1",
		StatusCode: 200,
		Method:     "POST",
	})
	if ev.Event.Action != "user-login" {
		t.Errorf("Event.Action = %q", ev.Event.Action)
	}
	if ev.Event.Outcome != "success" {
		t.Errorf("Event.Outcome = %q", ev.Event.Outcome)
	}
	if ev.Message != "login success" {
		t.Errorf("Message = %q", ev.Message)
	}
	if ev.User == nil || ev.User.Name != "alice" {
		t.Errorf("User = %+v", ev.User)
	}
	if ev.URL == nil || ev.URL.Full != "http://localhost/api/login" {
		t.Errorf("URL = %+v", ev.URL)
	}
	if ev.Client == nil || ev.Client.IP != "192.168.1.1" {
		t.Errorf("Client = %+v", ev.Client)
	}
	if ev.HTTP == nil || ev.HTTP.RequestMethod != "POST" || ev.HTTP.ResponseStatusCode != 200 {
		t.Errorf("HTTP = %+v", ev.HTTP)
	}
}

func TestBuildEvent_minimal(t *testing.T) {
	ev := BuildEvent(EventOpts{Action: "test", Message: "msg"})
	if ev.Event.Action != "test" {
		t.Errorf("Action = %q", ev.Event.Action)
	}
	if ev.URL != nil || ev.Client != nil || ev.HTTP != nil {
		t.Error("optional fields should be nil when not set")
	}
}

func TestBuildEvent_serializesToJSON(t *testing.T) {
	ev := BuildEvent(EventOpts{
		Action: "file-upload",
		User:   &UserFields{Name: "bob"},
		File:   &FileFields{Path: "/files/1/default/x.jpg", Size: 1024},
	})
	b, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var decoded Event
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if decoded.Event.Action != ev.Event.Action {
		t.Errorf("decoded Action = %q", decoded.Event.Action)
	}
}
