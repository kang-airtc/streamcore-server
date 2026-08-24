// Package tools holds native Go plugins compiled into the server.
//
// Movement tools drive whatever the connected client happens to be — a
// two-wheel desktop car, a rigged character in a browser, anything else that
// can go forward and turn. The server never learns which, so nothing here
// names a drivetrain.
//
// The tools only describe the surface; Execute is a no-op. The real work
// happens in the pipeline package, which intercepts any "movement.*" tool call
// and writes a topic-addressed data-channel packet to the device. Same pattern
// as vision.analyze.
package tools

import (
	"encoding/json"
	"fmt"

	"github.com/streamcoreai/streamcore-server/internal/plugin"
)

// MovementCommandTopic is the data-channel topic a client listens on for
// locomotion commands. Keep in sync with each client's on_data handler.
const MovementCommandTopic = "movement.command"

// Duration limits for the movement tools. The pipeline clamps to these and the
// schemas advertise them, both from here — telling the model one range and
// applying another is a lie it has no way to notice.
const (
	MinMovementMs = 100
	MaxMovementMs = 10000
)

// MovementTool is a metadata-only Tool. Its Execute method exists to satisfy
// the plugin.Tool interface; the pipeline never reaches it for "movement.*"
// names because it intercepts those calls and emits a data-channel
// packet directly.
type MovementTool struct {
	name        string
	description string
	parameters  json.RawMessage
}

func (t *MovementTool) Name() string                { return t.name }
func (t *MovementTool) Description() string         { return t.description }
func (t *MovementTool) Parameters() json.RawMessage { return t.parameters }
func (t *MovementTool) ConfirmationRequired() bool  { return false }
func (t *MovementTool) ThinkingSound() bool         { return false }

// Execute is never called in production — pipeline.go intercepts.
func (t *MovementTool) Execute(params json.RawMessage) (string, error) {
	return "", fmt.Errorf("movement.* tool calls must be intercepted by the pipeline")
}

// Movement returns the full set of locomotion tools to register with the
// plugin manager. Wire them in `main.go` with `pluginMgr.RegisterNative`.
func Movement() []plugin.Tool {
	// Only the actions that actually travel accept `continuous`. A turn or a
	// canned routine advertising it would let the model set the flag, be told
	// it worked, and leave the robot doing a normal timed move instead — the
	// exact mismatch that made "keep walking" report success while standing
	// still.
	//
	// The defaults below are what the clients fall back to when the model omits
	// the argument (see defaultsFor in the voice bot, parse_movement_command in
	// the firmware). They are advertised so the model can predict what it gets
	// for free; advertising a number nobody uses just makes it reason wrongly.
	travel := paramsSchema("How long to keep moving, in milliseconds. Ignored when continuous is true.", 1500, 80, true)
	turn := paramsSchema("How long to keep turning, in milliseconds.", 1500, 80, false)
	fancy := paramsSchema("Total length of the fancy-moves routine in milliseconds.", 3000, 80, false)
	shake := paramsSchema("How long to keep shaking, in milliseconds.", 1200, 90, false)

	// Descriptions stay device-neutral: the same tools drive a two-wheel car
	// and walk a rigged character, and the server has no idea which is on the
	// other end. Say "drive" here and a walking bot narrates itself as a car.
	// The same reasoning is why these are movement.* and not car.*.
	return []plugin.Tool{
		&MovementTool{"movement.forward", "Move the robot forward. Use when the user asks it to go, move, walk, drive, advance, or come closer.", travel},
		&MovementTool{"movement.backward", "Move the robot backward. Use when the user asks it to back up, reverse, or move away.", travel},
		&MovementTool{"movement.turn_left", "Turn the robot in place to its own left (counter-clockwise). Use when the user asks it to turn left.", turn},
		&MovementTool{"movement.turn_right", "Turn the robot in place to its own right (clockwise). Use when the user asks it to turn right.", turn},
		&MovementTool{"movement.pivot_forward_left", "Curve forward and to the left instead of turning on the spot. Gentler than a turn-in-place.", travel},
		&MovementTool{"movement.pivot_forward_right", "Curve forward and to the right instead of turning on the spot. Gentler than a turn-in-place.", travel},
		&MovementTool{"movement.pivot_back_left", "Curve backward and to the left instead of turning on the spot.", travel},
		&MovementTool{"movement.pivot_back_right", "Curve backward and to the right instead of turning on the spot.", travel},
		&MovementTool{"movement.stop", "Stop moving immediately. Use when the user asks the robot to stop, halt, freeze, or wait.", emptySchema()},
		&MovementTool{"movement.fancy", "Play a short choreographed sequence of fancy moves — alternating pivots and wiggles. Use when the user asks the robot to do fancy moves, show off, dance, party, celebrate, do tricks, or otherwise put on a little show.", fancy},
		&MovementTool{"movement.shake", "Quick left-right wiggle in place. Use when the user asks the robot to shake, wiggle, or nod no.", shake},
	}
}

// paramsSchema builds the argument schema shared by the timed actions.
// `continuous` is only offered when the caller can honour it.
func paramsSchema(durationDesc string, defaultDuration, defaultSpeed int, continuous bool) json.RawMessage {
	properties := map[string]any{
		"duration_ms": map[string]any{
			"type": "integer",
			"description": fmt.Sprintf("%s Clamped to %d..%d.",
				durationDesc, MinMovementMs, MaxMovementMs),
			"default": defaultDuration,
			"minimum": MinMovementMs,
			"maximum": MaxMovementMs,
		},
		"speed_percent": map[string]any{
			"type":        "integer",
			"description": fmt.Sprintf("How hard to move, 0..100. Defaults to %d if omitted.", defaultSpeed),
			"default":     defaultSpeed,
			"minimum":     0,
			"maximum":     100,
		},
	}
	if continuous {
		properties["continuous"] = map[string]any{
			"type": "boolean",
			"description": "Set true when the user asks the robot to keep going until told to stop " +
				"— 'keep walking', 'go until I say stop', 'keep driving'. It then moves until it " +
				"runs out of room or movement.stop arrives, and duration_ms is ignored. Leave unset for " +
				"a normal single move.",
			"default": false,
		}
	}

	b, _ := json.Marshal(map[string]any{
		"type":       "object",
		"properties": properties,
	})
	return b
}

func emptySchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{}}`)
}
