package controller

import (
	"time"
)

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

func (ls *ledStrip) display(ledValues map[int]uint32) error {
	for i := 0; i < len(ls.ws.Leds(0)); i++ {
		ls.ws.Leds(0)[i] = ledValues[i]
		if err := ls.ws.Render(); err != nil {
			return err
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err := ls.ws.Render(); err != nil {
		return err
	}
	return nil
}

// LEDLoop is the always running loop that controls the LEDs
func (p *TWCPrimary) LEDLoop() {
	for {
		time.Sleep(10 * time.Millisecond)
		if p.LEDSOn == true {
			if p.LEDCharging {
				p.LEDController.wipe(uint32(0x00ff00))
				p.LEDController.wipe(uint32(0x000000))
			} else {
				p.LEDController.display(p.LEDValues)
			}
		} else {
			p.LEDController.wipe(uint32(0x000000))
		}
	}
}

// SetPlugStateLED Set the color of the plugstate led
func (p *TWCPrimary) SetPlugStateLED(color uint32) {
	p.LEDValues[2] = color
}

// SetVINLED Set the color of the VIN led
func (p *TWCPrimary) SetVINLED(color uint32) {
	p.LEDValues[4] = color
}

// SetTWCStatusLED Set the color of the TWC status led
func (p *TWCPrimary) SetTWCStatusLED(color uint32) {
	p.LEDValues[7] = color
}

// SetLEDsOff turns all the LEDS off
func (p *TWCPrimary) SetLEDsOff() {
	for i := 0; i < 8; i++ {
		p.LEDValues[i] = 0x000000
	}
}
