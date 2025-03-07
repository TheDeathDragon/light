package ledcontroller

import (
	"fmt"
	"io/ioutil"
	"sync"
	"time"
)

// LED paths
const (
	BlueLEDPath  = "/sys/class/leds/sc27xx:blue/brightness"
	GreenLEDPath = "/sys/class/leds/sc27xx:green/brightness"
	RedLEDPath   = "/sys/class/leds/sc27xx:red/brightness"
)

// Color represents RGB values
type Color struct {
	Red   int
	Green int
	Blue  int
}

var (
	// Predefined colors
	ColorRed   = Color{255, 0, 0}
	ColorGreen = Color{0, 255, 0}
	ColorBlue  = Color{0, 0, 255}
	ColorOff   = Color{0, 0, 0}

	// Control variables
	stopChan     chan bool
	effectActive bool
	mutex        sync.Mutex
)

// Initialize the LED controller
func init() {
	stopChan = make(chan bool, 1)
}

// StopCurrentEffect stops any ongoing light effect
func StopCurrentEffect() {
	mutex.Lock()
	defer mutex.Unlock()

	if effectActive {
		stopChan <- true
		// Wait a little to ensure the effect has time to process the stop signal
		time.Sleep(50 * time.Millisecond)
	}
}

// setColor sets the LED colors
func setColor(color Color) error {
	if err := writeBrightness(RedLEDPath, color.Red); err != nil {
		return fmt.Errorf("failed to set red LED: %v", err)
	}
	if err := writeBrightness(GreenLEDPath, color.Green); err != nil {
		return fmt.Errorf("failed to set green LED: %v", err)
	}
	if err := writeBrightness(BlueLEDPath, color.Blue); err != nil {
		return fmt.Errorf("failed to set blue LED: %v", err)
	}
	return nil
}

// writeBrightness writes the brightness value to the specified path
func writeBrightness(path string, brightness int) error {
	if brightness < 0 {
		brightness = 0
	}
	if brightness > 255 {
		brightness = 255
	}

	// Convert the brightness value to a string then to bytes
	brightnessStr := fmt.Sprintf("%d", brightness)
	return ioutil.WriteFile(path, []byte(brightnessStr), 0644)
}

// TurnOffLED turns off all LEDs
func TurnOffLED() error {
	StopCurrentEffect()
	return setColor(ColorOff)
}

// CallNotificationEffect implements the call notification effect:
// Red and blue flashing for 200ms, off for 200ms, alternating for 5 seconds
func CallNotificationEffect() error {
	return runTimedEffect(func(stop <-chan bool) {
		startTime := time.Now()
		for time.Since(startTime) < 5*time.Second {
			select {
			case <-stop:
				setColor(ColorOff)
				return
			default:
				// Red and blue mix
				setColor(Color{255, 0, 255})
				select {
				case <-stop:
					setColor(ColorOff)
					return
				case <-time.After(200 * time.Millisecond):
				}

				setColor(ColorOff)
				select {
				case <-stop:
					return
				case <-time.After(200 * time.Millisecond):
				}
			}
		}
		setColor(ColorOff)
	})
}

// NotificationEffect implements notification effect:
// Green breathing effect, each cycle 2s (1s brighten, 1s dim), repeat 5 times
func NotificationEffect() error {
	return runTimedEffect(func(stop <-chan bool) {
		cycleCount := 5
		for i := 0; i < cycleCount; i++ {
			// Brighten from 0 to 255
			for brightness := 0; brightness <= 255; brightness += 5 {
				select {
				case <-stop:
					setColor(ColorOff)
					return
				default:
					setColor(Color{0, brightness, 0})
					time.Sleep(time.Duration(1000/51) * time.Millisecond) // 1s / 51 steps
				}
			}

			// Dim from 255 to 0
			for brightness := 255; brightness >= 0; brightness -= 5 {
				select {
				case <-stop:
					setColor(ColorOff)
					return
				default:
					setColor(Color{0, brightness, 0})
					time.Sleep(time.Duration(1000/51) * time.Millisecond) // 1s / 51 steps
				}
			}
		}
		setColor(ColorOff)
	})
}

// MusicEffect implements music effect
func MusicEffect() error {
	return runTimedEffect(func(stop <-chan bool) {
		for {
			// 1st second
			startTime := time.Now()
			for time.Since(startTime) < 1*time.Second {
				select {
				case <-stop:
					setColor(ColorOff)
					return
				default:
					// Blue light flashes every 200ms
					setColor(ColorBlue)
					time.Sleep(200 * time.Millisecond)
					setColor(ColorOff)
					time.Sleep(200 * time.Millisecond)

					// Green light on for 500ms
					setColor(ColorGreen)
					time.Sleep(500 * time.Millisecond)
					setColor(ColorOff)
					time.Sleep(500 * time.Millisecond)

					// Blue to purple, then purple to red transition
					setColor(Color{128, 0, 128}) // Purple
					time.Sleep(250 * time.Millisecond)
					setColor(ColorRed)
					time.Sleep(250 * time.Millisecond)
				}
			}

			// 2nd to 4th second
			for i := 0; i < 3; i++ {
				startTime = time.Now()
				for time.Since(startTime) < 1*time.Second {
					select {
					case <-stop:
						setColor(ColorOff)
						return
					default:
						// Blue light flashes every 100ms
						setColor(ColorBlue)
						time.Sleep(100 * time.Millisecond)
						setColor(ColorOff)
						time.Sleep(100 * time.Millisecond)

						// Green light on for 500ms
						setColor(ColorGreen)
						time.Sleep(500 * time.Millisecond)
						setColor(ColorOff)
						time.Sleep(500 * time.Millisecond)

						// Blue and green transition to yellow, then back to green
						setColor(Color{255, 255, 0}) // Yellow
						time.Sleep(500 * time.Millisecond)
						setColor(ColorGreen)
						time.Sleep(500 * time.Millisecond)
					}
				}
			}

			// 5th second
			startTime = time.Now()
			for time.Since(startTime) < 1*time.Second {
				select {
				case <-stop:
					setColor(ColorOff)
					return
				default:
					// Red light on for 300ms
					setColor(ColorRed)
					time.Sleep(300 * time.Millisecond)
					setColor(ColorOff)
					time.Sleep(700 * time.Millisecond)

					// Blue light resumes flashing
					setColor(ColorBlue)
					time.Sleep(200 * time.Millisecond)
					setColor(ColorOff)
					time.Sleep(200 * time.Millisecond)
				}
			}

			// 6th to 9th second
			for i := 0; i < 4; i++ {
				startTime = time.Now()
				for time.Since(startTime) < 1*time.Second {
					select {
					case <-stop:
						setColor(ColorOff)
						return
					default:
						// Blue and green alternate flashing every 200ms
						setColor(ColorBlue)
						time.Sleep(200 * time.Millisecond)
						setColor(ColorOff)
						time.Sleep(200 * time.Millisecond)
						setColor(ColorGreen)
						time.Sleep(200 * time.Millisecond)
						setColor(ColorOff)
						time.Sleep(200 * time.Millisecond)

						// Red light on for 300ms
						setColor(ColorRed)
						time.Sleep(300 * time.Millisecond)
						setColor(ColorOff)
					}
				}
			}

			// 10th second: repeat the cycle
		}
	})
}

// BluetoothConnectingEffect implements Bluetooth connecting effect:
// Blue flashing (300ms on, 500ms off) for up to 5 seconds
func BluetoothConnectingEffect() error {
	return runTimedEffect(func(stop <-chan bool) {
		startTime := time.Now()
		for time.Since(startTime) < 5*time.Second {
			select {
			case <-stop:
				setColor(ColorOff)
				return
			default:
				setColor(ColorBlue)
				select {
				case <-stop:
					setColor(ColorOff)
					return
				case <-time.After(300 * time.Millisecond):
				}

				setColor(ColorOff)
				select {
				case <-stop:
					return
				case <-time.After(500 * time.Millisecond):
				}
			}
		}
		setColor(ColorOff)
	})
}

// BluetoothConnectedEffect implements Bluetooth connected effect:
// Solid blue for 3 seconds
func BluetoothConnectedEffect() error {
	return runTimedEffect(func(stop <-chan bool) {
		setColor(ColorBlue)
		select {
		case <-stop:
			setColor(ColorOff)
			return
		case <-time.After(3 * time.Second):
			setColor(ColorOff)
		}
	})
}

// BluetoothFailedEffect implements Bluetooth connection failed effect:
// Red flashing (200ms on, 400ms off) for 5 seconds
func BluetoothFailedEffect() error {
	return runTimedEffect(func(stop <-chan bool) {
		startTime := time.Now()
		for time.Since(startTime) < 5*time.Second {
			select {
			case <-stop:
				setColor(ColorOff)
				return
			default:
				setColor(ColorRed)
				select {
				case <-stop:
					setColor(ColorOff)
					return
				case <-time.After(200 * time.Millisecond):
				}

				setColor(ColorOff)
				select {
				case <-stop:
					return
				case <-time.After(400 * time.Millisecond):
				}
			}
		}
		setColor(ColorOff)
	})
}

// WiFiConnectingEffect implements WiFi connecting effect:
// Green breathing effect with 1s transitions for 5 seconds
func WiFiConnectingEffect() error {
	return runTimedEffect(func(stop <-chan bool) {
		startTime := time.Now()
		for time.Since(startTime) < 5*time.Second {
			// Brighten from 0 to 255
			for brightness := 0; brightness <= 255; brightness += 5 {
				select {
				case <-stop:
					setColor(ColorOff)
					return
				default:
					if time.Since(startTime) >= 5*time.Second {
						setColor(ColorOff)
						return
					}
					setColor(Color{0, brightness, 0})
					time.Sleep(time.Duration(1000/51) * time.Millisecond) // 1s / 51 steps
				}
			}

			// Dim from 255 to 0
			for brightness := 255; brightness >= 0; brightness -= 5 {
				select {
				case <-stop:
					setColor(ColorOff)
					return
				default:
					if time.Since(startTime) >= 5*time.Second {
						setColor(ColorOff)
						return
					}
					setColor(Color{0, brightness, 0})
					time.Sleep(time.Duration(1000/51) * time.Millisecond) // 1s / 51 steps
				}
			}
		}
		setColor(ColorOff)
	})
}

// WiFiConnectedEffect implements WiFi connected effect:
// Solid green for 3 seconds
func WiFiConnectedEffect() error {
	return runTimedEffect(func(stop <-chan bool) {
		setColor(ColorGreen)
		select {
		case <-stop:
			setColor(ColorOff)
			return
		case <-time.After(3 * time.Second):
			setColor(ColorOff)
		}
	})
}

// WiFiFailedEffect implements WiFi connection failed effect:
// Red flashing (300ms on, 300ms off) for 5 seconds
func WiFiFailedEffect() error {
	return runTimedEffect(func(stop <-chan bool) {
		startTime := time.Now()
		for time.Since(startTime) < 5*time.Second {
			select {
			case <-stop:
				setColor(ColorOff)
				return
			default:
				setColor(ColorRed)
				select {
				case <-stop:
					setColor(ColorOff)
					return
				case <-time.After(300 * time.Millisecond):
				}

				setColor(ColorOff)
				select {
				case <-stop:
					return
				case <-time.After(300 * time.Millisecond):
				}
			}
		}
		setColor(ColorOff)
	})
}

// PartyEffect implements party effect:
// Red, green, blue flashing in sequence for 100ms each, 10 seconds total
func PartyEffect() error {
	return runTimedEffect(func(stop <-chan bool) {
		startTime := time.Now()
		for time.Since(startTime) < 10*time.Second {
			// Red
			select {
			case <-stop:
				setColor(ColorOff)
				return
			default:
				setColor(ColorRed)
				select {
				case <-stop:
					setColor(ColorOff)
					return
				case <-time.After(100 * time.Millisecond):
				}
			}

			// Green
			select {
			case <-stop:
				setColor(ColorOff)
				return
			default:
				setColor(ColorGreen)
				select {
				case <-stop:
					setColor(ColorOff)
					return
				case <-time.After(100 * time.Millisecond):
				}
			}

			// Blue
			select {
			case <-stop:
				setColor(ColorOff)
				return
			default:
				setColor(ColorBlue)
				select {
				case <-stop:
					setColor(ColorOff)
					return
				case <-time.After(100 * time.Millisecond):
				}
			}
		}
		setColor(ColorOff)
	})
}

// ChargingLowBatteryEffect implements low battery charging effect:
// Red breathing (1s brighten, 1s dim), repeat 5 times
func ChargingLowBatteryEffect() error {
	return runTimedEffect(func(stop <-chan bool) {
		cycleCount := 5
		for i := 0; i < cycleCount; i++ {
			// Brighten from 0 to 255
			for brightness := 0; brightness <= 255; brightness += 5 {
				select {
				case <-stop:
					setColor(ColorOff)
					return
				default:
					setColor(Color{brightness, 0, 0})
					time.Sleep(time.Duration(1000/51) * time.Millisecond) // 1s / 51 steps
				}
			}

			// Dim from 255 to 0
			for brightness := 255; brightness >= 0; brightness -= 5 {
				select {
				case <-stop:
					setColor(ColorOff)
					return
				default:
					setColor(Color{brightness, 0, 0})
					time.Sleep(time.Duration(1000/51) * time.Millisecond) // 1s / 51 steps
				}
			}
		}
		setColor(ColorOff)
	})
}

// ChargingHighBatteryEffect implements high battery charging effect:
// Green breathing (1s brighten, 1s dim), repeat 5 times
func ChargingHighBatteryEffect() error {
	return runTimedEffect(func(stop <-chan bool) {
		cycleCount := 5
		for i := 0; i < cycleCount; i++ {
			// Brighten from 0 to 255
			for brightness := 0; brightness <= 255; brightness += 5 {
				select {
				case <-stop:
					setColor(ColorOff)
					return
				default:
					setColor(Color{0, brightness, 0})
					time.Sleep(time.Duration(1000/51) * time.Millisecond) // 1s / 51 steps
				}
			}

			// Dim from 255 to 0
			for brightness := 255; brightness >= 0; brightness -= 5 {
				select {
				case <-stop:
					setColor(ColorOff)
					return
				default:
					setColor(Color{0, brightness, 0})
					time.Sleep(time.Duration(1000/51) * time.Millisecond) // 1s / 51 steps
				}
			}
		}
		setColor(ColorOff)
	})
}

// ChargingCompleteEffect implements charging complete effect:
// Solid blue light
func ChargingCompleteEffect() error {
	StopCurrentEffect()
	return setColor(ColorBlue)
}

// CameraFocusEffect implements camera focus effect:
// Solid red for 2 seconds
func CameraFocusEffect() error {
	return runTimedEffect(func(stop <-chan bool) {
		setColor(ColorRed)
		select {
		case <-stop:
			setColor(ColorOff)
			return
		case <-time.After(2 * time.Second):
			setColor(ColorOff)
		}
	})
}

// CameraCaptureEffect implements camera capture effect:
// Flash blue for 1 second
func CameraCaptureEffect() error {
	return runTimedEffect(func(stop <-chan bool) {
		startTime := time.Now()
		for time.Since(startTime) < 1*time.Second {
			select {
			case <-stop:
				setColor(ColorOff)
				return
			default:
				setColor(ColorBlue)
				select {
				case <-stop:
					setColor(ColorOff)
					return
				case <-time.After(100 * time.Millisecond):
				}

				setColor(ColorOff)
				select {
				case <-stop:
					return
				case <-time.After(100 * time.Millisecond):
				}
			}
		}
		setColor(ColorOff)
	})
}

// CameraSavePhotoEffect implements camera save photo effect:
// Solid green for 2 seconds
func CameraSavePhotoEffect() error {
	return runTimedEffect(func(stop <-chan bool) {
		setColor(ColorGreen)
		select {
		case <-stop:
			setColor(ColorOff)
			return
		case <-time.After(2 * time.Second):
			setColor(ColorOff)
		}
	})
}

// BootupEffect implements boot-up effect:
// Red -> Green -> Blue sequence, each color for 1s with 0.2s interval
func BootupEffect() error {
	return runTimedEffect(func(stop <-chan bool) {
		// Red
		setColor(ColorRed)
		select {
		case <-stop:
			setColor(ColorOff)
			return
		case <-time.After(1 * time.Second):
		}

		// Interval
		setColor(ColorOff)
		select {
		case <-stop:
			return
		case <-time.After(200 * time.Millisecond):
		}

		// Green
		setColor(ColorGreen)
		select {
		case <-stop:
			setColor(ColorOff)
			return
		case <-time.After(1 * time.Second):
		}

		// Interval
		setColor(ColorOff)
		select {
		case <-stop:
			return
		case <-time.After(200 * time.Millisecond):
		}

		// Blue
		setColor(ColorBlue)
		select {
		case <-stop:
			setColor(ColorOff)
			return
		case <-time.After(1 * time.Second):
		}

		setColor(ColorOff)
	})
}

// runTimedEffect runs an effect in a goroutine with proper mutex locking
func runTimedEffect(effect func(<-chan bool)) error {
	mutex.Lock()

	// Stop any running effect
	if effectActive {
		stopChan <- true
		// Give the effect a little time to stop
		time.Sleep(50 * time.Millisecond)
	}

	// Clear the stop channel
	select {
	case <-stopChan:
	default:
	}

	effectActive = true
	mutex.Unlock()

	// Run the effect in a goroutine
	go func() {
		localStopChan := make(chan bool, 1)

		go func() {
			select {
			case <-stopChan:
				localStopChan <- true
			}
		}()

		effect(localStopChan)

		mutex.Lock()
		effectActive = false
		mutex.Unlock()
	}()

	return nil
}

// EnableLED turns on the LED with the specified color
func EnableLED(color Color) error {
	StopCurrentEffect()
	return setColor(color)
}

// SetRGB sets the RGB values directly
func SetRGB(red, green, blue int) error {
	StopCurrentEffect()
	return setColor(Color{red, green, blue})
}

// SetLEDBrightness sets the brightness of a specific LED color
func SetLEDBrightness(red, green, blue int) error {
	StopCurrentEffect()
	return setColor(Color{red, green, blue})
}

// IsEffectActive returns whether an effect is currently running
func IsEffectActive() bool {
	mutex.Lock()
	defer mutex.Unlock()
	return effectActive
}
