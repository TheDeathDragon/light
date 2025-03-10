package ledcontroller

import (
	"fmt"
	"io/ioutil"
	"log"
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
	stopChan = make(chan bool, 5)
}

// StopCurrentEffect stops any ongoing light effect
func StopCurrentEffect() {
	mutex.Lock()
	defer mutex.Unlock()

	log.Println("StopCurrentEffect: 尝试停止当前效果")
	if effectActive {
		log.Println("StopCurrentEffect: 发送停止信号")
		select {
		case stopChan <- true:
			log.Println("StopCurrentEffect: 停止信号已发送")
		default:
			log.Println("StopCurrentEffect: 停止通道已满，无法发送信号")
		}
		time.Sleep(50 * time.Millisecond)
		log.Println("StopCurrentEffect: 等待完成")
	} else {
		log.Println("StopCurrentEffect: 当前没有活动的效果")
	}
}

// setRed sets the red LED value
func setRed(value int) error {
	if value < 0 {
		value = 0
	}
	if value > 255 {
		value = 255
	}

	valueStr := fmt.Sprintf("%d", value)
	return ioutil.WriteFile(RedLEDPath, []byte(valueStr), 0644)
}

// setGreen sets the green LED value
func setGreen(value int) error {
	if value < 0 {
		value = 0
	}
	if value > 255 {
		value = 255
	}

	valueStr := fmt.Sprintf("%d", value)
	return ioutil.WriteFile(GreenLEDPath, []byte(valueStr), 0644)
}

// setBlue sets the blue LED value
func setBlue(value int) error {
	if value < 0 {
		value = 0
	}
	if value > 255 {
		value = 255
	}

	valueStr := fmt.Sprintf("%d", value)
	return ioutil.WriteFile(BlueLEDPath, []byte(valueStr), 0644)
}

// setColor sets the LED colors
func setColor(color Color) error {
	if err := setRed(color.Red); err != nil {
		return fmt.Errorf("failed to set red LED: %v", err)
	}
	if err := setGreen(color.Green); err != nil {
		return fmt.Errorf("failed to set green LED: %v", err)
	}
	if err := setBlue(color.Blue); err != nil {
		return fmt.Errorf("failed to set blue LED: %v", err)
	}
	return nil
}

// TurnOffLED turns off all LEDs
func TurnOffLED() error {
	StopCurrentEffect()
	return setColor(ColorOff)
}

// FadeColor implements a smooth transition from one color to another
func FadeColor(from, to Color, duration time.Duration, stop <-chan bool) error {
	log.Printf("FadeColor: 开始从 %v 渐变到 %v, 持续时间 %v", from, to, duration)
	steps := 50 // 50 steps for smooth transition
	stepDuration := duration / time.Duration(steps)

	// 最大步长时间不超过20毫秒，确保能够及时响应停止信号
	if stepDuration > 20*time.Millisecond {
		steps = int(duration / (20 * time.Millisecond))
		if steps < 10 {
			steps = 10 // 至少10步，保证平滑过渡
		}
		stepDuration = duration / time.Duration(steps)
		log.Printf("FadeColor: 调整步数为 %d, 步长时间为 %v", steps, stepDuration)
	}

	for step := 0; step <= steps; step++ {
		// 更频繁地检查停止信号
		select {
		case <-stop:
			log.Println("FadeColor: 收到停止信号，关闭LED")
			// 确保在收到停止信号时关闭LED
			setColor(ColorOff)
			return nil
		default:
			progress := float64(step) / float64(steps)

			// Calculate intermediate color
			r := int(float64(from.Red) + progress*float64(to.Red-from.Red))
			g := int(float64(from.Green) + progress*float64(to.Green-from.Green))
			b := int(float64(from.Blue) + progress*float64(to.Blue-from.Blue))

			if err := setColor(Color{r, g, b}); err != nil {
				log.Printf("FadeColor: 设置颜色时出错: %v", err)
				setColor(ColorOff) // 确保在错误时关闭LED
				return err
			}

			// 将长时间的sleep分成多个短时间的sleep，以便更频繁地检查停止信号
			remainingTime := stepDuration
			for remainingTime > 0 {
				sleepTime := remainingTime
				if sleepTime > 10*time.Millisecond {
					sleepTime = 10 * time.Millisecond
				}

				select {
				case <-stop:
					log.Println("FadeColor: 在sleep期间收到停止信号，关闭LED")
					setColor(ColorOff)
					return nil
				case <-time.After(sleepTime):
					remainingTime -= sleepTime
				}
			}
		}
	}

	log.Println("FadeColor: 渐变完成")
	return nil
}

// PulseColor implements a breathing effect for a specific color
// If pulseCount is 0, it will continue indefinitely until stopped
func PulseColor(color Color, pulseCount int, pulseDuration time.Duration, stop <-chan bool) error {
	log.Printf("PulseColor: 开始脉冲效果，颜色 %v, 次数 %d, 持续时间 %v", color, pulseCount, pulseDuration)
	halfDuration := pulseDuration / 2

	for i := 0; pulseCount == 0 || i < pulseCount; i++ {
		log.Printf("PulseColor: 第 %d 次脉冲", i+1)
		// 在每次循环开始时检查停止信号
		select {
		case <-stop:
			log.Println("PulseColor: 循环开始时收到停止信号，关闭LED")
			setColor(ColorOff)
			return nil
		default:
			// 继续执行
			log.Println("PulseColor: 继续执行")
		}

		// Fade from off to color
		log.Println("PulseColor: 从关闭到亮起")
		if err := FadeColor(ColorOff, color, halfDuration, stop); err != nil {
			log.Printf("PulseColor: 亮起过程中出错: %v", err)
			setColor(ColorOff)
			return err
		}

		// 在每个阶段之间检查停止信号
		select {
		case <-stop:
			log.Println("PulseColor: 亮起后收到停止信号，关闭LED")
			setColor(ColorOff)
			return nil
		default:
			// 继续执行
			log.Println("PulseColor: 继续执行")
		}

		// Fade from color to off
		log.Println("PulseColor: 从亮起到关闭")
		if err := FadeColor(color, ColorOff, halfDuration, stop); err != nil {
			log.Printf("PulseColor: 关闭过程中出错: %v", err)
			setColor(ColorOff)
			return err
		}
	}

	log.Println("PulseColor: 脉冲效果完成")
	return nil
}

// BlinkColor implements a blinking effect for a specific color
func BlinkColor(color Color, blinkCount int, onDuration, offDuration time.Duration, stop <-chan bool) error {
	for i := 0; i < blinkCount; i++ {
		select {
		case <-stop:
			setColor(ColorOff)
			return nil
		default:
			setColor(color)
		}

		select {
		case <-stop:
			setColor(ColorOff)
			return nil
		case <-time.After(onDuration):
		}

		setColor(ColorOff)

		select {
		case <-stop:
			return nil
		case <-time.After(offDuration):
		}
	}

	return nil
}

// CallNotificationEffect implements the call notification effect:
// Red and blue alternating flashing (200ms on, 200ms off) until stopped
func CallNotificationEffect() error {
	return runTimedEffect(func(stop <-chan bool) {
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
			case <-stop:
				setColor(ColorOff)
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 灭200ms
			setColor(ColorOff)
			select {
			case <-stop:
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 切换颜色
			isRed = !isRed
		}
	})
}

// NotificationEffect implements notification effect:
// Green breathing effect, each cycle 2s (1s brighten, 1s dim), continuously until stopped
func NotificationEffect() error {
	log.Println("NotificationEffect: 开始通知效果")
	return runTimedEffect(func(stop <-chan bool) {
		// 当pulseCount=0时，PulseColor会无限循环，直到收到停止信号
		// 确保FadeColor在收到停止信号时会关闭LED
		log.Println("NotificationEffect: 调用PulseColor")
		PulseColor(ColorGreen, 0, 2*time.Second, stop)
		log.Println("NotificationEffect: PulseColor返回")
	})
}

// MusicEffect implements music effect
func MusicEffect() error {
	return runTimedEffect(func(stop <-chan bool) {
		for {
			// 第一秒
			// 0-0.2S 常亮蓝灯，0.4-0.6S，常亮蓝灯，0.8-1.0S，常亮蓝灯
			// 0-0.5S，常亮绿灯

			// 0-0.2S 常亮蓝灯，常亮绿灯
			setColor(Color{0, 255, 255})
			select {
			case <-stop:
				setColor(ColorOff)
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 0.2-0.4S 蓝灯灭
			setColor(Color{0, 255, 0})
			select {
			case <-stop:
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 0.4-0.5S 常亮蓝灯和绿灯
			setColor(Color{0, 255, 255})
			select {
			case <-stop:
				setColor(ColorOff)
				return
			case <-time.After(100 * time.Millisecond):
			}

			// 0.5-0.6S 绿灯灭
			setColor(Color{0, 0, 255})
			select {
			case <-stop:
				setColor(ColorOff)
				return
			case <-time.After(100 * time.Millisecond):
			}

			// 0.6-0.8S 蓝灯灭
			setColor(Color{0, 0, 0})
			select {
			case <-stop:
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 0.8-1.0S 常亮蓝灯
			setColor(Color{0, 0, 255})
			select {
			case <-stop:
				setColor(ColorOff)
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 0-0.5S 常亮绿灯（与蓝灯同时进行，需要单独控制绿色通道）
			// 由于时间已经过去1秒，这里不需要再执行绿灯效果

			// 第二秒
			// 1.0-1.5S，常亮蓝灯，渐变亮红灯，1.5-2.0S，常亮红灯，渐变暗蓝灯

			// 1.0-1.5S 常亮蓝灯，渐变亮红灯
			for i := 0; i <= 255; i += 5 {
				select {
				case <-stop:
					setColor(ColorOff)
					return
				default:
					// 蓝灯常亮，红灯渐变亮
					setColor(Color{i, 0, 255})
					time.Sleep(10 * time.Millisecond) // 500ms / 51步 ≈ 10ms
				}
			}

			// 1.5-2.0S 常亮红灯，渐变暗蓝灯
			for i := 255; i >= 0; i -= 5 {
				select {
				case <-stop:
					setColor(ColorOff)
					return
				default:
					// 红灯常亮，蓝灯渐变暗
					setColor(Color{255, 0, i})
					time.Sleep(10 * time.Millisecond) // 500ms / 51步 ≈ 10ms
				}
			}

			// 第三秒
			// 2-2.2S 常亮蓝灯，2.4-2.6S，常亮蓝灯，2.8-3.0S，常亮蓝灯
			// 2.0-2.5S，常亮绿灯

			// 2.0-2.2S 常亮蓝灯
			setColor(Color{0, 0, 255})
			select {
			case <-stop:
				setColor(ColorOff)
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 2.2-2.4S 蓝灯灭
			setColor(Color{0, 0, 0})
			select {
			case <-stop:
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 2.4-2.6S 常亮蓝灯
			setColor(Color{0, 0, 255})
			select {
			case <-stop:
				setColor(ColorOff)
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 2.6-2.8S 蓝灯灭
			setColor(Color{0, 0, 0})
			select {
			case <-stop:
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 2.8-3.0S 常亮蓝灯
			setColor(Color{0, 0, 255})
			select {
			case <-stop:
				setColor(ColorOff)
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 2.0-2.5S 常亮绿灯（与蓝灯同时进行，需要单独控制绿色通道）
			// 由于时间已经过去1秒，这里不需要再执行绿灯效果

			// 第四秒
			// 3.0-3.5S，渐变亮红灯，渐变亮绿灯，
			// 3.5S-4.0S，绿灯保持常亮，渐变暗红灯

			// 3.0-3.5S 渐变亮红灯，渐变亮绿灯
			for i := 0; i <= 255; i += 5 {
				select {
				case <-stop:
					setColor(ColorOff)
					return
				default:
					// 红灯和绿灯同时渐变亮
					setColor(Color{i, i, 0})
					time.Sleep(10 * time.Millisecond) // 500ms / 51步 ≈ 10ms
				}
			}

			// 3.5S-4.0S 绿灯保持常亮, 渐变暗红灯
			for i := 255; i >= 0; i -= 5 {
				select {
				case <-stop:
					setColor(ColorOff)
					return
				default:
					// 绿灯常亮，红灯渐变暗
					setColor(Color{i, 255, 0})
					time.Sleep(10 * time.Millisecond) // 500ms / 51步 ≈ 10ms
				}
			}

			// 第五秒
			// 4.0-4.3S，常亮红灯，4.4-4.6S，常亮蓝灯，4.8-4.9S，常亮蓝灯

			// 4.0-4.3S 常亮红灯
			setColor(Color{255, 0, 0})
			select {
			case <-stop:
				setColor(ColorOff)
				return
			case <-time.After(300 * time.Millisecond):
			}

			// 4.3-4.4S 灯灭
			setColor(Color{0, 0, 0})
			select {
			case <-stop:
				return
			case <-time.After(100 * time.Millisecond):
			}

			// 4.4-4.6S 常亮蓝灯
			setColor(Color{0, 0, 255})
			select {
			case <-stop:
				setColor(ColorOff)
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 4.6-4.8S 灯灭
			setColor(Color{0, 0, 0})
			select {
			case <-stop:
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 4.8-4.9S 常亮蓝灯
			setColor(Color{0, 0, 255})
			select {
			case <-stop:
				setColor(ColorOff)
				return
			case <-time.After(100 * time.Millisecond):
			}

			// 4.9-5.0S 灯灭
			setColor(Color{0, 0, 0})
			select {
			case <-stop:
				return
			case <-time.After(100 * time.Millisecond):
			}

			// 第六秒
			// 5.0-5.2S，常亮蓝灯，5.2-5.4S，常亮绿灯，5.4-5.6S，常亮蓝灯，5.6-5.8S，常亮绿灯，5.8-6.0S，常亮蓝灯

			// 5.0-5.2S 常亮蓝灯
			setColor(Color{0, 0, 255})
			select {
			case <-stop:
				setColor(ColorOff)
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 5.2-5.4S 常亮绿灯
			setColor(Color{0, 255, 0})
			select {
			case <-stop:
				setColor(ColorOff)
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 5.4-5.6S 常亮蓝灯
			setColor(Color{0, 0, 255})
			select {
			case <-stop:
				setColor(ColorOff)
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 5.6-5.8S 常亮绿灯
			setColor(Color{0, 255, 0})
			select {
			case <-stop:
				setColor(ColorOff)
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 5.8-6.0S 常亮蓝灯
			setColor(Color{0, 0, 255})
			select {
			case <-stop:
				setColor(ColorOff)
				return
			case <-time.After(200 * time.Millisecond):
			}

			// 第七秒
			// 6.0-6.5S，渐变暗蓝灯(255-80)，6.5-7.0S，渐变亮蓝灯(80-255)

			// 6.0-6.5S 渐变暗蓝灯(255-80)
			for i := 255; i >= 80; i -= 4 {
				select {
				case <-stop:
					setColor(ColorOff)
					return
				default:
					setColor(Color{0, 0, i})
					time.Sleep(10 * time.Millisecond) // 500ms / 约44步 ≈ 10ms
				}
			}

			// 6.5-7.0S 渐变亮蓝灯(80-255)
			for i := 80; i <= 255; i += 4 {
				select {
				case <-stop:
					setColor(ColorOff)
					return
				default:
					setColor(Color{0, 0, i})
					time.Sleep(10 * time.Millisecond) // 500ms / 约44步 ≈ 10ms
				}
			}

			// 第八秒
			// 7.0-7.5S，渐变亮绿灯(80-255)，7.5-8.0S，渐变暗绿灯(255-80)

			// 7.0-7.5S 渐变亮绿灯(80-255)
			for i := 80; i <= 255; i += 4 {
				select {
				case <-stop:
					setColor(ColorOff)
					return
				default:
					setColor(Color{0, i, 0})
					time.Sleep(10 * time.Millisecond) // 500ms / 约44步 ≈ 10ms
				}
			}

			// 7.5-8.0S 渐变暗绿灯(255-80)
			for i := 255; i >= 80; i -= 4 {
				select {
				case <-stop:
					setColor(ColorOff)
					return
				default:
					setColor(Color{0, i, 0})
					time.Sleep(10 * time.Millisecond) // 500ms / 约44步 ≈ 10ms
				}
			}

			// 第九秒
			// 8.0-8.7S，渐变亮红灯(80-255)，8.7-9.0S，常亮红灯

			// 8.0-8.7S 渐变亮红灯(80-255)
			for i := 80; i <= 255; i += 3 {
				select {
				case <-stop:
					setColor(ColorOff)
					return
				default:
					setColor(Color{i, 0, 0})
					time.Sleep(10 * time.Millisecond) // 700ms / 约58步 ≈ 10ms
				}
			}

			// 8.7-9.0S 常亮红灯
			setColor(Color{255, 0, 0})
			select {
			case <-stop:
				setColor(ColorOff)
				return
			case <-time.After(300 * time.Millisecond):
			}

			// 第十秒
			// 9.0-9.5S，常亮绿灯，9.5-10.0S，常亮蓝灯

			// 9.0-9.5S 常亮绿灯
			setColor(Color{0, 255, 0})
			select {
			case <-stop:
				setColor(ColorOff)
				return
			case <-time.After(500 * time.Millisecond):
			}

			// 9.5-10.0S 常亮蓝灯
			setColor(Color{0, 0, 255})
			select {
			case <-stop:
				setColor(ColorOff)
				return
			case <-time.After(500 * time.Millisecond):
			}

			// 循环结束，重新开始
		}
	})
}

// BluetoothConnectingEffect implements Bluetooth connecting effect:
// Blue flashing (300ms on, 500ms off)
func BluetoothConnectingEffect() error {
	return runTimedEffect(func(stop <-chan bool) {
		BlinkColor(ColorBlue, 0, 300*time.Millisecond, 500*time.Millisecond, stop)
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
// Red flashing (200ms on, 400ms off) for 3 times
func BluetoothFailedEffect() error {
	return runTimedEffect(func(stop <-chan bool) {
		BlinkColor(ColorRed, 3, 200*time.Millisecond, 400*time.Millisecond, stop)
		setColor(ColorOff)
	})
}

// WiFiConnectingEffect implements WiFi connecting effect:
// Green breathing effect with 1s transitions
func WiFiConnectingEffect() error {
	return runTimedEffect(func(stop <-chan bool) {
		PulseColor(ColorGreen, 0, 2*time.Second+500*time.Millisecond, stop)
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
// Red flashing (300ms on, 300ms off) for 3 times
func WiFiFailedEffect() error {
	return runTimedEffect(func(stop <-chan bool) {
		BlinkColor(ColorRed, 3, 300*time.Millisecond, 300*time.Millisecond, stop)
		setColor(ColorOff)
	})
}

// PartyEffect implements a complex light show with different patterns over 9 seconds
func PartyEffect() error {
	return runTimedEffect(func(stop <-chan bool) {
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
			case <-stop:
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
					setRed(255)
					time.Sleep(300 * time.Millisecond)
					setRed(0)
					redLightTimer.Reset(3 * time.Second)
				default:
					// 不做任何事
				}

				// 处理蓝灯（每50ms闪烁一次，亮200ms，熄灭50ms，亮度波动）
				blueIntensity := 150 + int(50*float64(time.Now().UnixNano()%100)/100.0) // 亮度波动150-200
				setBlue(blueIntensity)
				time.Sleep(200 * time.Millisecond)
				setBlue(0)
				time.Sleep(50 * time.Millisecond)

				// 处理绿灯（每100ms闪烁一次，亮200ms，熄灭100ms，亮度渐变）
				greenProgress := float64(currentTime.Milliseconds()%1000) / 1000.0 // 0-1之间的渐变进度
				greenIntensity := int(100 + 155*greenProgress)                     // 亮度从100到255渐变
				setGreen(greenIntensity)
				time.Sleep(200 * time.Millisecond)
				setGreen(0)
				time.Sleep(100 * time.Millisecond)

			case currentSecond >= 2 && currentSecond <= 4: // 第2-4秒
				// 处理红灯（每隔3秒亮起300ms）
				select {
				case <-redLightTimer.C:
					// 红灯亮起
					setRed(255)
					time.Sleep(300 * time.Millisecond)
					setRed(0)
					redLightTimer.Reset(3 * time.Second)
				default:
					// 不做任何事
				}

				// 处理蓝灯（每50ms闪烁一次，亮200ms，熄灭50ms，亮度波动）
				blueIntensity := 150 + int(50*float64(time.Now().UnixNano()%100)/100.0) // 亮度波动150-200
				setBlue(blueIntensity)
				time.Sleep(200 * time.Millisecond)
				setBlue(0)
				time.Sleep(50 * time.Millisecond)

				// 处理绿灯（每100ms闪烁一次，亮200ms，熄灭100ms，亮度波动）
				greenIntensity := 150 + int(50*float64(time.Now().UnixNano()%100)/100.0) // 亮度波动150-200
				setGreen(greenIntensity)
				time.Sleep(200 * time.Millisecond)
				setGreen(0)
				time.Sleep(100 * time.Millisecond)

			case currentSecond == 5: // 第5秒
				// 多彩过渡：蓝色渐变到紫色，紫色渐变为绿色，再从绿色渐变为黄色，整个过程持续500ms
				transitionStart := time.Now()
				for time.Since(transitionStart) < 500*time.Millisecond {
					progress := float64(time.Since(transitionStart).Milliseconds()) / 500.0 // 0-1之间的进度

					// 根据进度计算当前颜色
					var r, g, b int
					if progress < 0.33 { // 蓝色到紫色
						subProgress := progress / 0.33
						r = int(255 * subProgress)
						b = 255
						g = 0
					} else if progress < 0.66 { // 紫色到绿色
						subProgress := (progress - 0.33) / 0.33
						r = int(255 * (1 - subProgress))
						b = int(255 * (1 - subProgress))
						g = int(255 * subProgress)
					} else { // 绿色到黄色
						subProgress := (progress - 0.66) / 0.34
						r = int(255 * subProgress)
						g = 255
						b = 0
					}

					setColor(Color{r, g, b})

					// 检查是否需要停止
					select {
					case <-stop:
						setColor(ColorOff)
						return
					default:
						time.Sleep(10 * time.Millisecond) // 小间隔使过渡更平滑
					}
				}

				// 继续处理蓝灯和绿灯的闪烁
				for i := 0; i < 3; i++ { // 执行几次闪烁循环
					// 蓝灯：每50ms闪烁一次，亮200ms，熄灭50ms
					setBlue(200)
					time.Sleep(200 * time.Millisecond)
					setBlue(0)
					time.Sleep(50 * time.Millisecond)

					// 绿灯：每100ms闪烁一次，亮200ms，熄灭100ms
					setGreen(200)
					time.Sleep(200 * time.Millisecond)
					setGreen(0)
					time.Sleep(100 * time.Millisecond)

					// 红灯点缀
					if i == 1 {
						setRed(255)
						time.Sleep(100 * time.Millisecond)
						setRed(0)
					}
				}

			case currentSecond >= 6 && currentSecond <= 8: // 第6-8秒
				// 蓝灯：每50ms闪烁一次，亮200ms，熄灭50ms
				setBlue(200)
				time.Sleep(200 * time.Millisecond)
				setBlue(0)
				time.Sleep(50 * time.Millisecond)

				// 绿灯：每100ms闪烁一次，亮200ms，熄灭100ms
				setGreen(200)
				time.Sleep(200 * time.Millisecond)
				setGreen(0)
				time.Sleep(100 * time.Millisecond)

			case currentSecond == 9: // 第9秒
				// 蓝、绿灯交替闪烁，亮度波动
				for i := 0; i < 5; i++ { // 执行几次交替闪烁
					// 蓝灯闪烁
					blueIntensity := 150 + int(100*float64(time.Now().UnixNano()%100)/100.0) // 亮度波动150-250
					setBlue(blueIntensity)
					time.Sleep(100 * time.Millisecond)
					setBlue(0)

					// 绿灯闪烁
					greenIntensity := 150 + int(100*float64(time.Now().UnixNano()%100)/100.0) // 亮度波动150-250
					setGreen(greenIntensity)
					time.Sleep(150 * time.Millisecond)
					setGreen(0)

					// 检查是否需要停止
					select {
					case <-stop:
						setColor(ColorOff)
						return
					default:
						// 继续执行
					}
				}
			}

			// 检查是否需要停止
			select {
			case <-stop:
				setColor(ColorOff)
				return
			default:
				// 继续执行，短暂休眠以避免CPU过度使用
				time.Sleep(10 * time.Millisecond)
			}
		}

		// 效果结束，关闭所有灯
		setColor(ColorOff)
	})
}

// ChargingLowBatteryEffect implements low battery charging effect:
// Red breathing (1s brighten, 1s dim), continuously until stopped
func ChargingLowBatteryEffect() error {
	return runTimedEffect(func(stop <-chan bool) {
		PulseColor(ColorRed, 0, 2*time.Second, stop)
		setColor(ColorOff)
	})
}

// ChargingHighBatteryEffect implements high battery charging effect:
// Green breathing (1s brighten, 1s dim), continuously until stopped
func ChargingHighBatteryEffect() error {
	return runTimedEffect(func(stop <-chan bool) {
		PulseColor(ColorGreen, 0, 2*time.Second, stop)
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
// Solid orange for 2 seconds (R255 G128 B0)
func CameraFocusEffect() error {
	return runTimedEffect(func(stop <-chan bool) {
		// 橙色 R255 G128 B0
		setColor(Color{255, 128, 0})
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
// Solid white for 1 second, then off for 0.5 second, then solid white for 0.2 second
func CameraCaptureEffect() error {
	return runTimedEffect(func(stop <-chan bool) {
		// 白色常亮1秒
		setColor(Color{255, 255, 255})
		select {
		case <-stop:
			setColor(ColorOff)
			return
		case <-time.After(1 * time.Second):
		}

		// 熄灭0.5秒
		setColor(ColorOff)
		select {
		case <-stop:
			return
		case <-time.After(500 * time.Millisecond):
		}

		// 白色常亮0.2秒
		setColor(Color{255, 255, 255})
		select {
		case <-stop:
			setColor(ColorOff)
			return
		case <-time.After(200 * time.Millisecond):
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
		case <-time.After(1 * time.Second):
			setColor(ColorOff)
		}
	})
}

// BootupEffect implements boot-up effect:
// Red -> Green -> Blue sequence, each color for 1s with 0.2s interval
func BootupEffect() error {
	return runTimedEffect(func(stop <-chan bool) {
		colors := []Color{ColorRed, ColorGreen, ColorBlue}

		for _, color := range colors {
			// Show color
			setColor(color)
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
		}

		setColor(ColorOff)
	})
}

// runTimedEffect runs an effect in a goroutine with proper mutex locking
func runTimedEffect(effect func(<-chan bool)) error {
	mutex.Lock()
	log.Println("runTimedEffect: 开始运行效果")

	// Stop any running effect
	if effectActive {
		log.Println("runTimedEffect: 停止当前运行的效果")
		select {
		case stopChan <- true:
			log.Println("runTimedEffect: 停止信号已发送")
		default:
			log.Println("runTimedEffect: 停止通道已满，无法发送信号")
		}
		time.Sleep(50 * time.Millisecond)
	}

	// 清空停止通道
	drainCount := 0
	for {
		select {
		case <-stopChan:
			drainCount++
			log.Printf("runTimedEffect: 清空停止通道，已清空%d个信号", drainCount)
		default:
			log.Printf("runTimedEffect: 停止通道已清空，共清空%d个信号", drainCount)
			goto channelDrained
		}
	}
channelDrained:

	effectActive = true
	log.Println("runTimedEffect: 设置effectActive为true")
	mutex.Unlock()

	// Run the effect in a goroutine
	go func() {
		log.Println("runTimedEffect: 启动效果goroutine")
		localStopChan := make(chan bool, 5)
		log.Println("runTimedEffect: 创建本地停止通道")

		// 创建一个单独的goroutine来监听停止信号
		// 这个goroutine会一直存在直到收到停止信号
		stopMonitor := make(chan struct{})
		go func() {
			log.Println("runTimedEffect: 启动监听停止信号的goroutine")
			defer log.Println("runTimedEffect: 监听停止信号的goroutine结束")
			defer close(stopMonitor)

			for {
				select {
				case <-stopChan:
					log.Println("runTimedEffect: 收到全局停止信号")
					// 使用非阻塞方式发送本地停止信号
					select {
					case localStopChan <- true:
						log.Println("runTimedEffect: 发送本地停止信号成功")
					default:
						log.Println("runTimedEffect: 本地停止通道已满，无法发送信号")
					}

					// 确保停止信号被传递，即使effect函数没有及时检查
					// 多次尝试发送停止信号，增加被接收的机会
					for i := 0; i < 10; i++ {
						select {
						case localStopChan <- true:
							log.Printf("runTimedEffect: 第%d次成功发送额外停止信号", i+1)
						default:
							log.Printf("runTimedEffect: 第%d次通道已满，跳过", i+1)
						}
						time.Sleep(10 * time.Millisecond)
					}
					log.Println("runTimedEffect: 完成发送所有停止信号")
					return // 收到停止信号后退出goroutine
				case <-time.After(100 * time.Millisecond):
					// 定期检查，防止goroutine永远阻塞
					if !effectActive {
						log.Println("runTimedEffect: 效果已不再活动，停止监听")
						return
					}
				}
			}
		}()

		log.Println("runTimedEffect: 调用effect函数")
		effect(localStopChan)
		log.Println("runTimedEffect: effect函数返回")

		// 等待监听goroutine结束
		<-stopMonitor
		log.Println("runTimedEffect: 监听goroutine已结束")

		mutex.Lock()
		effectActive = false
		log.Println("runTimedEffect: 设置effectActive为false")
		mutex.Unlock()
		log.Println("runTimedEffect: 效果goroutine结束")
	}()

	log.Println("runTimedEffect: 返回nil")
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

// IsEffectActive returns whether an effect is currently running
func IsEffectActive() bool {
	mutex.Lock()
	defer mutex.Unlock()
	return effectActive
}
