package internal

import (
	"bufio"
	"encoding/json"
	"io"
	"sort"
	"strings"
	"time"
)

type testEvent struct {
	Time    time.Time // encodes as an RFC3339-format string
	Action  string
	Package string
	Test    string
	Elapsed float64 // seconds
	Output  string
}

var resultActions = []string{"pass", "fail"}

func (e *testEvent) key() string {
	return strings.Join([]string{e.Package, e.Test}, ":")
}

type testEvents []*testEvent

func (te testEvents) ByAction() testEventsMaps {
	m := testEventsMaps{}
	for _, ev := range te {
		m[ev.Action] = append(m[ev.Action], ev)
	}
	return m
}

func (te testEvents) byKey() testEventsMaps {
	m := testEventsMaps{}
	for _, ev := range te {
		m[ev.key()] = append(m[ev.key()], ev)
	}
	return m
}

func (te testEvents) withTest() testEvents {
	res := testEvents{}
	for _, event := range te {
		if event.Test != "" {
			res = append(res, event)
		}
	}
	return res
}

func (te testEvents) withPackage() testEvents {
	res := testEvents{}
	for _, event := range te {
		if event.Package != "" && event.Package != "command-line-arguments" {
			res = append(res, event)
		}
	}
	return res
}

func (te testEvents) result() *testEvent {
	for _, event := range te {
		for _, res := range resultActions {
			if event.Action == res {
				return event
			}
		}
	}
	return nil
}

type testEventsMaps map[string]testEvents

func (m testEventsMaps) sortedKeys() []string {
	keys := []string{}
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (m testEventsMaps) filterByResult(desiredResult string) testEventsMaps {
	out := make(testEventsMaps)
	for key, events := range m {
		event := events.result()
		if event == nil {
			continue
		}
		result := event.Action
		if result == desiredResult {
			out[key] = events
		}
	}
	return out
}

func parseEvents(reader io.Reader, passthrough io.Writer) testEvents {
	events := testEvents{}
	jsonScanner := bufio.NewScanner(reader)
	for jsonScanner.Scan() {
		event := new(testEvent)
		err := json.Unmarshal(jsonScanner.Bytes(), event)
		if err != nil {
			continue
		}
		if passthrough != nil && event.Action == "output" {
			_, err := passthrough.Write([]byte(event.Output))
			if err != nil {
				panic(err)
			}
		}

		events = append(events, event)
	}

	return events
}
