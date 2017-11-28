package main

import (
	"testing"
)

func TestSimple(t *testing.T) {
	testStr := `Domain: 'test'
  state.state=1234
  state.reason=4321`
	domains := getDomainStats([]byte(testStr))
	if len(domains) != 1 {
		t.Fatalf("Wrong domain count, got %d", len(domains))
	}

	dom := domains["test"]

	if dom.domain != "test" {
		t.Fatalf("Wrong domain name, got %s", dom.domain)
	}
	if len(dom.stats) != 1 {
		t.Fatalf("Wrong stat count, got %d", len(dom.stats))
	}

	stat := dom.stats[0]

	if len(stat.tags()) != 0 {
		t.Fatalf("Wrong tag count, got %d", len(stat.tags()))
	}
	if len(stat.values()) != 2 {
		t.Fatalf("Wrong value count, got %d", len(stat.values()))
	}

	if stat.values()["state"] != "1234" {
		t.Fatalf("Wrong state, got %s", stat.values()["state"])
	}

	if stat.values()["reason"] != "4321" {
		t.Fatalf("Wrong reason, got %s", stat.values()["reason"])
	}
}

func TestMulti(t *testing.T) {
	testStr := `Domain: 'test'
  state.state=1234
  state.reason=4321
  
Domain: 'test2'
  state.state=4321
  state.reason=1234`
	domains := getDomainStats([]byte(testStr))
	if len(domains) != 2 {
		t.Fatalf("Wrong domain count, got %d", len(domains))
	}

	dom := domains["test"]

	if dom.domain != "test" {
		t.Fatalf("Wrong domain name, got %s", dom.domain)
	}
	if len(dom.stats) != 1 {
		t.Fatalf("Wrong stat count, got %d", len(dom.stats))
	}

	stat := dom.stats[0]

	if len(stat.tags()) != 0 {
		t.Fatalf("Wrong tag count, got %d", len(stat.tags()))
	}
	if len(stat.values()) != 2 {
		t.Fatalf("Wrong value count, got %d", len(stat.values()))
	}

	if stat.values()["state"] != "1234" {
		t.Fatalf("Wrong state, got %s", stat.values()["state"])
	}

	if stat.values()["reason"] != "4321" {
		t.Fatalf("Wrong reason, got %s", stat.values()["reason"])
	}

	dom = domains["test2"]

	if dom.domain != "test2" {
		t.Fatalf("Wrong domain name, got %s", dom.domain)
	}
	if len(dom.stats) != 1 {
		t.Fatalf("Wrong stat count, got %d", len(dom.stats))
	}

	stat = dom.stats[0]

	if len(stat.tags()) != 0 {
		t.Fatalf("Wrong tag count, got %d", len(stat.tags()))
	}
	if len(stat.values()) != 2 {
		t.Fatalf("Wrong value count, got %d", len(stat.values()))
	}

	if stat.values()["state"] != "4321" {
		t.Fatalf("Wrong state, got %s", stat.values()["state"])
	}

	if stat.values()["reason"] != "1234" {
		t.Fatalf("Wrong reason, got %s", stat.values()["reason"])
	}
}
