package tools

import (
	"encoding/json"
	"fmt"

	"github.com/streamcoreai/streamcore-server/internal/plugin"
)

// BotGestureTopic is the data-channel topic a rigged client listens on for
// arm and head poses. Separate from CarCommandTopic because a two-wheel
// drivetrain has neither: a device receiving these can ignore the topic
// wholesale rather than having to know which car.* actions it cannot do.
const BotGestureTopic = "bot.gesture"

// BotTool is a metadata-only Tool, same as CarTool — the pipeline
// intercepts "bot.*" names and emits a data-channel packet rather than
// ever reaching Execute.
type BotTool struct {
	name        string
	description string
	parameters  json.RawMessage
}

func (t *BotTool) Name() string                { return t.name }
func (t *BotTool) Description() string         { return t.description }
func (t *BotTool) Parameters() json.RawMessage { return t.parameters }
func (t *BotTool) ConfirmationRequired() bool  { return false }
func (t *BotTool) ThinkingSound() bool         { return false }

// Execute is never called in production — pipeline.go intercepts.
func (t *BotTool) Execute(params json.RawMessage) (string, error) {
	return "", fmt.Errorf("bot.* tool calls must be intercepted by the pipeline")
}

// Gestures returns the arm and head tools to register with the plugin
// manager. Left and right are the bot's own, matching car.turn_left.
func Gestures() []plugin.Tool {
	hold := gestureSchema("How long to hold the pose in milliseconds (clamped to 200..10000).", 2500)
	beat := gestureSchema("How long to keep going in milliseconds (clamped to 200..10000).", 1800)
	// A sweep is slow: cut it short and the head stops halfway through, which
	// reads as the bot changing its mind rather than looking around.
	sweep := gestureSchema("How long to keep looking around in milliseconds (clamped to 200..10000).", 4500)

	return []plugin.Tool{
		&BotTool{"bot.wave", "Wave hello or goodbye with one hand. Use when the user greets the robot, says hi or bye, waves at it, or asks it to wave.", beat},
		&BotTool{"bot.raise_arms", "Throw BOTH arms up in the air at once. Only when the user means both — 'hands up', 'both arms up', 'cheer', 'celebrate'. If they name one hand, use the single-arm tool instead.", hold},
		&BotTool{"bot.raise_left_arm", "Raise only the arm on the robot's own LEFT side. Use for 'put your left hand up', 'raise your left arm', 'lift your left hand'.", hold},
		&BotTool{"bot.raise_right_arm", "Raise only the arm on the robot's own RIGHT side. Use for 'put your right hand up', 'raise your right arm', 'lift your right hand'.", hold},
		&BotTool{"bot.point_left", "Point with the arm on the robot's own left side. Use when the user asks it to point left, or at something on its left.", hold},
		&BotTool{"bot.point_right", "Point with the arm on the robot's own right side. Use when the user asks it to point right, or at something on its right.", hold},
		&BotTool{"bot.look_left", "Turn the head to the robot's own left without moving its feet. Use for 'look left', 'look over there'.", hold},
		&BotTool{"bot.look_right", "Turn the head to the robot's own right without moving its feet. Use for 'look right'.", hold},
		&BotTool{"bot.look_up", "Tilt the head up. Use for 'look up', 'look at the ceiling'.", hold},
		&BotTool{"bot.look_down", "Tilt the head down. Use for 'look down', 'look at the floor'.", hold},
		&BotTool{"bot.look_around", "Sweep the head from side to side, as if taking in the room. Use for 'look around', 'spin your head', 'have a look about', 'what can you see', 'scan the room'.", sweep},
		&BotTool{"bot.nod", "Nod the head yes. Use when agreeing, confirming, or when the user asks the robot to nod.", beat},
		&BotTool{"bot.shake_head", "Shake the head no. Use when disagreeing, declining, or when the user asks the robot to shake its head.", beat},
		&BotTool{"bot.rest", "Drop the arms and face front again. Use when the user asks it to relax, put its arms down, stop pointing, or look forward.", emptySchema()},
	}
}

func gestureSchema(durationDesc string, defaultDuration int) json.RawMessage {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"duration_ms": map[string]any{
				"type":        "integer",
				"description": durationDesc,
				"default":     defaultDuration,
				"minimum":     200,
				"maximum":     10000,
			},
		},
	}
	b, _ := json.Marshal(schema)
	return b
}
