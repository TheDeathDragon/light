package ledcontroller

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

// LED paths
const (
	BlueLEDPath  = "/sys/devices/platform/11016000.i2c5/i2c-5/5-0034/mt6370_pmu_rgbled/leds/mt6370_pmu_led3/brightness"
	GreenLEDPath = "/sys/devices/platform/11016000.i2c5/i2c-5/5-0034/mt6370_pmu_rgbled/leds/mt6370_pmu_led2/brightness"
	RedLEDPath   = "/sys/devices/platform/11016000.i2c5/i2c-5/5-0034/mt6370_pmu_rgbled/leds/mt6370_pmu_led1/brightness"
)

// Brightness constants
const (
	MinBrightness = 0
	MaxBrightness = 6
)

// Effect types
const (
	EFFECT_NONE                 = 0
	EFFECT_BOOTUP               = 1
	EFFECT_NOTIFICATION         = 2
	EFFECT_CALL                 = 3
	EFFECT_CHARGING_LOW         = 4
	EFFECT_CHARGING_HIGH        = 5
	EFFECT_CHARGING_COMPLETE    = 6
	EFFECT_WIFI_CONNECTING      = 7
	EFFECT_WIFI_CONNECTED       = 8
	EFFECT_WIFI_FAILED          = 9
	EFFECT_BLUETOOTH_CONNECTING = 10
	EFFECT_BLUETOOTH_CONNECTED  = 11
	EFFECT_BLUETOOTH_FAILED     = 12
	EFFECT_CAMERA_FOCUS         = 13
	EFFECT_CAMERA_CAPTURE       = 14
	EFFECT_CAMERA_SAVE          = 15
	EFFECT_PARTY                = 16
	EFFECT_MUSIC                = 17
)

// Color represents RGB values
type Color struct {
	Red   int
	Green int
	Blue  int
}

var (
	// Predefined colors (0-6 brightness range)
	ColorRed   = Color{6, 0, 0}
	ColorGreen = Color{0, 6, 0}
	ColorBlue  = Color{0, 0, 6}
	ColorOff   = Color{0, 0, 0}

	// Control variables
	effectActive      bool
	currentEffectType int
	ledEnabled        bool = true
	mutex             sync.Mutex

	// Context for effect control
	currentCancel context.CancelFunc
	effectDone    chan struct{}

	// File handles for LED control (kept open for performance)
	redFile    *os.File
	greenFile  *os.File
	blueFile   *os.File
	filesMutex sync.Mutex

	// Debug mode control
	DebugMode bool = false
)

// Initialize the LED controller
func init() {
	// Open LED files for performance
	var err error
	redFile, err = os.OpenFile(RedLEDPath, os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Warning: Failed to open red LED file: %v", err)
	}
	greenFile, err = os.OpenFile(GreenLEDPath, os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Warning: Failed to open green LED file: %v", err)
	}
	blueFile, err = os.OpenFile(BlueLEDPath, os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Warning: Failed to open blue LED file: %v", err)
	}
}

// debugLog logs only when debug mode is enabled
func debugLog(format string, args ...interface{}) {
	if DebugMode {
		log.Printf(format, args...)
	}
}

// clamp restricts value to [min, max] range
func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// StopCurrentEffect stops any ongoing light effect
func StopCurrentEffect() {
	mutex.Lock()
	cancel := currentCancel
	done := effectDone
	active := effectActive
	mutex.Unlock()

	if !active {
		return
	}

	debugLog("StopCurrentEffect: Stopping current effect")

	// Cancel the effect context
	if cancel != nil {
		cancel()
	}

	// Wait for effect to complete (with timeout)
	if done != nil {
		select {
		case <-done:
			debugLog("StopCurrentEffect: Effect stopped successfully")
		case <-time.After(1 * time.Second):
			log.Println("StopCurrentEffect: Timeout waiting for effect to stop")
		}
	}
}

// writeLEDValue writes a value to the specified LED file handle
func writeLEDValue(file *os.File, value int) error {
	if file == nil {
		return fmt.Errorf("LED file not opened")
	}

	filesMutex.Lock()
	defer filesMutex.Unlock()

	// Seek to beginning of file
	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("seek error: %w", err)
	}

	// Write value
	valueStr := strconv.Itoa(value) + "\n"
	if _, err := file.WriteString(valueStr); err != nil {
		return fmt.Errorf("write error: %w", err)
	}

	// Truncate file to new length
	if err := file.Truncate(int64(len(valueStr))); err != nil {
		return fmt.Errorf("truncate error: %w", err)
	}

	return nil
}

// setRed sets the red LED value (0-6 range)
func setRed(value int) error {
	mutex.Lock()
	enabled := ledEnabled
	mutex.Unlock()

	if !enabled {
		return nil
	}

	value = clamp(value, MinBrightness, MaxBrightness)
	return writeLEDValue(redFile, value)
}

// setGreen sets the green LED value (0-6 range)
func setGreen(value int) error {
	mutex.Lock()
	enabled := ledEnabled
	mutex.Unlock()

	if !enabled {
		return nil
	}

	value = clamp(value, MinBrightness, MaxBrightness)
	return writeLEDValue(greenFile, value)
}

// setBlue sets the blue LED value (0-6 range)
func setBlue(value int) error {
	mutex.Lock()
	enabled := ledEnabled
	mutex.Unlock()

	if !enabled {
		return nil
	}

	value = clamp(value, MinBrightness, MaxBrightness)
	return writeLEDValue(blueFile, value)
}

// setColor sets the LED colors (0-6 range)
func setColor(color Color) error {
	if !IsLEDEnabled() {
		return nil
	}

	// Clamp values to valid range
	color.Red = clamp(color.Red, MinBrightness, MaxBrightness)
	color.Green = clamp(color.Green, MinBrightness, MaxBrightness)
	color.Blue = clamp(color.Blue, MinBrightness, MaxBrightness)

	// Write color values
	var errs []error
	if err := setRed(color.Red); err != nil {
		errs = append(errs, fmt.Errorf("red LED: %w", err))
	}
	if err := setGreen(color.Green); err != nil {
		errs = append(errs, fmt.Errorf("green LED: %w", err))
	}
	if err := setBlue(color.Blue); err != nil {
		errs = append(errs, fmt.Errorf("blue LED: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("setColor failed: %v", errs)
	}
	return nil
}

// TurnOffLED turns off all LEDs
func TurnOffLED() error {
	StopCurrentEffect()
	return setColor(ColorOff)
}

// FadeColor implements a blinking effect between two colors (name kept for compatibility)
func FadeColor(from, to Color, duration time.Duration, ctx context.Context) error {
	debugLog("FadeColor: Starting blink from %v to %v, duration %v", from, to, duration)

	blinkInterval := 100 * time.Millisecond
	ticker := time.NewTicker(blinkInterval)
	defer ticker.Stop()

	deadline := time.Now().Add(duration)
	showingFrom := true

	for time.Now().Before(deadline) {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			debugLog("FadeColor: Context cancelled")
			setColor(ColorOff)
			return ctx.Err()
		case <-ticker.C:
			// Blink between colors
			if showingFrom {
				if err := setColor(from); err != nil {
					log.Printf("FadeColor: Error setting color: %v", err)
				}
			} else {
				if err := setColor(to); err != nil {
					log.Printf("FadeColor: Error setting color: %v", err)
				}
			}
			showingFrom = !showingFrom
		}
	}

	debugLog("FadeColor: Blink complete")
	return nil
}

// PulseColor implements a blinking effect for a specific color (name kept for compatibility)
// If pulseCount is 0, it will continue indefinitely until stopped
func PulseColor(color Color, pulseCount int, pulseDuration time.Duration, ctx context.Context) error {
	debugLog("PulseColor: Starting blink effect, color %v, count %d, duration %v", color, pulseCount, pulseDuration)
	halfDuration := pulseDuration / 2

	for i := 0; pulseCount == 0 || i < pulseCount; i++ {
		debugLog("PulseColor: Pulse %d", i+1)

		// Turn on
		setColor(color)
		select {
		case <-ctx.Done():
			debugLog("PulseColor: Context cancelled during ON phase")
			setColor(ColorOff)
			return ctx.Err()
		case <-time.After(halfDuration):
		}

		// Turn off
		setColor(ColorOff)
		select {
		case <-ctx.Done():
			debugLog("PulseColor: Context cancelled during OFF phase")
			return ctx.Err()
		case <-time.After(halfDuration):
		}
	}

	debugLog("PulseColor: Blink complete")
	return nil
}

// BlinkColor implements a blinking effect for a specific color
func BlinkColor(color Color, blinkCount int, onDuration, offDuration time.Duration, ctx context.Context) error {
	debugLog("BlinkColor: Starting blink, color %v, count %d, on %v, off %v", color, blinkCount, onDuration, offDuration)

	for i := 0; blinkCount == 0 || i < blinkCount; i++ {
		debugLog("BlinkColor: Blink %d", i+1)

		// Turn on
		setColor(color)
		select {
		case <-ctx.Done():
			debugLog("BlinkColor: Context cancelled during ON phase")
			setColor(ColorOff)
			return ctx.Err()
		case <-time.After(onDuration):
		}

		// Turn off
		setColor(ColorOff)
		select {
		case <-ctx.Done():
			debugLog("BlinkColor: Context cancelled during OFF phase")
			return ctx.Err()
		case <-time.After(offDuration):
		}
	}

	debugLog("BlinkColor: Blink complete")
	return nil
}

// CallNotificationEffect implements the call notification effect:
// Red and blue alternating flashing (200ms on, 200ms off) until stopped
func CallNotificationEffect() error {
	return runTimedEffect(func(ctx context.Context) {
		isRed := true

		for {
			// 设置当前颜色（红色或蓝色）
			if isRed {
				setColor(ColorRed)
			} else {
				setColor(ColorBlue)
			}

			// 亮200ms
			select {
			case <-ctx.Done():
				setColor(ColorOff)
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 灭200ms
			setColor(ColorOff)
			select {
			case <-ctx.Done():
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 切换颜色
			isRed = !isRed
		}
	}, EFFECT_CALL)
}

// NotificationEffect implements notification effect:
// Green breathing effect, each cycle 2s (1s brighten, 1s dim), continuously until stopped
func NotificationEffect() error {
	return runTimedEffect(func(ctx context.Context) {
		debugLog("NotificationEffect: Starting notification effect")
		err := PulseColor(ColorGreen, 0, 2*time.Second, ctx)
		if err != nil {
			debugLog("NotificationEffect: Error in PulseColor: %v", err)
		}
	}, EFFECT_NOTIFICATION)
}

// MusicEffect implements music effect
func MusicEffect() error {
	return runTimedEffect(func(ctx context.Context) {
		for {
			// 第一秒
			// 0-0.2S 常亮蓝灯，0.4-0.6S，常亮蓝灯，0.8-1.0S，常亮蓝灯
			// 0-0.5S，常亮绿灯

			// 0-0.2S 常亮蓝灯，常亮绿灯
			setColor(Color{0, 6, 6})
			select {
			case <-ctx.Done():
				setColor(ColorOff)
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 0.2-0.4S 蓝灯灭
			setColor(Color{0, 6, 0})
			select {
			case <-ctx.Done():
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 0.4-0.5S 常亮蓝灯和绿灯
			setColor(Color{0, 6, 6})
			select {
			case <-ctx.Done():
				setColor(ColorOff)
				return
			case <-time.After(100 * time.Millisecond):
			}

			// 0.5-0.6S 绿灯灭
			setColor(Color{0, 0, 6})
			select {
			case <-ctx.Done():
				setColor(ColorOff)
				return
			case <-time.After(100 * time.Millisecond):
			}

			// 0.6-0.8S 蓝灯灭
			setColor(Color{0, 0, 0})
			select {
			case <-ctx.Done():
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 0.8-1.0S 常亮蓝灯
			setColor(Color{0, 0, 6})
			select {
			case <-ctx.Done():
				setColor(ColorOff)
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 0-0.5S 常亮绿灯（与蓝灯同时进行，需要单独控制绿色通道）
			// 由于时间已经过去1秒，这里不需要再执行绿灯效果

			// 第二秒
			// 1.0-1.5S，常亮蓝灯，闪烁红灯，1.5-2.0S，常亮红灯，闪烁蓝灯

			// 1.0-1.5S 蓝灯常亮，红灯在0和6之间闪烁
			blinkStart := time.Now()
			showRed := false
			for time.Since(blinkStart) < 500*time.Millisecond {
				select {
				case <-ctx.Done():
					setColor(ColorOff)
					return
				default:
					if showRed {
						setColor(Color{6, 0, 6}) // 红蓝都亮
					} else {
						setColor(Color{0, 0, 6}) // 只有蓝灯亮
					}
					showRed = !showRed
					time.Sleep(70 * time.Millisecond)
				}
			}

			// 1.5-2.0S 红灯常亮，蓝灯在6和0之间闪烁
			blinkStart = time.Now()
			showBlue := true
			for time.Since(blinkStart) < 500*time.Millisecond {
				select {
				case <-ctx.Done():
					setColor(ColorOff)
					return
				default:
					if showBlue {
						setColor(Color{6, 0, 6}) // 红蓝都亮
					} else {
						setColor(Color{6, 0, 0}) // 只有红灯亮
					}
					showBlue = !showBlue
					time.Sleep(70 * time.Millisecond)
				}
			}

			// 第三秒
			// 2-2.2S 常亮蓝灯，2.4-2.6S，常亮蓝灯，2.8-3.0S，常亮蓝灯
			// 2.0-2.5S，常亮绿灯

			// 2.0-2.2S 常亮蓝灯
			setColor(Color{0, 0, 6})
			select {
			case <-ctx.Done():
				setColor(ColorOff)
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 2.2-2.4S 蓝灯灭
			setColor(Color{0, 0, 0})
			select {
			case <-ctx.Done():
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 2.4-2.6S 常亮蓝灯
			setColor(Color{0, 0, 6})
			select {
			case <-ctx.Done():
				setColor(ColorOff)
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 2.6-2.8S 蓝灯灭
			setColor(Color{0, 0, 0})
			select {
			case <-ctx.Done():
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 2.8-3.0S 常亮蓝灯
			setColor(Color{0, 0, 6})
			select {
			case <-ctx.Done():
				setColor(ColorOff)
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 2.0-2.5S 常亮绿灯（与蓝灯同时进行，需要单独控制绿色通道）
			// 由于时间已经过去1秒，这里不需要再执行绿灯效果

			// 第四秒
			// 3.0-3.5S，闪烁红绿灯，3.5S-4.0S，绿灯保持常亮，闪烁红灯

			// 3.0-3.5S 红灯和绿灯在0和6之间同时闪烁
			blinkStart = time.Now()
			showBoth := false
			for time.Since(blinkStart) < 500*time.Millisecond {
				select {
				case <-ctx.Done():
					setColor(ColorOff)
					return
				default:
					if showBoth {
						setColor(Color{6, 6, 0}) // 红绿都亮
					} else {
						setColor(Color{0, 0, 0}) // 都灭
					}
					showBoth = !showBoth
					time.Sleep(70 * time.Millisecond)
				}
			}

			// 3.5S-4.0S 绿灯常亮，红灯在6和0之间闪烁
			blinkStart = time.Now()
			showRed = true
			for time.Since(blinkStart) < 500*time.Millisecond {
				select {
				case <-ctx.Done():
					setColor(ColorOff)
					return
				default:
					if showRed {
						setColor(Color{6, 6, 0}) // 红绿都亮
					} else {
						setColor(Color{0, 6, 0}) // 只有绿灯亮
					}
					showRed = !showRed
					time.Sleep(70 * time.Millisecond)
				}
			}

			// 第五秒
			// 4.0-4.3S，常亮红灯，4.4-4.6S，常亮蓝灯，4.8-4.9S，常亮蓝灯

			// 4.0-4.3S 常亮红灯
			setColor(Color{6, 0, 0})
			select {
			case <-ctx.Done():
				setColor(ColorOff)
				return
			case <-time.After(300 * time.Millisecond):
			}

			// 4.3-4.4S 灯灭
			setColor(Color{0, 0, 0})
			select {
			case <-ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
			}

			// 4.4-4.6S 常亮蓝灯
			setColor(Color{0, 0, 6})
			select {
			case <-ctx.Done():
				setColor(ColorOff)
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 4.6-4.8S 灯灭
			setColor(Color{0, 0, 0})
			select {
			case <-ctx.Done():
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 4.8-4.9S 常亮蓝灯
			setColor(Color{0, 0, 6})
			select {
			case <-ctx.Done():
				setColor(ColorOff)
				return
			case <-time.After(100 * time.Millisecond):
			}

			// 4.9-5.0S 灯灭
			setColor(Color{0, 0, 0})
			select {
			case <-ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
			}

			// 第六秒
			// 5.0-5.2S，常亮蓝灯，5.2-5.4S，常亮绿灯，5.4-5.6S，常亮蓝灯，5.6-5.8S，常亮绿灯，5.8-6.0S，常亮蓝灯

			// 5.0-5.2S 常亮蓝灯
			setColor(Color{0, 0, 6})
			select {
			case <-ctx.Done():
				setColor(ColorOff)
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 5.2-5.4S 常亮绿灯
			setColor(Color{0, 6, 0})
			select {
			case <-ctx.Done():
				setColor(ColorOff)
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 5.4-5.6S 常亮蓝灯
			setColor(Color{0, 0, 6})
			select {
			case <-ctx.Done():
				setColor(ColorOff)
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 5.6-5.8S 常亮绿灯
			setColor(Color{0, 6, 0})
			select {
			case <-ctx.Done():
				setColor(ColorOff)
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 5.8-6.0S 常亮蓝灯
			setColor(Color{0, 0, 6})
			select {
			case <-ctx.Done():
				setColor(ColorOff)
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 第七秒
			// 6.0-6.5S，蓝灯在6和2之间闪烁，6.5-7.0S，蓝灯在2和6之间闪烁

			// 6.0-6.5S 蓝灯在6和2之间闪烁
			blinkStart = time.Now()
			showHigh := true
			for time.Since(blinkStart) < 500*time.Millisecond {
				select {
				case <-ctx.Done():
					setColor(ColorOff)
					return
				default:
					if showHigh {
						setColor(Color{0, 0, 6})
					} else {
						setColor(Color{0, 0, 2})
					}
					showHigh = !showHigh
					time.Sleep(100 * time.Millisecond)
				}
			}

			// 6.5-7.0S 蓝灯在2和6之间闪烁
			blinkStart = time.Now()
			showHigh = false
			for time.Since(blinkStart) < 500*time.Millisecond {
				select {
				case <-ctx.Done():
					setColor(ColorOff)
					return
				default:
					if showHigh {
						setColor(Color{0, 0, 6})
					} else {
						setColor(Color{0, 0, 2})
					}
					showHigh = !showHigh
					time.Sleep(100 * time.Millisecond)
				}
			}

			// 第八秒
			// 7.0-7.5S，绿灯在2和6之间闪烁，7.5-8.0S，绿灯在6和2之间闪烁

			// 7.0-7.5S 绿灯在2和6之间闪烁
			blinkStart = time.Now()
			showHigh = false
			for time.Since(blinkStart) < 500*time.Millisecond {
				select {
				case <-ctx.Done():
					setColor(ColorOff)
					return
				default:
					if showHigh {
						setColor(Color{0, 6, 0})
					} else {
						setColor(Color{0, 2, 0})
					}
					showHigh = !showHigh
					time.Sleep(100 * time.Millisecond)
				}
			}

			// 7.5-8.0S 绿灯在6和2之间闪烁
			blinkStart = time.Now()
			showHigh = true
			for time.Since(blinkStart) < 500*time.Millisecond {
				select {
				case <-ctx.Done():
					setColor(ColorOff)
					return
				default:
					if showHigh {
						setColor(Color{0, 6, 0})
					} else {
						setColor(Color{0, 2, 0})
					}
					showHigh = !showHigh
					time.Sleep(100 * time.Millisecond)
				}
			}

			// 第九秒
			// 8.0-8.7S，红灯在2和6之间闪烁，8.7-9.0S，常亮红灯

			// 8.0-8.7S 红灯在2和6之间闪烁
			blinkStart = time.Now()
			showHigh = false
			for time.Since(blinkStart) < 700*time.Millisecond {
				select {
				case <-ctx.Done():
					setColor(ColorOff)
					return
				default:
					if showHigh {
						setColor(Color{6, 0, 0})
					} else {
						setColor(Color{2, 0, 0})
					}
					showHigh = !showHigh
					time.Sleep(140 * time.Millisecond)
				}
			}

			// 8.7-9.0S 常亮红灯
			setColor(Color{6, 0, 0})
			select {
			case <-ctx.Done():
				setColor(ColorOff)
				return
			case <-time.After(300 * time.Millisecond):
			}

			// 第十秒
			// 9.0-9.5S，常亮绿灯，9.5-10.0S，常亮蓝灯

			// 9.0-9.5S 常亮绿灯
			setColor(Color{0, 6, 0})
			select {
			case <-ctx.Done():
				setColor(ColorOff)
				return
			case <-time.After(500 * time.Millisecond):
			}

			// 9.5-10.0S 常亮蓝灯
			setColor(Color{0, 0, 6})
			select {
			case <-ctx.Done():
				setColor(ColorOff)
				return
			case <-time.After(500 * time.Millisecond):
			}

			// 循环结束，重新开始
		}
	}, EFFECT_MUSIC)
}

// BluetoothConnectingEffect implements Bluetooth connecting effect:
// Blue flashing (300ms on, 500ms off)
func BluetoothConnectingEffect() error {
	return runTimedEffect(func(ctx context.Context) {
		BlinkColor(ColorBlue, 0, 300*time.Millisecond, 500*time.Millisecond, ctx)
	}, EFFECT_BLUETOOTH_CONNECTING)
}

// BluetoothConnectedEffect implements Bluetooth connected effect:
// Solid blue for 3 seconds
func BluetoothConnectedEffect() error {
	return runTimedEffect(func(ctx context.Context) {
		setColor(ColorBlue)
		select {
		case <-ctx.Done():
			return
		case <-time.After(3 * time.Second):
			return
		}
	}, EFFECT_BLUETOOTH_CONNECTED)
}

// BluetoothFailedEffect implements Bluetooth connection failed effect:
// Red flashing (200ms on, 400ms off) for 3 times
func BluetoothFailedEffect() error {
	return runTimedEffect(func(ctx context.Context) {
		BlinkColor(ColorRed, 3, 200*time.Millisecond, 400*time.Millisecond, ctx)
	}, EFFECT_BLUETOOTH_FAILED)
}

// WiFiConnectingEffect implements WiFi connecting effect:
// Green breathing effect with 1s transitions
func WiFiConnectingEffect() error {
	return runTimedEffect(func(ctx context.Context) {
		log.Println("WiFiConnectingEffect: 开始WiFi连接效果")
		err := PulseColor(ColorGreen, 0, 2*time.Second+500*time.Millisecond, ctx)
		if err != nil {
			log.Printf("WiFiConnectingEffect: 执行PulseColor时出错: %v", err)
		}

		log.Println("WiFiConnectingEffect: PulseColor返回，确保LED关闭")
	}, EFFECT_WIFI_CONNECTING)
}

// WiFiConnectedEffect implements WiFi connected effect:
// Solid green for 3 seconds
func WiFiConnectedEffect() error {
	return runTimedEffect(func(ctx context.Context) {
		setColor(ColorGreen)
		select {
		case <-ctx.Done():
			setColor(ColorOff)
			return
		case <-time.After(3 * time.Second):
			setColor(ColorOff)
			return // 显式返回，确保goroutine结束
		}
	}, EFFECT_WIFI_CONNECTED)
}

// WiFiFailedEffect implements WiFi connection failed effect:
// Red flashing (300ms on, 300ms off) for 3 times
func WiFiFailedEffect() error {
	return runTimedEffect(func(ctx context.Context) {
		BlinkColor(ColorRed, 3, 300*time.Millisecond, 300*time.Millisecond, ctx)
	}, EFFECT_WIFI_FAILED)
}

// PartyEffect implements a complex light show with different patterns over 9 seconds
// Now loops continuously until stopped
func PartyEffect() error {
	return runTimedEffect(func(ctx context.Context) {
		for { // 添加无限循环
			startTime := time.Now()
			totalDuration := 9 * time.Second

			// 用于控制红灯的计时器
			redLightTimer := time.NewTimer(3 * time.Second)
			defer redLightTimer.Stop()

			// 主循环，持续9秒
			for time.Since(startTime) < totalDuration {
				currentTime := time.Since(startTime)
				currentSecond := int(currentTime.Seconds()) + 1 // 从第1秒开始

				// 检查是否需要停止
				select {
				case <-ctx.Done():
					setColor(ColorOff)
					return
				default:
					// 继续执行
				}

				// 根据当前时间执行不同的灯光效果
				switch {
				case currentSecond == 1: // 第1秒
					// 处理红灯（每隔3秒亮起300ms）
					select {
					case <-redLightTimer.C:
						// 红灯亮起
						setRed(6)
						time.Sleep(300 * time.Millisecond)
						setRed(0)
						redLightTimer.Reset(3 * time.Second)
					default:
						// 不做任何事
					}

					// 处理蓝灯（每50ms闪烁一次，亮200ms，熄灭50ms，亮度波动）
					blueIntensity := 4 + int(2*float64(time.Now().UnixNano()%100)/100.0) // 亮度波动4-6
					setBlue(blueIntensity)
					time.Sleep(200 * time.Millisecond)
					setBlue(0)
					time.Sleep(50 * time.Millisecond)

					// 处理绿灯（每100ms闪烁一次，亮200ms，熄灭100ms，亮度渐变）
					greenProgress := float64(currentTime.Milliseconds()%1000) / 1000.0 // 0-1之间的渐变进度
					greenIntensity := int(2 + 4*greenProgress)                         // 亮度从2到6渐变
					setGreen(greenIntensity)
					time.Sleep(200 * time.Millisecond)
					setGreen(0)
					time.Sleep(100 * time.Millisecond)

				case currentSecond >= 2 && currentSecond <= 4: // 第2-4秒
					// 处理红灯（每隔3秒亮起300ms）
					select {
					case <-redLightTimer.C:
						// 红灯亮起
						setRed(6)
						time.Sleep(300 * time.Millisecond)
						setRed(0)
						redLightTimer.Reset(3 * time.Second)
					default:
						// 不做任何事
					}

					// 处理蓝灯（每50ms闪烁一次，亮200ms，熄灭50ms，亮度波动）
					blueIntensity := 4 + int(2*float64(time.Now().UnixNano()%100)/100.0) // 亮度波动4-6
					setBlue(blueIntensity)
					time.Sleep(200 * time.Millisecond)
					setBlue(0)
					time.Sleep(50 * time.Millisecond)

					// 处理绿灯（每100ms闪烁一次，亮200ms，熄灭100ms，亮度波动）
					greenIntensity := 4 + int(2*float64(time.Now().UnixNano()%100)/100.0) // 亮度波动4-6
					setGreen(greenIntensity)
					time.Sleep(200 * time.Millisecond)
					setGreen(0)
					time.Sleep(100 * time.Millisecond)

				case currentSecond == 5: // 第5秒
					// 多彩闪烁：蓝色、紫色、绿色、黄色快速切换，整个过程持续500ms
					transitionStart := time.Now()
					colorIndex := 0
					colors := []Color{
						{0, 0, 6}, // 蓝色
						{6, 0, 6}, // 紫色
						{0, 6, 0}, // 绿色
						{6, 6, 0}, // 黄色
					}
					for time.Since(transitionStart) < 500*time.Millisecond {
						setColor(colors[colorIndex])
						colorIndex = (colorIndex + 1) % len(colors)

						// 检查是否需要停止
						select {
						case <-ctx.Done():
							setColor(ColorOff)
							return
						default:
							time.Sleep(50 * time.Millisecond) // 快速切换颜色
						}
					}

					// 继续处理蓝灯和绿灯的闪烁
					for i := 0; i < 3; i++ { // 执行几次闪烁循环
						// 蓝灯：每50ms闪烁一次，亮200ms，熄灭50ms
						setBlue(5)
						time.Sleep(200 * time.Millisecond)
						setBlue(0)
						time.Sleep(50 * time.Millisecond)

						// 绿灯：每100ms闪烁一次，亮200ms，熄灭100ms
						setGreen(5)
						time.Sleep(200 * time.Millisecond)
						setGreen(0)
						time.Sleep(100 * time.Millisecond)

						// 红灯点缀
						if i == 1 {
							setRed(6)
							time.Sleep(100 * time.Millisecond)
							setRed(0)
						}
					}

				case currentSecond >= 6 && currentSecond <= 8: // 第6-8秒
					// 蓝灯：每50ms闪烁一次，亮200ms，熄灭50ms
					setBlue(5)
					time.Sleep(200 * time.Millisecond)
					setBlue(0)
					time.Sleep(50 * time.Millisecond)

					// 绿灯：每100ms闪烁一次，亮200ms，熄灭100ms
					setGreen(5)
					time.Sleep(200 * time.Millisecond)
					setGreen(0)
					time.Sleep(100 * time.Millisecond)

				case currentSecond == 9: // 第9秒
					// 蓝、绿灯交替闪烁，亮度波动
					for i := 0; i < 5; i++ { // 执行几次交替闪烁
						// 蓝灯闪烁
						blueIntensity := 4 + int(2*float64(time.Now().UnixNano()%100)/100.0) // 亮度波动4-6
						setBlue(blueIntensity)
						time.Sleep(100 * time.Millisecond)
						setBlue(0)

						// 绿灯闪烁
						greenIntensity := 4 + int(2*float64(time.Now().UnixNano()%100)/100.0) // 亮度波动4-6
						setGreen(greenIntensity)
						time.Sleep(150 * time.Millisecond)
						setGreen(0)

						// 检查是否需要停止
						select {
						case <-ctx.Done():
							setColor(ColorOff)
							return
						default:
							// 继续执行
						}
					}
				}

				// 检查是否需要停止
				select {
				case <-ctx.Done():
					setColor(ColorOff)
					return
				default:
					// 继续执行，短暂休眠以避免CPU过度使用
					time.Sleep(10 * time.Millisecond)
				}
			}

			// 检查是否需要停止，在开始下一个循环前
			select {
			case <-ctx.Done():
				setColor(ColorOff)
				return
			default:
				// 继续执行下一个循环
			}
		}
	}, EFFECT_PARTY)
}

// ChargingLowBatteryEffect implements low battery charging effect:
// Red breathing (1s brighten, 1s dim), continuously until stopped
func ChargingLowBatteryEffect() error {
	return runTimedEffect(func(ctx context.Context) {
		log.Println("ChargingLowBatteryEffect: 开始执行")
		err := PulseColor(ColorRed, 0, 2*time.Second, ctx)
		if err != nil {
			log.Printf("ChargingLowBatteryEffect: 执行PulseColor时出错: %v", err)
		}

		log.Println("ChargingLowBatteryEffect: PulseColor返回，确保LED关闭")
	}, EFFECT_CHARGING_LOW)
}

// ChargingHighBatteryEffect implements high battery charging effect:
// Green breathing (1s brighten, 1s dim), continuously until stopped
func ChargingHighBatteryEffect() error {
	return runTimedEffect(func(ctx context.Context) {
		log.Println("ChargingHighBatteryEffect: 开始执行")
		err := PulseColor(ColorGreen, 0, 2*time.Second, ctx)
		if err != nil {
			log.Printf("ChargingHighBatteryEffect: 执行PulseColor时出错: %v", err)
		}

		log.Println("ChargingHighBatteryEffect: PulseColor返回，确保LED关闭")
	}, EFFECT_CHARGING_HIGH)
}

// ChargingCompleteEffect implements charging complete effect:
// Solid blue light
func ChargingCompleteEffect() error {
	return runTimedEffect(func(ctx context.Context) {
		setColor(ColorBlue)
		// 使用无限循环定期检查停止信号，避免永久阻塞
		for {
			select {
			case <-ctx.Done():
				setColor(ColorOff)
				return
			case <-time.After(100 * time.Millisecond):
				// 定期检查，不做任何事
			}
		}
	}, EFFECT_CHARGING_COMPLETE)
}

// CameraFocusEffect implements camera focus effect:
// Solid orange for 2 seconds (R6 G3 B0)
func CameraFocusEffect() error {
	return runTimedEffect(func(ctx context.Context) {
		// 橙色 R6 G3 B0
		setColor(Color{6, 3, 0})
		select {
		case <-ctx.Done():
			setColor(ColorOff)
			return
		case <-time.After(2 * time.Second):
			setColor(ColorOff)
			return // 显式返回，确保goroutine结束
		}
	}, EFFECT_CAMERA_FOCUS)
}

// CameraCaptureEffect implements camera capture effect:
// Solid white for 1 second, then off for 0.5 second, then solid white for 0.2 second
func CameraCaptureEffect() error {
	return runTimedEffect(func(ctx context.Context) {
		// 白色常亮1秒
		setColor(Color{6, 6, 6})
		select {
		case <-ctx.Done():
			setColor(ColorOff)
			return
		case <-time.After(1 * time.Second):
		}

		// 熄灭0.5秒
		setColor(ColorOff)
		select {
		case <-ctx.Done():
			setColor(ColorOff) // 为了一致性，显式关闭LED
			return
		case <-time.After(500 * time.Millisecond):
		}

		// 白色常亮0.2秒
		setColor(Color{6, 6, 6})
		select {
		case <-ctx.Done():
			setColor(ColorOff)
			return
		case <-time.After(200 * time.Millisecond):
		}

		// 关闭LED并返回，确保goroutine结束
	}, EFFECT_CAMERA_CAPTURE)
}

// CameraSavePhotoEffect implements camera save photo effect:
// Solid green for 2 seconds
func CameraSavePhotoEffect() error {
	return runTimedEffect(func(ctx context.Context) {
		setColor(ColorGreen)
		select {
		case <-ctx.Done():
			setColor(ColorOff)
			return
		case <-time.After(1 * time.Second):
			setColor(ColorOff)
			return // 显式返回，确保goroutine结束
		}
	}, EFFECT_CAMERA_SAVE)
}

// BootupEffect implements boot-up effect:
// Complex sequence with smooth transitions and solid colors
func BootupEffect() error {
	return runTimedEffect(func(ctx context.Context) {
		log.Println("BootupEffect: 开始执行启动灯效")

		// 第一至二秒: 闪烁效果
		// 0-0.5S 绿灯在0和4之间闪烁、蓝灯在6和4之间闪烁
		startTime := time.Now()
		duration := 500 * time.Millisecond
		showLow := true
		for time.Since(startTime) < duration {
			if showLow {
				setColor(Color{0, 0, 6})
			} else {
				setColor(Color{0, 4, 4})
			}
			showLow = !showLow

			select {
			case <-ctx.Done():
				log.Println("BootupEffect: 在第一阶段收到停止信号")
				setColor(ColorOff)
				return
			case <-time.After(50 * time.Millisecond):
			}
		}

		// 0.5-1S 绿灯在4和6之间闪烁、蓝灯在4和6之间闪烁
		startTime = time.Now()
		duration = 500 * time.Millisecond
		showLow = true
		for time.Since(startTime) < duration {
			if showLow {
				setColor(Color{0, 4, 4})
			} else {
				setColor(Color{0, 6, 6})
			}
			showLow = !showLow

			select {
			case <-ctx.Done():
				log.Println("BootupEffect: 在第二阶段收到停止信号")
				setColor(ColorOff)
				return
			case <-time.After(50 * time.Millisecond):
			}
		}

		// 1S-1.5S 绿灯在6和2之间闪烁、红灯在0和2之间闪烁
		startTime = time.Now()
		duration = 500 * time.Millisecond
		showLow = true
		for time.Since(startTime) < duration {
			if showLow {
				setColor(Color{0, 6, 0})
			} else {
				setColor(Color{2, 2, 0})
			}
			showLow = !showLow

			select {
			case <-ctx.Done():
				log.Println("BootupEffect: 在第三阶段收到停止信号")
				setColor(ColorOff)
				return
			case <-time.After(50 * time.Millisecond):
			}
		}

		// 1.5S-2S 绿灯在2和6之间闪烁，红灯在2和6之间闪烁
		startTime = time.Now()
		duration = 500 * time.Millisecond
		showLow = true
		for time.Since(startTime) < duration {
			if showLow {
				setColor(Color{2, 2, 0})
			} else {
				setColor(Color{6, 6, 0})
			}
			showLow = !showLow

			select {
			case <-ctx.Done():
				log.Println("BootupEffect: 在第四阶段收到停止信号")
				setColor(ColorOff)
				return
			case <-time.After(50 * time.Millisecond):
			}
		}

		// 第三至四秒: 交替常亮
		// 2S-2.4S 常亮绿灯
		setColor(ColorGreen)
		select {
		case <-ctx.Done():
			log.Println("BootupEffect: 在2-2.4S阶段收到停止信号")
			setColor(ColorOff)
			return
		case <-time.After(400 * time.Millisecond):
		}

		// 2.4-2.8S 常亮蓝灯
		setColor(ColorBlue)
		select {
		case <-ctx.Done():
			log.Println("BootupEffect: 在2.4-2.8S阶段收到停止信号")
			setColor(ColorOff)
			return
		case <-time.After(400 * time.Millisecond):
		}

		// 2.8S-3.2S 常亮绿灯
		setColor(ColorGreen)
		select {
		case <-ctx.Done():
			log.Println("BootupEffect: 在2.8-3.2S阶段收到停止信号")
			setColor(ColorOff)
			return
		case <-time.After(400 * time.Millisecond):
		}

		// 3.2S-3.6S 常亮蓝灯
		setColor(ColorBlue)
		select {
		case <-ctx.Done():
			log.Println("BootupEffect: 在3.2-3.6S阶段收到停止信号")
			setColor(ColorOff)
			return
		case <-time.After(400 * time.Millisecond):
		}

		// 3.6S-4S 常亮绿灯
		setColor(ColorGreen)
		select {
		case <-ctx.Done():
			log.Println("BootupEffect: 在3.6-4S阶段收到停止信号")
			setColor(ColorOff)
			return
		case <-time.After(400 * time.Millisecond):
		}

		// 第四至六秒: 混合常亮和闪烁
		// 4-4.5S 常亮橙色（红6，绿2）
		setColor(Color{6, 2, 0})
		select {
		case <-ctx.Done():
			log.Println("BootupEffect: 在4-4.5S阶段收到停止信号")
			setColor(ColorOff)
			return
		case <-time.After(500 * time.Millisecond):
		}

		// 4.5-5S 蓝灯在2和6之间闪烁
		startTime = time.Now()
		duration = 500 * time.Millisecond
		showLow = true
		for time.Since(startTime) < duration {
			if showLow {
				setColor(Color{0, 0, 2})
			} else {
				setColor(Color{0, 0, 6})
			}
			showLow = !showLow

			select {
			case <-ctx.Done():
				log.Println("BootupEffect: 在4.5-5S阶段收到停止信号")
				setColor(ColorOff)
				return
			case <-time.After(50 * time.Millisecond):
			}
		}

		// 5S-5.5S 蓝灯在6和2之间闪烁、绿灯在0和2之间闪烁
		startTime = time.Now()
		duration = 500 * time.Millisecond
		showLow = true
		for time.Since(startTime) < duration {
			if showLow {
				setColor(Color{0, 0, 6})
			} else {
				setColor(Color{0, 2, 2})
			}
			showLow = !showLow

			select {
			case <-ctx.Done():
				log.Println("BootupEffect: 在5-5.5S阶段收到停止信号")
				setColor(ColorOff)
				return
			case <-time.After(50 * time.Millisecond):
			}
		}

		// 5.5S-6S 蓝灯在2和6之间闪烁，绿灯在2和6之间闪烁
		startTime = time.Now()
		duration = 500 * time.Millisecond
		showLow = true
		for time.Since(startTime) < duration {
			if showLow {
				setColor(Color{0, 2, 2})
			} else {
				setColor(Color{0, 6, 6})
			}
			showLow = !showLow

			select {
			case <-ctx.Done():
				log.Println("BootupEffect: 在5.5-6S阶段收到停止信号")
				setColor(ColorOff)
				return
			case <-time.After(50 * time.Millisecond):
			}
		}

		// 第七至八秒: 交替常亮
		// 6S-6.5S 青色常亮
		setColor(Color{0, 6, 6})
		select {
		case <-ctx.Done():
			log.Println("BootupEffect: 在6-6.5S阶段收到停止信号")
			setColor(ColorOff)
			return
		case <-time.After(500 * time.Millisecond):
		}

		// 6.5S-7S 白色常亮
		setColor(Color{6, 6, 6})
		select {
		case <-ctx.Done():
			log.Println("BootupEffect: 在6.5-7S阶段收到停止信号")
			setColor(ColorOff)
			return
		case <-time.After(500 * time.Millisecond):
		}

		// 7S-7.5S 青色常亮
		setColor(Color{0, 6, 6})
		select {
		case <-ctx.Done():
			log.Println("BootupEffect: 在7-7.5S阶段收到停止信号")
			setColor(ColorOff)
			return
		case <-time.After(500 * time.Millisecond):
		}

		// 7.5S-8S 白色常亮
		setColor(Color{6, 6, 6})
		select {
		case <-ctx.Done():
			log.Println("BootupEffect: 在7.5-8S阶段收到停止信号")
			setColor(ColorOff)
			return
		case <-time.After(500 * time.Millisecond):
		}

		// 第八至九秒: 白色常亮
		// 8.0-9S 白色常亮
		setColor(Color{6, 6, 6})
		select {
		case <-ctx.Done():
			log.Println("BootupEffect: 在8-9S阶段收到停止信号")
			setColor(ColorOff)
			return
		case <-time.After(1 * time.Second):
		}

		// 9.0S-12S 蓝色常亮
		setColor(ColorBlue)
		select {
		case <-ctx.Done():
			log.Println("BootupEffect: 在9-12S阶段收到停止信号")
			setColor(ColorOff)
			return
		case <-time.After(3 * time.Second):
		}

		// 效果结束，关闭所有灯
		log.Println("BootupEffect: 灯效执行完成，关闭所有灯")
	}, EFFECT_BOOTUP)
}

// runTimedEffect runs an effect in a goroutine with context-based cancellation
func runTimedEffect(effect func(context.Context), effectType int) error {
	mutex.Lock()

	// Cancel any existing effect
	if currentCancel != nil {
		debugLog("runTimedEffect: Cancelling previous effect")
		currentCancel()
		// Wait for previous effect to finish
		if effectDone != nil {
			mutex.Unlock()
			<-effectDone
			mutex.Lock()
		}
	}

	// Create new context for this effect
	ctx, cancel := context.WithCancel(context.Background())
	currentCancel = cancel
	effectDone = make(chan struct{})

	effectActive = true
	currentEffectType = effectType
	debugLog("runTimedEffect: Starting effect type %d", effectType)
	mutex.Unlock()

	// Run effect in goroutine
	go func() {
		defer func() {
			// Clean up
			setColor(ColorOff)

			mutex.Lock()
			effectActive = false
			currentEffectType = EFFECT_NONE
			currentCancel = nil
			mutex.Unlock()

			close(effectDone)
			debugLog("runTimedEffect: Effect completed")
		}()

		// Run the effect function
		effect(ctx)
	}()

	return nil
}

// SetRed sets only the red LED
func SetRed(value int) error {
	StopCurrentEffect()
	return setRed(value)
}

// SetGreen sets only the green LED
func SetGreen(value int) error {
	StopCurrentEffect()
	return setGreen(value)
}

// SetBlue sets only the blue LED
func SetBlue(value int) error {
	StopCurrentEffect()
	return setBlue(value)
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

// GetCurrentEffect returns the currently active effect type
func GetCurrentEffect() int {
	mutex.Lock()
	defer mutex.Unlock()

	if !effectActive {
		return EFFECT_NONE
	}

	return currentEffectType
}

// IsEffectActive returns whether an effect is currently running
func IsEffectActive() bool {
	mutex.Lock()
	defer mutex.Unlock()
	return effectActive
}

// SetLEDEnabled Sets the LED enabled state
func SetLEDEnabled(enabled bool) bool {
	mutex.Lock()
	defer mutex.Unlock()

	// 保存先前的状态用于判断是否需要关灯
	prevEnabled := ledEnabled

	// 更新LED总开关状态
	ledEnabled = enabled

	// 如果关闭LED总开关，立即关闭所有灯光，但不停止正在运行的效果
	if prevEnabled && !enabled {
		// 直接写入文件关闭LED，绕过ledEnabled检查
		ioutil.WriteFile(RedLEDPath, []byte("0"), 0644)
		ioutil.WriteFile(GreenLEDPath, []byte("0"), 0644)
		ioutil.WriteFile(BlueLEDPath, []byte("0"), 0644)
		log.Println("SetLEDEnabled: 已关闭LED灯光")
	}

	return true
}

// IsLEDEnabled returns whether the LED is enabled
func IsLEDEnabled() bool {
	mutex.Lock()
	defer mutex.Unlock()
	return ledEnabled
}

// StartEffect starts the specified effect
func StartEffect(effectType int) bool {
	if !IsLEDEnabled() {
		return false
	}

	// Stop current effect first
	StopCurrentEffect()

	// Start new effect based on type
	var err error
	switch effectType {
	case EFFECT_BOOTUP:
		err = BootupEffect()
	case EFFECT_NOTIFICATION:
		err = NotificationEffect()
	case EFFECT_CALL:
		err = CallNotificationEffect()
	case EFFECT_CHARGING_LOW:
		err = ChargingLowBatteryEffect()
	case EFFECT_CHARGING_HIGH:
		err = ChargingHighBatteryEffect()
	case EFFECT_CHARGING_COMPLETE:
		err = ChargingCompleteEffect()
	case EFFECT_WIFI_CONNECTING:
		err = WiFiConnectingEffect()
	case EFFECT_WIFI_CONNECTED:
		err = WiFiConnectedEffect()
	case EFFECT_WIFI_FAILED:
		err = WiFiFailedEffect()
	case EFFECT_BLUETOOTH_CONNECTING:
		err = BluetoothConnectingEffect()
	case EFFECT_BLUETOOTH_CONNECTED:
		err = BluetoothConnectedEffect()
	case EFFECT_BLUETOOTH_FAILED:
		err = BluetoothFailedEffect()
	case EFFECT_PARTY:
		err = PartyEffect()
	case EFFECT_CAMERA_FOCUS:
		err = CameraFocusEffect()
	case EFFECT_CAMERA_CAPTURE:
		err = CameraCaptureEffect()
	case EFFECT_CAMERA_SAVE:
		err = CameraSavePhotoEffect()
	case EFFECT_MUSIC:
		err = MusicEffect()
	default:
		return false
	}

	return err == nil
}
