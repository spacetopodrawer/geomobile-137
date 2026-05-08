package neogeo

import (
	"time"

	"cadastreia/pkg/arcade"
)

// Controller handles NEO-GEO controller input mapping
// Implements arcade.ControllerMapping interface
type Controller struct {
	inputBuffer    chan arcade.JoystickButton
	gameInputChan  chan *arcade.GameInput
	currentState   arcade.JoystickButton
	previousState  arcade.JoystickButton
	mappings       map[arcade.JoystickButton]string
	deadZone       float32
	pollRate       time.Duration
	calibrationMap map[arcade.JoystickButton]arcade.JoystickButton
}

// NewController creates a new NEO-GEO controller mapper
func NewController() *Controller {
	cm := &Controller{
		inputBuffer:    make(chan arcade.JoystickButton, 100),
		gameInputChan:  make(chan *arcade.GameInput, 100),
		mappings:       getDefaultMappings(),
		deadZone:       0.15,
		pollRate:       16 * time.Millisecond,
		calibrationMap: make(map[arcade.JoystickButton]arcade.JoystickButton),
	}

	go cm.pollInputs()

	return cm
}

// PushInput sends raw joystick button state to mapper
func (cm *Controller) PushInput(buttons arcade.JoystickButton) error {
	select {
	case cm.inputBuffer <- buttons:
		return nil
	default:
		// Buffer full, drop oldest
		select {
		case <-cm.inputBuffer:
			cm.inputBuffer <- buttons
		default:
		}
		return nil
	}
}

// GetGameInput retrieves the next processed game input
func (cm *Controller) GetGameInput() *arcade.GameInput {
	select {
	case input := <-cm.gameInputChan:
		return input
	case <-time.After(1 * time.Millisecond):
		return nil
	}
}

// MapButton maps a physical button to a logical action
func (cm *Controller) MapButton(physical string, logical string) error {
	// Map physical button name to logical action
	// Example: "button_a" -> "punch"
	return nil
}

// GetButtonMap returns the current button mapping
func (cm *Controller) GetButtonMap() map[string]string {
	buttonMap := make(map[string]string)
	buttonMap["button_a"] = "punch"
	buttonMap["button_b"] = "kick"
	buttonMap["button_c"] = "special"
	buttonMap["button_d"] = "guard"
	return buttonMap
}

// GetSupportedDirections returns number of supported directions
func (cm *Controller) GetSupportedDirections() int {
	return 8 // 8-directional support
}

// GetActionButtons returns list of action button names
func (cm *Controller) GetActionButtons() []string {
	return []string{"button_a", "button_b", "button_c", "button_d"}
}

// CalibrateAxis calibrates joystick axis
func (cm *Controller) CalibrateAxis(axisName string, minValue, maxValue uint16) error {
	// Joystick calibration (placeholder)
	return nil
}

// pollInputs polls the input buffer at NEO-GEO refresh rate
func (cm *Controller) pollInputs() {
	ticker := time.NewTicker(cm.pollRate)
	defer ticker.Stop()

	for range ticker.C {
		select {
		case buttons := <-cm.inputBuffer:
			input := cm.processInput(buttons)
			select {
			case cm.gameInputChan <- input:
			default:
				// Channel full, drop
			}
		default:
		}
	}
}

// processInput converts raw joystick buttons to game input
func (cm *Controller) processInput(buttons arcade.JoystickButton) *arcade.GameInput {
	input := &arcade.GameInput{
		Timestamp:      time.Now(),
		ControllerType: "neogeo",
		Direction:      "idle",
		Action:         "none",
		Button:         "",
	}

	// Direction processing with diagonal support
	dirX := float32(0)
	dirY := float32(0)

	if buttons&arcade.JoystickUp != 0 {
		dirY = -1
		input.Direction = "up"
	}
	if buttons&arcade.JoystickDown != 0 {
		dirY = 1
		if input.Direction == "up" {
			input.Direction = "idle"
		} else {
			input.Direction = "down"
		}
	}

	if buttons&arcade.JoystickLeft != 0 {
		dirX = -1
		if input.Direction != "idle" {
			input.Direction = "up-left"
		} else {
			input.Direction = "left"
		}
	}
	if buttons&arcade.JoystickRight != 0 {
		dirX = 1
		if input.Direction != "idle" {
			if input.Direction == "left" {
				input.Direction = "idle"
			} else {
				input.Direction = "up-right"
			}
		} else {
			input.Direction = "right"
		}
	}

	// Normalize diagonal movement
	if dirX != 0 && dirY != 0 {
		length := float32(1.414) // sqrt(2)
		dirX /= length
		dirY /= length
	}

	input.Movement = [2]float32{dirX, dirY}

	// Action button processing
	if buttons&arcade.ButtonA != 0 {
		input.Action = "punch"
		input.Button = "button_a"
	} else if buttons&arcade.ButtonB != 0 {
		input.Action = "kick"
		input.Button = "button_b"
	} else if buttons&arcade.ButtonC != 0 {
		input.Action = "special"
		input.Button = "button_c"
	} else if buttons&arcade.ButtonD != 0 {
		input.Action = "guard"
		input.Button = "button_d"
	}

	// Extended controls
	if buttons&arcade.Coin != 0 {
		input.Action = "coin"
	} else if buttons&arcade.Start != 0 {
		input.Action = "start"
	}

	return input
}

// getDefaultMappings returns the default button mappings
func getDefaultMappings() map[arcade.JoystickButton]string {
	return map[arcade.JoystickButton]string{
		arcade.JoystickUp:    "move_up",
		arcade.JoystickDown:  "move_down",
		arcade.JoystickLeft:  "move_left",
		arcade.JoystickRight: "move_right",
		arcade.ButtonA:       "punch",
		arcade.ButtonB:       "kick",
		arcade.ButtonC:       "special",
		arcade.ButtonD:       "guard",
		arcade.Coin:          "coin",
		arcade.Start:         "start",
	}
}
