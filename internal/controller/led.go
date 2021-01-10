package controller

import (
	"time"
)

// LEDValues is used for the LED values
type LEDValues struct {
	LED1 uint32
	LED2 uint32
	LED3 uint32
	LED4 uint32
	LED5 uint32
	LED6 uint32
	LED7 uint32
	LED8 uint32
}

// LED strip support
type wsEngine interface {
	Init() error
	Render() error
	Wait() error
	Fini()
	Leds(channel int) []uint32
}

type ledStrip struct {
	ws wsEngine
}

func (ls *ledStrip) setup() error {
	return ls.ws.Init()
}

func (ls *ledStrip) wipe(color uint32) error {
	for i := 0; i < len(ls.ws.Leds(0)); i++ {
		ls.ws.Leds(0)[i] = color
		if err := ls.ws.Render(); err != nil {
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

func (ls *ledStrip) display(ledValues LEDValues) error {
	ls.ws.Leds(0)[0] = ledValues.LED1
	ls.ws.Leds(0)[1] = ledValues.LED2
	ls.ws.Leds(0)[2] = ledValues.LED3
	ls.ws.Leds(0)[3] = ledValues.LED4
	ls.ws.Leds(0)[4] = ledValues.LED5
	ls.ws.Leds(0)[5] = ledValues.LED6
	ls.ws.Leds(0)[6] = ledValues.LED7
	ls.ws.Leds(0)[7] = ledValues.LED8
	time.Sleep(50 * time.Millisecond)
	if err := ls.ws.Render(); err != nil {
		return err
	}
	return nil
}

// LEDLoop is the always running loop that controls the LEDs
func (p *TWCPrimary) LEDLoop() {
	prevLEDSOn := false
	for {
		time.Sleep(10 * time.Millisecond)
		if prevLEDSOn == false && p.LEDSOn == true {
			if p.LEDCharging {
				p.LEDController.wipe(uint32(0x00ff00))
				p.LEDController.wipe(uint32(0x000000))
			} else {
				p.LEDController.display(*p.LEDValues)
			}
		} else if prevLEDSOn == true && p.LEDSOn == false {
			p.LEDController.wipe(uint32(0x000000))
		}
		prevLEDSOn = p.LEDSOn
	}
}

// SetPlugStateLED Set the color of the plugstate led
func (p *TWCPrimary) SetPlugStateLED(color uint32) {
	p.LEDValues.LED3 = color
}

// SetVINLED Set the color of the VIN led
func (p *TWCPrimary) SetVINLED(color uint32) {
	p.LEDValues.LED5 = color
}

// SetTWCStatusLED Set the color of the TWC status led
func (p *TWCPrimary) SetTWCStatusLED(color uint32) {
	p.LEDValues.LED8 = color
}

// SetLEDsOff turns all the LEDS off
func (p *TWCPrimary) SetLEDsOff() {
	p.LEDValues.LED1 = 0x000000
	p.LEDValues.LED2 = 0x000000
	p.LEDValues.LED3 = 0x000000
	p.LEDValues.LED4 = 0x000000
	p.LEDValues.LED5 = 0x000000
	p.LEDValues.LED6 = 0x000000
	p.LEDValues.LED7 = 0x000000
	p.LEDValues.LED8 = 0x000000
}
