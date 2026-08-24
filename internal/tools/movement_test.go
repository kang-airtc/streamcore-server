package tools

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestContinuousOnlyOnTravellingTools(t *testing.T) {
	// Client-side, `continuous` only feeds the distance calculation, so these
	// are the only actions that can honour it.
	honours := map[string]bool{
		"movement.forward": true, "movement.backward": true,
		"movement.pivot_forward_left": true, "movement.pivot_forward_right": true,
		"movement.pivot_back_left": true, "movement.pivot_back_right": true,
	}
	for _, tool := range Movement() {
		var schema struct {
			Properties map[string]json.RawMessage `json:"properties"`
		}
		if err := json.Unmarshal(tool.Parameters(), &schema); err != nil {
			t.Fatalf("%s: bad schema: %v", tool.Name(), err)
		}
		_, offered := schema.Properties["continuous"]
		if offered != honours[tool.Name()] {
			t.Errorf("%s: offers continuous=%v, honoured=%v", tool.Name(), offered, honours[tool.Name()])
		}
		if !strings.HasPrefix(tool.Name(), "movement.") {
			t.Errorf("%s: not namespaced under movement.", tool.Name())
		}
	}
}

// The advertised defaults tell the model what it gets when it omits an
// argument, so they have to match what the clients actually fall back to —
// defaultsFor in the voice bot, parse_movement_command in the firmware. Both
// clients agree on this table; if it changes, change it in all three.
func TestAdvertisedDefaultsMatchClients(t *testing.T) {
	want := map[string][2]int{ // duration_ms, speed_percent
		"movement.forward": {1500, 80}, "movement.backward": {1500, 80},
		"movement.turn_left": {1500, 80}, "movement.turn_right": {1500, 80},
		"movement.pivot_forward_left": {1500, 80}, "movement.pivot_forward_right": {1500, 80},
		"movement.pivot_back_left": {1500, 80}, "movement.pivot_back_right": {1500, 80},
		"movement.fancy": {3000, 80},
		"movement.shake": {1200, 90},
	}
	for _, tool := range Movement() {
		expected, ok := want[tool.Name()]
		if !ok {
			if tool.Name() != "movement.stop" {
				t.Errorf("%s: no expected defaults recorded", tool.Name())
			}
			continue
		}
		var schema struct {
			// Per-property, because `continuous` defaults to a bool.
			Properties map[string]struct {
				Default json.RawMessage `json:"default"`
			} `json:"properties"`
		}
		if err := json.Unmarshal(tool.Parameters(), &schema); err != nil {
			t.Fatalf("%s: %v", tool.Name(), err)
		}
		intDefault := func(field string) int {
			var v int
			if err := json.Unmarshal(schema.Properties[field].Default, &v); err != nil {
				t.Fatalf("%s: %s default: %v", tool.Name(), field, err)
			}
			return v
		}
		if got := intDefault("duration_ms"); got != expected[0] {
			t.Errorf("%s: duration_ms default %d, clients use %d", tool.Name(), got, expected[0])
		}
		if got := intDefault("speed_percent"); got != expected[1] {
			t.Errorf("%s: speed_percent default %d, clients use %d", tool.Name(), got, expected[1])
		}
	}
}

// The schema's advertised bounds and the clamp the pipeline applies come from
// the same constants; this checks the schemas really do carry them, so a
// hand-edited "minimum" cannot quietly disagree with what the model is given.
func TestSchemaBoundsMatchConstants(t *testing.T) {
	check := func(t *testing.T, name string, params []byte, min, max int) {
		t.Helper()
		var schema struct {
			Properties map[string]struct {
				Minimum *int `json:"minimum"`
				Maximum *int `json:"maximum"`
			} `json:"properties"`
		}
		if err := json.Unmarshal(params, &schema); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		d, ok := schema.Properties["duration_ms"]
		if !ok {
			return // stop takes no arguments
		}
		if d.Minimum == nil || *d.Minimum != min {
			t.Errorf("%s: duration_ms minimum %v, clamp uses %d", name, d.Minimum, min)
		}
		if d.Maximum == nil || *d.Maximum != max {
			t.Errorf("%s: duration_ms maximum %v, clamp uses %d", name, d.Maximum, max)
		}
	}
	for _, tool := range Movement() {
		check(t, tool.Name(), tool.Parameters(), MinMovementMs, MaxMovementMs)
	}
	for _, tool := range Gestures() {
		check(t, tool.Name(), tool.Parameters(), MinGestureMs, MaxGestureMs)
	}
}
