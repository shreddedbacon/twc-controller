package controller

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	"regexp"
	"time"
)

// check if the message contains if the secondary TWC is ready to link
func (p *TWCPrimary) isSecondaryReadyToLink(msg []byte, foundMsgMatch *bool) {
	msgMatch, _ := regexp.MatchString(`^fde2(....)(..)(....)000000000000.+.+$`, fmt.Sprintf("%x", msg))
	matches := regexp.MustCompile(`^fde2(....)(..)(....)000000000000.+.+$`).FindAllStringSubmatch(fmt.Sprintf("%x", msg), -1)
	if msgMatch && *foundMsgMatch == false {
		*foundMsgMatch = true
		secondaryID, _ := hex.DecodeString(matches[0][1])
		sign, _ := hex.DecodeString(matches[0][2])
		data, _ := hex.DecodeString(matches[0][3])
		maxAmps := []byte{data[0], data[1]}
		if p.DebugLevel >= 9 {
			log.Println(log2JSONString(LogData{
				Type:     "INFO",
				Source:   "messaging",
				Sender:   fmt.Sprintf("%x", secondaryID),
				Receiver: fmt.Sprintf("%x", p.ID),
				Message:  fmt.Sprintf("Secondary TWC is ready to link, signed %x", sign),
			}))
		}
		spikeAmpsToCancel6ALimitB := []byte{}
		if Bytes2Dec2(maxAmps, false) >= 8000 {
			spikeAmpsToCancel6ALimitB = Dec2Bytes(2100)
		} else {
			spikeAmpsToCancel6ALimitB = Dec2Bytes(1600)
		}
		if p.DebugLevel >= 15 {
			// @TODO: holder for future usage?
			fmt.Println(spikeAmpsToCancel6ALimitB)
		}
		if bytes.Compare(secondaryID, p.ID) == 0 {
			// secondary should resolve by changing twcid
			p.numInitMsgsToSend = 10
			// continue // @TODO: return instead?
		}
		secondaryTWC, ok := p.GetSecondary(secondaryID)
		if !ok {
			p.AddSecondary(secondaryTWC, secondaryID)
		}
		if secondaryTWC.ProtocolVersion == 1 && secondaryTWC.MinAmpsTWCSupports == 6 {
			if len(msg) == 14 {
				secondaryTWC.ProtocolVersion = 1
				secondaryTWC.MinAmpsTWCSupports = 5
			} else if len(msg) == 16 {
				secondaryTWC.ProtocolVersion = 2
				secondaryTWC.MinAmpsTWCSupports = 6
			}
			if p.DebugLevel >= 9 {
				log.Println(log2JSONString(LogData{
					Type:     "INFO",
					Source:   "messaging",
					Sender:   fmt.Sprintf("%x", secondaryID),
					Receiver: fmt.Sprintf("%x", p.ID),
					Message:  fmt.Sprintf("Secondary TWC protocolVersion to %d, minAmpsTWCSupports to %d", secondaryTWC.ProtocolVersion, secondaryTWC.MinAmpsTWCSupports),
				}))
			}
		}
		if secondaryTWC.wiringMaxAmps > int(Bytes2Dec2(maxAmps, false)/100) {
			log.Println(log2JSONString(LogData{
				Type:     "DANGER",
				Source:   "messaging",
				Sender:   fmt.Sprintf("%x", secondaryID),
				Receiver: fmt.Sprintf("%x", p.ID),
				Message:  fmt.Sprintf("DANGER!!: wiringMaxAmpsPerTWC is %d which is greater than the max %d amps your charger says it can handle.", secondaryTWC.wiringMaxAmps, maxAmps),
			}))
			secondaryTWC.wiringMaxAmps = int(Bytes2Dec2(maxAmps, false)/100) / 4
		}
		secondaryTWC.TimeLastRx = time.Now().UTC().Unix()
		if !secondaryTWC.AllowCharge {
			// If the TWC has been told to stop charging, set the reported state to something that the TWC would probably
			// never send, so we know.
			if p.DebugLevel >= 9 {
				log.Println(log2JSONString(LogData{
					Type:     "INFO",
					Source:   "messaging",
					Sender:   fmt.Sprintf("%x", secondaryID),
					Receiver: fmt.Sprintf("%x", p.ID),
					Message:  "Secondary TWC has been disabled, setting state and charge rates to 0",
				}))
			}
			secondaryTWC.ReportedState = 99
			secondaryTWC.ReportedAmpsActual = []byte{0x00, 0x00}
			secondaryTWC.ReportedAmpsMax = []byte{0x00, 0x00}
		}
		p.timeLastTx, _ = secondaryTWC.sendPrimaryHeartbeat(p.port, p.ID)
	}
}

func (p *TWCPrimary) receiveSecondaryHeartbeatData(msg []byte, foundMsgMatch *bool) {
	msgMatch, _ := regexp.MatchString(`\Afde0(....)(....)(.............+?.+?)..$`, fmt.Sprintf("%x", msg))
	matches := regexp.MustCompile(`\Afde0(....)(....)(.............+?.+?)..$`).FindAllStringSubmatch(fmt.Sprintf("%x", msg), -1)
	if msgMatch && *foundMsgMatch == false {
		*foundMsgMatch = true
		secondaryID, _ := hex.DecodeString(matches[0][1])
		primaryID, _ := hex.DecodeString(matches[0][2])
		heartbeatData, _ := hex.DecodeString(matches[0][3])

		secondaryTWC, ok := p.GetSecondary(secondaryID)
		if ok {
			if bytes.Compare(primaryID, p.ID) == 0 {
				secondaryTWC.ReceiveSecondaryHeartbeat(heartbeatData)
				reportedAmpsActual := []byte{heartbeatData[3], heartbeatData[4]}
				if bytes.Compare(reportedAmpsActual, []byte{}) != 0 {
					if Bytes2Dec2(reportedAmpsActual, false) > 0 {
						secondaryTWC.ChargeState = true // set the TWC to be in the charging state
					} else {
						secondaryTWC.ChargeState = false // set the TWC to be in the not charging state
					}
				}
			} else {
				if p.DebugLevel >= 9 {
					log.Println(log2JSONString(LogData{
						Type:     "INFO",
						Source:   "messaging",
						Sender:   fmt.Sprintf("%x", secondaryID),
						Receiver: fmt.Sprintf("%x", p.ID),
						Message:  "Received heartbeat message from secondary TWC that we've not met before",
					}))
				}
			}
		}
	}
}

func (p *TWCPrimary) receiveVinStart(msg []byte, foundMsgMatch *bool) {
	msgMatch, _ := regexp.MatchString(`\Afdee(....)(..............)(.+?.+?)$`, fmt.Sprintf("%x", msg))
	matches := regexp.MustCompile(`\Afdee(....)(..............)(.+?.+?)$`).FindAllStringSubmatch(fmt.Sprintf("%x", msg), -1)
	if msgMatch && *foundMsgMatch == false {
		*foundMsgMatch = true
		secondaryID, _ := hex.DecodeString(matches[0][1])
		vinStart, _ := hex.DecodeString(matches[0][2])
		if p.DebugLevel >= 9 {
			log.Println(log2JSONString(LogData{
				Type:     "INFO",
				Source:   "messaging",
				Sender:   fmt.Sprintf("%x", secondaryID),
				Receiver: fmt.Sprintf("%x", p.ID),
				Message:  fmt.Sprintf("Received from VIN Start %x from secondary TWC", vinStart),
			}))
		}
		secondaryTWC, ok := p.GetSecondary(secondaryID)
		if ok {
			if bytes.Compare(vinStart, []byte{0, 0, 0, 0, 0, 0, 0}) == 0 {
				secondaryTWC.VINStart = ""
			} else {
				secondaryTWC.VINStart = fmt.Sprintf("%s", vinStart)
			}
		}
	}
}

func (p *TWCPrimary) receiveVinMiddle(msg []byte, foundMsgMatch *bool) {
	msgMatch, _ := regexp.MatchString(`\Afdef(....)(..............)(.+?.+?)$`, fmt.Sprintf("%x", msg))
	matches := regexp.MustCompile(`\Afdef(....)(..............)(.+?.+?)$`).FindAllStringSubmatch(fmt.Sprintf("%x", msg), -1)
	if msgMatch && *foundMsgMatch == false {
		*foundMsgMatch = true
		secondaryID, _ := hex.DecodeString(matches[0][1])
		vinMiddle, _ := hex.DecodeString(matches[0][2])
		if p.DebugLevel >= 9 {
			log.Println(log2JSONString(LogData{
				Type:     "INFO",
				Source:   "messaging",
				Sender:   fmt.Sprintf("%x", secondaryID),
				Receiver: fmt.Sprintf("%x", p.ID),
				Message:  fmt.Sprintf("Received from VIN Middle %x from secondary TWC", vinMiddle),
			}))
		}
		secondaryTWC, ok := p.GetSecondary(secondaryID)
		if ok {
			if bytes.Compare(vinMiddle, []byte{0, 0, 0, 0, 0, 0, 0}) == 0 {
				secondaryTWC.VINMiddle = ""
			} else {
				secondaryTWC.VINMiddle = fmt.Sprintf("%s", vinMiddle)
			}
		}
	}
}

func (p *TWCPrimary) receiveVinEnd(msg []byte, foundMsgMatch *bool) {
	msgMatch, _ := regexp.MatchString(`\Afdf1(....)(......)(.+?.+?)$`, fmt.Sprintf("%x", msg))
	matches := regexp.MustCompile(`\Afdf1(....)(......)(.+?.+?)$`).FindAllStringSubmatch(fmt.Sprintf("%x", msg), -1)
	if msgMatch && *foundMsgMatch == false {
		*foundMsgMatch = true
		secondaryID, _ := hex.DecodeString(matches[0][1])
		vinEnd, _ := hex.DecodeString(matches[0][2])
		if p.DebugLevel >= 9 {
			log.Println(log2JSONString(LogData{
				Type:     "INFO",
				Source:   "messaging",
				Sender:   fmt.Sprintf("%x", secondaryID),
				Receiver: fmt.Sprintf("%x", p.ID),
				Message:  fmt.Sprintf("Received from VIN End %x from secondary TWC", vinEnd),
			}))
		}
		secondaryTWC, ok := p.GetSecondary(secondaryID)
		if ok {
			//@TODO: if vin is 0, empty string it
			if bytes.Compare(vinEnd, []byte{0, 0, 0}) == 0 {
				secondaryTWC.VINEnd = ""
			} else {
				secondaryTWC.VINEnd = fmt.Sprintf("%s", vinEnd)
			}
		}
	}
}

func (p *TWCPrimary) receivePlugState(msg []byte, foundMsgMatch *bool) {
	msgMatch, _ := regexp.MatchString(`\Afdb4(....)(..)(.+?.+?)$`, fmt.Sprintf("%x", msg))
	matches := regexp.MustCompile(`\Afdb4(....)(..)(.+?.+?).$`).FindAllStringSubmatch(fmt.Sprintf("%x", msg), -1)
	if msgMatch && *foundMsgMatch == false {
		*foundMsgMatch = true
		secondaryID, _ := hex.DecodeString(matches[0][1])
		plugState, _ := hex.DecodeString(matches[0][2])
		if p.DebugLevel >= 9 {
			log.Println(log2JSONString(LogData{
				Type:     "INFO",
				Source:   "messaging",
				Sender:   fmt.Sprintf("%x", secondaryID),
				Receiver: fmt.Sprintf("%x", p.ID),
				Message:  fmt.Sprintf("Received from Plug state %x from secondary TWC", plugState),
			}))
		}
		// set some LED values
		switch int(plugState[0]) {
		case 0:
			p.SetPlugStateLED(0x000000) // set the LED to off to indicate that nothing is plugged in
		case 1:
			p.SetPlugStateLED(0x00ff00) // set the LED to green to indicate that a car is plugged in and charging
		case 3:
			p.SetPlugStateLED(0x0000ff) // set the LED to blue to indicate that a car is plugged in but not charging
		}
		secondaryTWC, ok := p.GetSecondary(secondaryID)
		if ok {
			prevPlugState := secondaryTWC.PlugState
			secondaryTWC.PlugState = int(plugState[0])
			if secondaryTWC.PlugState != 0 {
				splitAmps := p.AvailableAmps / len(p.knownTWCs)
				if prevPlugState == 0 {
					// when a car is plugged in, charge at the minimum 6 amps (600 Watts) for a little bit while the system identifies the VIN and current powerwall state
					// the powerwall monitoring will override this value if it needs to based on solar generation
					// or it will stop charging entirely
					splitAmps = p.MinAmpsPerTWC
				}
				_, _ = p.sendChargeRate(secondaryTWC.TWCID, Dec2Bytes(uint16(splitAmps*100)), byte(0x09))
			}
		}
	}
}

func (p *TWCPrimary) receivePeriodicPollData(msg []byte, foundMsgMatch *bool) {
	msgMatch, _ := regexp.MatchString(`\Afdeb(....)(........)(....)(....)(....)(.+?.+?)..$`, fmt.Sprintf("%x", msg))
	matches := regexp.MustCompile(`\Afdeb(....)(........)(....)(....)(....)(..)(..)(..)(.+?.+?)..$`).FindAllStringSubmatch(fmt.Sprintf("%x", msg), -1)
	if msgMatch && *foundMsgMatch == false {
		*foundMsgMatch = true
		secondaryID, _ := hex.DecodeString(matches[0][1])
		data, _ := hex.DecodeString(matches[0][2])
		p1, _ := hex.DecodeString(matches[0][3])
		p2, _ := hex.DecodeString(matches[0][4])
		p3, _ := hex.DecodeString(matches[0][5])
		p1amps, _ := hex.DecodeString(matches[0][6])
		p2amps, _ := hex.DecodeString(matches[0][7])
		p3amps, _ := hex.DecodeString(matches[0][8])

		if p.DebugLevel >= 12 {
			log.Println(log2JSONString(LogData{
				Type:     "INFO",
				Source:   "messaging",
				Sender:   fmt.Sprintf("%x", secondaryID),
				Receiver: fmt.Sprintf("%x", p.ID),
				Message: fmt.Sprintf(" Secondary TWC unexpectedly reported kWh and voltage data: %d %d %d %d %d %d %d",
					Bytes2Dec4(data, false),
					Bytes2Dec2(p1, false),
					Bytes2Dec2(p2, false),
					Bytes2Dec2(p3, false),
					int(p1amps[0])/2,
					int(p2amps[0])/2,
					int(p3amps[0])/2,
				),
			}))
		}
		secondaryTWC, ok := p.GetSecondary(secondaryID)
		if ok {
			if secondaryTWC.ReportedAmpsActual == nil {
				secondaryTWC.ReportedAmpsActual = []byte{0x00, 0x00}
			}
			if Bytes2Dec2(secondaryTWC.ReportedAmpsActual, false) > 0 {
				secondaryTWC.ChargeState = true // set the TWC to be in the charging state
			} else {
				secondaryTWC.ChargeState = false // set the TWC to be in the not charging state
			}
			currentWatts := uint32(Bytes2Dec2(p1, false) * (Bytes2Dec2(secondaryTWC.ReportedAmpsActual, false) / 100))
			if Bytes2Dec2(p2, false) != 0 && Bytes2Dec2(p3, false) != 0 {
				volts := Bytes2Dec2(p1, false) + Bytes2Dec2(p3, false) + Bytes2Dec2(p3, false)
				currentWatts = uint32(volts * (Bytes2Dec2(secondaryTWC.ReportedAmpsActual, false) / 100))
			}
			// update the stats on the secondary so we can display them :)
			secondaryTWC.StatsCurrentWatts = currentWatts
			secondaryTWC.StatsKWH = Bytes2Dec4(data, false)
			secondaryTWC.StatsP1Volts = Bytes2Dec2(p1, false)
			secondaryTWC.StatsP2Volts = Bytes2Dec2(p2, false)
			secondaryTWC.StatsP3Volts = Bytes2Dec2(p3, false)
			secondaryTWC.StatsP1Amps = int(p1amps[0]) / 2
			secondaryTWC.StatsP2Amps = int(p2amps[0]) / 2
			secondaryTWC.StatsP3Amps = int(p3amps[0]) / 2
		}
	}
}

func (p *TWCPrimary) isPrimaryTWC(msg []byte, foundMsgMatch *bool) {
	msgMatch, _ := regexp.MatchString(`\Afc(e1|e2)(....)(..)0000000000000000.+.+$`, fmt.Sprintf("%x", msg))
	// Don't need to get any matches here, nothing to do with this data yet?
	// matches := regexp.MustCompile(`\Afc(e1|e2)(....)(..)0000000000000000.+.+$`).FindAllStringSubmatch(fmt.Sprintf("%x", msg), -1)
	if msgMatch && *foundMsgMatch == false {
		*foundMsgMatch = true
		log.Println(log2JSONString(LogData{
			Type:    "ERROR",
			Source:  "messaging",
			Message: "ERR: TWC is set to Primary mode so it can't be controlled",
		}))
	}
}
