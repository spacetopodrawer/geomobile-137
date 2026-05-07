package arcade

import (
	"fmt"
	"time"
)

// JoystickButton represents a physical button on arcade controller
type JoystickButton uint16

const (
	// Direction buttons
	JoystickUp    JoystickButton = 0x01
	JoystickDown  JoystickButton = 0x02
	JoystickLeft  JoystickButton = 0x04
	JoystickRight JoystickButton = 0x08

	// Action buttons
	ButtonA JoystickButton = 0x10 // Button A (punch)
	ButtonB JoystickButton = 0x20 // Button B (kick)
	ButtonC JoystickButton = 0x40 // Button C (special)
	ButtonD JoystickButton = 0x80 // Button D (guard)

	// Extra controls
	Coin  JoystickButton = 0x100
	Start JoystickButton = 0x200
)

// GameInput represents the result of controller input mapping
type GameInput struct {
	Direction      string    // "up", "down", "left", "right", "idle"
	Movement       [2]float32 // {dx, dy}
	Action         string     // "punch", "kick", "special", "guard", "scan"
	Button         string     // Which button pressed
	Timestamp      time.Time
	DeviceID       string
	ControllerType string
}

// ControlMapper maps physical joystick inputs to game commands
type ControlMapper struct {
	inputBuffer    chan JoystickButton
	gameInputChan  chan *GameInput
	currentState   JoystickButton
	previousState  JoystickButton
	mappings       map[JoystickButton]string
	deadZone       float32
	pollRate       time.Duration
	emulatorType   string // "neoragex5", "mame", "fbneo"
	calibrationMap map[JoystickButton]JoystickButton
}

// NewControlMapper creates a new controller mapper
func NewControlMapper(emulatorType string) *ControlMapper {
	cm := &ControlMapper{
		inputBuffer:   make(chan JoystickButton, 100),
		gameInputChan: make(chan *GameInput, 100),
		mappings:      getDefaultMappings(),
		deadZone:      0.15,
		pollRate:      16 * time.Millisecond, // ~62.5 Hz (NEO-GEO refresh rate)
		emulatorType:  emulatorType,
		calibrationMap: make(map[JoystickButton]JoystickButton),
	}

	// Start polling routine
	go cm.pollInputs()

	return cm
}

// PushInput sends a raw joystick button state to the mapper
func (cm *ControlMapper) PushInput(buttons JoystickButton) {
	select {
	case cm.inputBuffer <- buttons:
	default:
		// Buffer full, drop oldest input
		select {
		case <-cm.inputBuffer:
			cm.inputBuffer <- buttons
		default:
		}
	}
}

// pollInputs processes raw input into game commands
func (cm *ControlMapper) pollInputs() {
	ticker := time.NewTicker(cm.pollRate)
	defer ticker.Stop()

	for range ticker.C {
		select {
		case rawInput := <-cm.inputBuffer:
			cm.processInput(rawInput)
		default:
			// No input, check for idle state
			if cm.currentState != 0 {
				cm.currentState = 0
				cm.generateIdleInput()
			}
		}
	}
}

// processInput converts raw button states to game input
func (cm *ControlMapper) processInput(buttons JoystickButton) {
	cm.previousState = cm.currentState
	cm.currentState = buttons

	// Detect button press (not held)
	newPresses := buttons & ^cm.previousState

	input := &GameInput{
		Timestamp:      time.Now(),
		ControllerType: cm.emulatorType,
	}

	// Process directional input
	dirBits := buttons & 0x0F
	input.Direction, input.Movement = cm.mapDirection(dirBits)

	// Process action buttons
	if newPresses&ButtonA != 0 {
		input.Action = "punch"
		input.Button = "A"
	} else if newPresses&ButtonB != 0 {
		input.Action = "kick"
		input.Button = "B"
	} else if newPresses&ButtonC != 0 {
		input.Action = "special"
		input.Button = "C"
	} else if newPresses&ButtonD != 0 {
		input.Action = "guard"
		input.Button = "D"
	} else if newPresses&0x0F != 0 {
		// Direction buttons alone without action
		input.Action = "move"
	}

	// Handle coin/start (for arcade)
	if newPresses&Coin != 0 {
		input.Button = "COIN"
	}
	if newPresses&Start != 0 {
		input.Button = "START"
	}

	cm.gameInputChan <- input
}

// mapDirection converts directional bits to movement vector
func (cm *ControlMapper) mapDirection(dirBits JoystickButton) (string, [2]float32) {
	movement := [2]float32{0, 0}
	direction := "idle"
	speed := float32(2.0)

	// Diagonal support: both X and Y movement allowed
	if dirBits&JoystickUp != 0 {
		movement[1] -= speed
		direction = "up"
	}
	if dirBits&JoystickDown != 0 {
		movement[1] += speed
		direction = "down"
	}
	if dirBits&JoystickLeft != 0 {
		movement[0] -= speed
		direction = "left"
	}
	if dirBits&JoystickRight != 0 {
		movement[0] += speed
		direction = "right"
	}

	// Handle diagonals
	if movement[0] != 0 && movement[1] != 0 {
		// Normalize diagonal (Sqrt(2) = 1.414...)
		magnitude := float32(1.414)
		movement[0] /= magnitude
		movement[1] /= magnitude
		direction = cm.describeDiagonal(dirBits)
	}

	return direction, movement
}

// describeDiagonal returns readable name for diagonal direction
func (cm *ControlMapper) describeDiagonal(dirBits JoystickButton) string {
	if dirBits&JoystickUp != 0 && dirBits&JoystickLeft != 0 {
		return "up-left"
	}
	if dirBits&JoystickUp != 0 && dirBits&JoystickRight != 0 {
		return "up-right"
	}
	if dirBits&JoystickDown != 0 && dirBits&JoystickLeft != 0 {
		return "down-left"
	}
	if dirBits&JoystickDown != 0 && dirBits&JoystickRight != 0 {
		return "down-right"
	}
	return "idle"
}

// generateIdleInput sends idle state when no input is pressed
func (cm *ControlMapper) generateIdleInput() {
	input := &GameInput{
		Direction: "idle",
		Movement:  [2]float32{0, 0},
		Action:    "idle",
		Timestamp: time.Now(),
	}
	cm.gameInputChan <- input
}

// GetGameInput receives the next game input from the queue
func (cm *ControlMapper) GetGameInput() *GameInput {
	select {
	case input := <-cm.gameInputChan:
		return input
	default:
		return nil
	}
}

// CalibrateAxis calibrates a joystick axis for deadzone compensation
func (cm *ControlMapper) CalibrateAxis(axis string, minValue, maxValue float32) {
	// Store calibration values for later axis processing
	// This is for analog sticks (future enhancement)
	fmt.Printf("Calibrated %s: min=%.2f, max=%.2f, deadzone=%.2f\n", axis, minValue, maxValue, cm.deadZone)
}

// SetEmulatorType changes the emulator target
func (cm *ControlMapper) SetEmulatorType(emulatorType string) {
	cm.emulatorType = emulatorType
	fmt.Printf("Control mapper switched to: %s\n", emulatorType)
}

// GetCurrentState returns the current button state
func (cm *ControlMapper) GetCurrentState() JoystickButton {
	return cm.currentState
}

// IsButtonPressed checks if a specific button is currently pressed
func (cm *ControlMapper) IsButtonPressed(button JoystickButton) bool {
	return (cm.currentState & button) != 0
}

// ClearInputBuffer empties the input queue (for emergencies)
func (cm *ControlMapper) ClearInputBuffer() {
	for {
		select {
		case <-cm.inputBuffer:
		default:
			return
		}
	}
}

// getDefaultMappings returns the standard NEO-GEO button mappings
func getDefaultMappings() map[JoystickButton]string {
	return map[JoystickButton]string{
		JoystickUp:    "up",
		JoystickDown:  "down",
		JoystickLeft:  "left",
		JoystickRight: "right",
		ButtonA:       "punch",
		ButtonB:       "kick",
		ButtonC:       "special",
		ButtonD:       "guard",
		Coin:          "coin",
		Start:         "start",
	}
}

// NEOGEOControllerSimulator simulates NEO-GEO controller input for testing
type NEOGEOControllerSimulator struct {
	mapper *ControlMapper
}

// NewNEOGEOControllerSimulator creates a test controller
func NewNEOGEOControllerSimulator(mapper *ControlMapper) *NEOGEOControllerSimulator {
	return &NEOGEOControllerSimulator{mapper: mapper}
}

// SimulateMoveUp simulates pressing up direction
func (ncs *NEOGEOControllerSimulator) SimulateMoveUp() {
	ncs.mapper.PushInput(JoystickUp)
}

// SimulateMoveDown simulates pressing down direction
func (ncs *NEOGEOControllerSimulator) SimulateMoveDown() {
	ncs.mapper.PushInput(JoystickDown)
}

// SimulateMoveLeft simulates pressing left direction
func (ncs *NEOGEOControllerSimulator) SimulateMoveLeft() {
	ncs.mapper.PushInput(JoystickLeft)
}

// SimulateMoveRight simulates pressing right direction
func (ncs *NEOGEOControllerSimulator) SimulateMoveRight() {
	ncs.mapper.PushInput(JoystickRight)
}

// SimulatePunch simulates button A (punch)
func (ncs *NEOGEOControllerSimulator) SimulatePunch() {
	ncs.mapper.PushInput(ButtonA)
}

// SimulateKick simulates button B (kick)
func (ncs *NEOGEOControllerSimulator) SimulateKick() {
	ncs.mapper.PushInput(ButtonB)
}

// SimulateSpecial simulates button C (special)
func (ncs *NEOGEOControllerSimulator) SimulateSpecial() {
	ncs.mapper.PushInput(ButtonC)
}

// SimulateGuard simulates button D (guard)
func (ncs *NEOGEOControllerSimulator) SimulateGuard() {
	ncs.mapper.PushInput(ButtonD)
}

// SimulateStart simulates start button
func (ncs *NEOGEOControllerSimulator) SimulateStart() {
	ncs.mapper.PushInput(Start)
}

// SimulateCoin simulates coin insertion
func (ncs *NEOGEOControllerSimulator) SimulateCoin() {
	ncs.mapper.PushInput(Coin)
}

// ReleaseAll simulates releasing all buttons
func (ncs *NEOGEOControllerSimulator) ReleaseAll() {
	ncs.mapper.PushInput(0) // All bits zero = no buttons pressed
}

// NeoRageX5Protocol defines the protocol for NeoRageX5 emulator communication
type NeoRageX5Protocol struct {
	version      string
	commandPort  int
	dataPort     int
	authenticated bool
}

// NewNeoRageX5Protocol creates a protocol handler for NeoRageX5
func NewNeoRageX5Protocol() *NeoRageX5Protocol {
	return &NeoRageX5Protocol{
		version:     "1.0",
		commandPort: 6001,
		dataPort:    6002,
	}
}

// EncodeInputFrame encodes a game input as NEO-GEO machine code
func (nrp *NeoRageX5Protocol) EncodeInputFrame(input *GameInput) []byte {
	frame := make([]byte, 4)

	// Byte 0: Direction bits
	dirBits := byte(0)
	if input.Direction == "up" || input.Direction == "up-left" || input.Direction == "up-right" {
		dirBits |= 0x01
	}
	if input.Direction == "down" || input.Direction == "down-left" || input.Direction == "down-right" {
		dirBits |= 0x02
	}
	if input.Direction == "left" || input.Direction == "up-left" || input.Direction == "down-left" {
		dirBits |= 0x04
	}
	if input.Direction == "right" || input.Direction == "up-right" || input.Direction == "down-right" {
		dirBits |= 0x08
	}
	frame[0] = dirBits

	// Byte 1: Action buttons
	buttonBits := byte(0)
	if input.Action == "punch" || input.Button == "A" {
		buttonBits |= 0x10
	}
	if input.Action == "kick" || input.Button == "B" {
		buttonBits |= 0x20
	}
	if input.Action == "special" || input.Button == "C" {
		buttonBits |= 0x40
	}
	if input.Action == "guard" || input.Button == "D" {
		buttonBits |= 0x80
	}
	frame[1] = buttonBits

	// Byte 2: Timestamp (frame counter, modulo 256)
	frame[2] = byte(input.Timestamp.UnixMilli() % 256)

	// Byte 3: Device ID / Controller type
	frame[3] = 0x01 // Player 1 by default

	return frame
}

// DecodeInputFrame decodes NEO-GEO input frame back to GameInput
func (nrp *NeoRageX5Protocol) DecodeInputFrame(frame []byte) *GameInput {
	if len(frame) < 4 {
		return nil
	}

	input := &GameInput{
		Timestamp: time.Now(),
	}

	// Decode direction
	dirBits := frame[0]
	input.Direction, input.Movement = mapDirectionBits(dirBits)

	// Decode buttons
	buttonBits := frame[1]
	if (buttonBits & 0x10) != 0 {
		input.Action = "punch"
		input.Button = "A"
	} else if (buttonBits & 0x20) != 0 {
		input.Action = "kick"
		input.Button = "B"
	} else if (buttonBits & 0x40) != 0 {
		input.Action = "special"
		input.Button = "C"
	} else if (buttonBits & 0x80) != 0 {
		input.Action = "guard"
		input.Button = "D"
	}

	return input
}

// mapDirectionBits is a helper function to decode direction bits
func mapDirectionBits(dirBits byte) (string, [2]float32) {
	movement := [2]float32{0, 0}
	direction := "idle"
	speed := float32(2.0)

	if (dirBits & 0x01) != 0 {
		movement[1] -= speed
		direction = "up"
	}
	if (dirBits & 0x02) != 0 {
		movement[1] += speed
		direction = "down"
	}
	if (dirBits & 0x04) != 0 {
		movement[0] -= speed
		direction = "left"
	}
	if (dirBits & 0x08) != 0 {
		movement[0] += speed
		direction = "right"
	}

	return direction, movement
}
