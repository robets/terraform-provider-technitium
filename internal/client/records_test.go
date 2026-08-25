// Copyright (c) 2026 Alex Ackerman, Russell Obets
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestRecordValueParam_FWD(t *testing.T) {
	if got := RecordValueParam("FWD"); got != "forwarder" {
		t.Fatalf("RecordValueParam(FWD) = %q, want forwarder", got)
	}
}

func TestRecordValueFromRData_FWD(t *testing.T) {
	rdata := map[string]interface{}{"forwarder": "dns.quad9.net:853 (9.9.9.9)"}
	if got := RecordValueFromRData("FWD", rdata); got != "dns.quad9.net:853 (9.9.9.9)" {
		t.Fatalf("RecordValueFromRData(FWD) = %q", got)
	}
}

func TestRecordLifecycle_APP(t *testing.T) {
	requests := 0
	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		query := r.URL.Query()
		if query.Get("domain") != "app.example.com" || query.Get("zone") != "example.com" || query.Get("type") != "APP" {
			t.Fatalf("unexpected APP query: %s", r.URL.RawQuery)
		}

		switch r.URL.Path {
		case "/api/zones/records/add", "/api/zones/records/update":
			if query.Get("appName") != "Split Horizon" || query.Get("classPath") != "SplitHorizon.SimpleAddress" {
				t.Fatalf("missing APP handler identity: %s", r.URL.RawQuery)
			}
			if query.Get("recordData") != `{"private":["10.1.2.3"]}` {
				t.Fatalf("recordData = %q", query.Get("recordData"))
			}
		case "/api/zones/records/delete":
			if query.Has("appName") || query.Has("classPath") || query.Has("recordData") {
				t.Fatalf("Technitium APP deletion is RRset-based; unexpected RDATA identity: %s", r.URL.RawQuery)
			}
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}

		response := json.RawMessage(`{}`)
		if r.URL.Path == "/api/zones/records/add" {
			response = json.RawMessage(`{"addedRecord":{"name":"app.example.com","type":"APP","ttl":300,"rData":{"appName":"Split Horizon","classPath":"SplitHorizon.SimpleAddress","data":"{\"private\":[\"10.1.2.3\"]}"}}}`)
		}
		if err := json.NewEncoder(w).Encode(APIResponse{Status: "ok", Response: response}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	})
	defer ts.Close()

	c, err := NewClient(ClientConfig{BaseURL: ts.URL, Token: "test-token"})
	if err != nil {
		t.Fatal(err)
	}
	params := map[string]string{
		"appName":    "Split Horizon",
		"classPath":  "SplitHorizon.SimpleAddress",
		"recordData": `{"private":["10.1.2.3"]}`,
	}
	if _, err := c.RecordAdd(context.Background(), "app.example.com", "example.com", "APP", 300, false, params); err != nil {
		t.Fatal(err)
	}
	if err := c.RecordUpdate(context.Background(), "app.example.com", "example.com", "APP", 300, params); err != nil {
		t.Fatal(err)
	}
	if err := c.RecordDelete(context.Background(), "app.example.com", "example.com", "APP", nil); err != nil {
		t.Fatal(err)
	}
	if requests != 3 {
		t.Fatalf("requests = %d, want 3", requests)
	}
}
