package atm

import (
	"testing"
	"time"
)

func TestExpiringCache(t *testing.T) {
	c := NewExpiringCache(1 * time.Second)
	c.Set("a", "one", 2*time.Second)
	a, found := c.Get("a")
	if !found {
		t.Error("Was not able to find a right after being set")
	}
	if "one" != a.(string) {
		t.Error("Found a was not the exected a", a)
	}
	<-time.After(5 * time.Second)
	a, found = c.Get("a")
	if found {
		t.Error("Should not have found a after expiration", a)
	}
	c.Set("a", "three", 5*time.Minute)
	a, found = c.Get("a")
	if !found {
		t.Error("Was not able to find a after being reset")
	}
	if "three" != a.(string) {
		t.Error("Found a was not the exected a", a)
	}
}

func TestExpiringCacheLong(t *testing.T) {
	c := NewExpiringCache(1 * time.Hour)
	c.Set("a", "one", 1*time.Second)
	c.Set("b", "two", 1*time.Hour)
	a, found := c.Get("a")
	if !found {
		t.Error("Was not able to find a right after being set")
	}
	if "one" != a.(string) {
		t.Error("Found a was not the exected a", a)
	}
	<-time.After(5 * time.Second)
	a, found = c.Get("a")
	if found {
		t.Error("Should not have found a after expiration", a)
	}
	var b interface{}
	b, found = c.Get("b")
	if !found {
		t.Error("Was not able to find b after being set")
	}
	if "two" != b.(string) {
		t.Error("Found b was not the exected b", b)
	}
}
