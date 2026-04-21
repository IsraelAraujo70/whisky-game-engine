//go:build darwin

package macos

import (
	"sync"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"

	"github.com/IsraelAraujo70/whisky-game-engine/input"
)

var (
	gcOnce      sync.Once
	gcErr       error
	gcLoaded    bool

	classGCController       objc.Class
	classGCExtendedGamepad  objc.Class
	classGCControllerAxisInput objc.Class
	classGCControllerButtonInput objc.Class
	classGCControllerDirectionPad objc.Class

	selControllers          objc.SEL
	selExtendedGamepad      objc.SEL
	selIsPressed            objc.SEL
	selValue                objc.SEL
	selXAxis                objc.SEL
	selYAxis                objc.SEL
	selButtonA              objc.SEL
	selButtonB              objc.SEL
	selButtonX              objc.SEL
	selButtonY              objc.SEL
	selLeftShoulder         objc.SEL
	selRightShoulder        objc.SEL
	selLeftTrigger          objc.SEL
	selRightTrigger         objc.SEL
	selLeftThumbstick       objc.SEL
	selRightThumbstick      objc.SEL
	selButtonMenu           objc.SEL
	selButtonOptions        objc.SEL
	selButtonHome           objc.SEL
	selLeftThumbstickButton objc.SEL
	selRightThumbstickButton objc.SEL
	selDPad                 objc.SEL
	selCount                objc.SEL
	selObjectAtIndex        objc.SEL
)

func initGCController() {
	gcOnce.Do(func() {
		if _, err := purego.Dlopen("/System/Library/Frameworks/GameController.framework/GameController", purego.RTLD_GLOBAL|purego.RTLD_LAZY); err != nil {
			gcErr = err
			return
		}
		classGCController = objc.GetClass("GCController")
		classGCExtendedGamepad = objc.GetClass("GCExtendedGamepad")
		classGCControllerAxisInput = objc.GetClass("GCControllerAxisInput")
		classGCControllerButtonInput = objc.GetClass("GCControllerButtonInput")
		classGCControllerDirectionPad = objc.GetClass("GCControllerDirectionPad")

		selControllers = objc.RegisterName("controllers")
		selExtendedGamepad = objc.RegisterName("extendedGamepad")
		selIsPressed = objc.RegisterName("isPressed")
		selValue = objc.RegisterName("value")
		selXAxis = objc.RegisterName("xAxis")
		selYAxis = objc.RegisterName("yAxis")
		selButtonA = objc.RegisterName("buttonA")
		selButtonB = objc.RegisterName("buttonB")
		selButtonX = objc.RegisterName("buttonX")
		selButtonY = objc.RegisterName("buttonY")
		selLeftShoulder = objc.RegisterName("leftShoulder")
		selRightShoulder = objc.RegisterName("rightShoulder")
		selLeftTrigger = objc.RegisterName("leftTrigger")
		selRightTrigger = objc.RegisterName("rightTrigger")
		selLeftThumbstick = objc.RegisterName("leftThumbstick")
		selRightThumbstick = objc.RegisterName("rightThumbstick")
		selButtonMenu = objc.RegisterName("buttonMenu")
		selButtonOptions = objc.RegisterName("buttonOptions")
		selButtonHome = objc.RegisterName("buttonHome")
		selLeftThumbstickButton = objc.RegisterName("leftThumbstickButton")
		selRightThumbstickButton = objc.RegisterName("rightThumbstickButton")
		selDPad = objc.RegisterName("dpad")
		selCount = objc.RegisterName("count")
		selObjectAtIndex = objc.RegisterName("objectAtIndex:")
		gcLoaded = true
	})
}

func pollGCControllers(state *input.State) {
	initGCController()
	if !gcLoaded {
		return
	}

	controllers := objc.ID(classGCController).Send(selControllers)
	count := int(objc.Send[uintptr](controllers, selCount))
	if count > input.MaxGamepads {
		count = input.MaxGamepads
	}

	for i := 0; i < count; i++ {
		pad := state.Gamepad(i)
		controller := objc.Send[objc.ID](controllers, selObjectAtIndex, uintptr(i))
		gamepad := objc.Send[objc.ID](controller, selExtendedGamepad)
		if gamepad == 0 {
			if pad.Connected() {
				pad.SetConnected(false)
			}
			continue
		}
		if !pad.Connected() {
			pad.SetConnected(true)
		}

		pad.SetButton(input.GamepadButtonA, buttonPressed(gamepad, selButtonA))
		pad.SetButton(input.GamepadButtonB, buttonPressed(gamepad, selButtonB))
		pad.SetButton(input.GamepadButtonX, buttonPressed(gamepad, selButtonX))
		pad.SetButton(input.GamepadButtonY, buttonPressed(gamepad, selButtonY))
		pad.SetButton(input.GamepadButtonLB, buttonPressed(gamepad, selLeftShoulder))
		pad.SetButton(input.GamepadButtonRB, buttonPressed(gamepad, selRightShoulder))
		pad.SetButton(input.GamepadButtonBack, buttonPressed(gamepad, selButtonOptions))
		pad.SetButton(input.GamepadButtonStart, buttonPressed(gamepad, selButtonMenu))
		pad.SetButton(input.GamepadButtonGuide, buttonPressed(gamepad, selButtonHome))
		pad.SetButton(input.GamepadButtonLeftStick, buttonPressed(gamepad, selLeftThumbstickButton))
		pad.SetButton(input.GamepadButtonRightStick, buttonPressed(gamepad, selRightThumbstickButton))

		// D-Pad
		dpad := objc.Send[objc.ID](gamepad, selDPad)
		pad.SetButton(input.GamepadButtonDPadUp, dpadPressed(dpad, selYAxis, -1))
		pad.SetButton(input.GamepadButtonDPadDown, dpadPressed(dpad, selYAxis, 1))
		pad.SetButton(input.GamepadButtonDPadLeft, dpadPressed(dpad, selXAxis, -1))
		pad.SetButton(input.GamepadButtonDPadRight, dpadPressed(dpad, selXAxis, 1))

		// Axes
		pad.SetAxis(input.GamepadAxisLX, axisValue(gamepad, selLeftThumbstick, selXAxis))
		pad.SetAxis(input.GamepadAxisLY, -axisValue(gamepad, selLeftThumbstick, selYAxis))
		pad.SetAxis(input.GamepadAxisRX, axisValue(gamepad, selRightThumbstick, selXAxis))
		pad.SetAxis(input.GamepadAxisRY, -axisValue(gamepad, selRightThumbstick, selYAxis))
		pad.SetAxis(input.GamepadAxisLT, triggerValue(gamepad, selLeftTrigger))
		pad.SetAxis(input.GamepadAxisRT, triggerValue(gamepad, selRightTrigger))
	}

	// Mark disconnected slots beyond current count
	for i := count; i < input.MaxGamepads; i++ {
		pad := state.Gamepad(i)
		if pad.Connected() {
			pad.SetConnected(false)
		}
	}
}

func buttonPressed(gamepad objc.ID, sel objc.SEL) bool {
	btn := objc.Send[objc.ID](gamepad, sel)
	if btn == 0 {
		return false
	}
	return objc.Send[bool](btn, selIsPressed)
}

func axisValue(gamepad objc.ID, stickSel, axisSel objc.SEL) float64 {
	stick := objc.Send[objc.ID](gamepad, stickSel)
	if stick == 0 {
		return 0
	}
	axis := objc.Send[objc.ID](stick, axisSel)
	if axis == 0 {
		return 0
	}
	return objc.Send[float64](axis, selValue)
}

func triggerValue(gamepad objc.ID, triggerSel objc.SEL) float64 {
	trigger := objc.Send[objc.ID](gamepad, triggerSel)
	if trigger == 0 {
		return 0
	}
	return objc.Send[float64](trigger, selValue)
}

func dpadPressed(dpad objc.ID, axisSel objc.SEL, threshold float64) bool {
	if dpad == 0 {
		return false
	}
	axis := objc.Send[objc.ID](dpad, axisSel)
	if axis == 0 {
		return false
	}
	v := objc.Send[float64](axis, selValue)
	if threshold > 0 {
		return v > 0.5
	}
	return v < -0.5
}
