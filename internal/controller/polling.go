package controller

import (
	"log"
	"time"
)

// PollSecondaryKWH polls the secondary TWCs for their usage statistics
func (p *TWCPrimary) PollSecondaryKWH() (int64, error) {
	if p.DebugLevel >= 9 {
		log.Println("PRIMARY: Poll Secondary for stats")
	}
	for _, twc := range p.knownTWCs {
		// msg := append(append(append([]byte{0xFB, 0xEB}, p.ID...), twc.TWCID...), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}...)
		msg := append(append([]byte{0xFB, 0xEB}, p.ID...), twc.TWCID...)
		padBytes(&msg)
		_, _ = SendMessage(p.DebugLevel, p.port, msg)
	}
	return time.Now().UTC().Unix(), nil
}

// PollFirmwareVersion polls the secondary TWCs for their usage statistics
func (p *TWCPrimary) PollFirmwareVersion() (int64, error) {
	if p.DebugLevel >= 9 {
		log.Println("PRIMARY: Poll firmware version")
	}
	for _, twc := range p.knownTWCs {
		// msg := append(append(append([]byte{0xFB, 0x1B}, p.ID...), twc.TWCID...), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}...)
		msg := append(append([]byte{0xFB, 0x1B}, p.ID...), twc.TWCID...)
		padBytes(&msg)
		_, _ = SendMessage(p.DebugLevel, p.port, msg)
	}
	return time.Now().UTC().Unix(), nil
}

// PollSerialNumber polls the secondary TWCs for their usage statistics
func (p *TWCPrimary) PollSerialNumber() (int64, error) {
	if p.DebugLevel >= 9 {
		log.Println("PRIMARY: Poll serial number")
	}
	for _, twc := range p.knownTWCs {
		// msg := append(append(append([]byte{0xFB, 0x19}, p.ID...), twc.TWCID...), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}...)
		msg := append(append([]byte{0xFB, 0x19}, p.ID...), twc.TWCID...)
		padBytes(&msg)
		_, _ = SendMessage(p.DebugLevel, p.port, msg)
	}
	return time.Now().UTC().Unix(), nil
}

// PollModel polls the secondary TWCs for their usage statistics
func (p *TWCPrimary) PollModel() (int64, error) {
	if p.DebugLevel >= 9 {
		log.Println("PRIMARY: Poll model")
	}
	for _, twc := range p.knownTWCs {
		// msg := append(append(append([]byte{0xFB, 0x1A}, p.ID...), twc.TWCID...), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}...)
		msg := append(append([]byte{0xFB, 0x1A}, p.ID...), twc.TWCID...)
		padBytes(&msg)
		_, _ = SendMessage(p.DebugLevel, p.port, msg)
	}
	return time.Now().UTC().Unix(), nil
}

// PollVINStart polls the secondary TWCs for the current VIN
func (p *TWCPrimary) PollVINStart() (int64, error) {
	if p.DebugLevel >= 9 {
		log.Println("PRIMARY: Poll Secondary for VIN start")
	}
	for _, twc := range p.knownTWCs {
		// msg := append(append(append([]byte{0xFB, 0xEE}, p.ID...), twc.TWCID...), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}...)
		msg := append(append([]byte{0xFB, 0xEE}, p.ID...), twc.TWCID...)
		padBytes(&msg)
		_, _ = SendMessage(p.DebugLevel, p.port, msg)
	}
	return time.Now().UTC().Unix(), nil
}

// PollVINMiddle polls the secondary TWCs for the current VIN
func (p *TWCPrimary) PollVINMiddle() (int64, error) {
	if p.DebugLevel >= 9 {
		log.Println("PRIMARY: Poll Secondary for VIN middle")
	}
	for _, twc := range p.knownTWCs {
		// msg := append(append(append([]byte{0xFB, 0xEF}, p.ID...), twc.TWCID...), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}...)
		msg := append(append([]byte{0xFB, 0xEF}, p.ID...), twc.TWCID...)
		padBytes(&msg)
		_, _ = SendMessage(p.DebugLevel, p.port, msg)
	}
	return time.Now().UTC().Unix(), nil
}

// PollVINEnd polls the secondary TWCs for the current VIN
func (p *TWCPrimary) PollVINEnd() (int64, error) {
	if p.DebugLevel >= 9 {
		log.Println("PRIMARY: Poll Secondary for VIN end")
	}
	for _, twc := range p.knownTWCs {
		// msg := append(append(append([]byte{0xFB, 0xF1}, p.ID...), twc.TWCID...), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}...)
		msg := append(append([]byte{0xFB, 0xF1}, p.ID...), twc.TWCID...)
		padBytes(&msg)
		_, _ = SendMessage(p.DebugLevel, p.port, msg)
	}
	return time.Now().UTC().Unix(), nil
}

// PollPlugState polls the secondary TWCs for their plug state
func (p *TWCPrimary) PollPlugState() (int64, error) {
	if p.DebugLevel >= 9 {
		log.Println("PRIMARY: Poll Secondary for plug state")
	}
	for _, twc := range p.knownTWCs {
		// msg := append(append(append([]byte{0xFB, 0xB4}, p.ID...), twc.TWCID...), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}...)
		msg := append(append([]byte{0xFB, 0xB4}, p.ID...), twc.TWCID...)
		padBytes(&msg)
		_, _ = SendMessage(p.DebugLevel, p.port, msg)
	}
	return time.Now().UTC().Unix(), nil
}
