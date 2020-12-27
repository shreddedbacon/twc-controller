package controller

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/shreddedbacon/tesla"
	"github.com/tarm/serial"
	"gopkg.in/yaml.v2"

	ws2811 "github.com/rpi-ws281x/rpi-ws281x-go"
)

var spikeAmpsToCancel6ALimit = 16

// TWCPrimary is the primary structure of the TWC controller
type TWCPrimary struct {
	port                   *serial.Port    `yaml:"-"`
	ID                     []byte          `yaml:"-"` // []byte{0x77, 0x77}
	sign                   []byte          `yaml:"-"` // []byte{0x77}
	WiringMaxAmpsAllTWC    int             `yaml:"wiringMaxAmpsAllTWC"`
	WiringMaxAmpsPerTWC    int             `yaml:"wiringMaxAmpsPerTWC"`
	MinAmpsPerTWC          int             `yaml:"minAmpsPerTWC"` // When tracking Solar/Powerwall usage, this is the minimum value to allow charging at (12A = 2880W)
	SupplyVoltage          int             `yaml:"supplyVoltage"` // Voltage of a single phase, used to convert watts to amps
	SupplyPhases           int             `yaml:"supplyPhases"`  // Voltage of a single phase, used to convert watts to amps
	knownTWCs              []*TWCSecondary // slice of all the TWCs that this primary knows about
	DebugLevel             int             `yaml:"debugLevel"`
	timeLastTx             int64           `yaml:"-"`
	numInitMsgsToSend      int             `yaml:"-"`
	SerialConfig           SerialConfig    `yaml:"serial"`
	ConfigPath             string          `yaml:"-"`
	AvailableAmps          int             `yaml:"availableAmps"`
	Powerwall              string          `yaml:"powerwall"`
	EnablePowerwall        bool            `yaml:"enablePowerwall"`
	AutoStartStopInterval  bool            `yaml:"autoStartStopInterval"`
	PowerOffset            int             `yaml:"powerOffset"`
	PowerwallCheckInterval int             `yaml:"powerwallCheckInterval"`
	TeslaAPITokens         []*TeslaAPIUser `yaml:"-"` // slice of all known tesla api tokens
	timeLastVINCron        int64           `yaml:"-"`
	timeLastStatePoll      int64           `yaml:"-"`
	timeLastSecondaryPoll  int64           `yaml:"-"`
	timeLastPowerwallCheck int64           `yaml:"-"`
	twcNextHeartbeatID     int             `yaml:"-"`
	LEDSOn                 bool            `yaml:"ledEnable"`
	LEDController          *ledStrip       `yaml:"-"`
	LEDValues              *LEDValues      `yaml:"-"`
	LEDCharging            bool            `yaml:"-"`
}

// TeslaAPIUser holds the API user
type TeslaAPIUser struct {
	Username string
	Token    *tesla.Token
}

// SerialConfig contains the serial port configuration
type SerialConfig struct {
	DevicePath string `yaml:"port"`
	BaudRate   int    `yaml:"baudRate"`
}

// LogData is used to encode a log to JSON for shipping somewhere later on
type LogData struct {
	Type     string `json:"type"`
	Source   string `json:"source"`
	Sender   string `json:"sender,omitempty"`
	Receiver string `json:"receiver,omitempty"`
	Message  string `json:"message"`
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// NewPrimary creates a new primary TWC controller.
func NewPrimary(primary TWCPrimary, port *serial.Port) (*TWCPrimary, error) {
	// func NewPrimary(newPrimaryID []byte, wiringMaxAmpsAllTWC int, wiringMaxAmpsPerTWC int, port *serial.Port, DebugLevel int, sign []byte, configPath string) (*TWCPrimary, error) {
	if primary.SupplyPhases != 1 && primary.SupplyPhases != 3 {
		return nil, fmt.Errorf("supply phases should be 1 or 2")
	}
	if primary.SupplyVoltage < 100 || primary.SupplyVoltage > 260 {
		return nil, fmt.Errorf("supply voltage should be between 100 or 260")
	}

	// LED Controller setup
	opt := ws2811.DefaultOptions
	opt.Channels[0].Brightness = 150
	opt.Channels[0].LedCount = 8

	dev, err := ws2811.MakeWS2811(&opt)
	if err != nil {
		log.Fatal(err)
	}
	// start the LED loop controller
	ls := &ledStrip{
		ws: dev,
	}
	err = ls.setup()
	if err != nil {
		log.Fatal(err)
	}

	// get the env POWERWALL_HOST as an override if defined
	primary.Powerwall = getEnv("POWERWALL_HOST", primary.Powerwall)
	primary.sign = []byte{0x77}
	primary.timeLastTx = int64(0)
	primary.numInitMsgsToSend = 10
	primary.timeLastVINCron = time.Now().UTC().Unix()
	primary.timeLastStatePoll = time.Now().UTC().Unix()
	primary.timeLastSecondaryPoll = time.Now().UTC().Unix()
	primary.timeLastPowerwallCheck = time.Now().UTC().Unix()
	primary.port = port
	primary.LEDController = ls
	primary.LEDCharging = false
	primary.LEDValues = &LEDValues{
		// values are RGB hex values, 00 meaning off, ff being full colour (255)
		LED1: 0x00ff00, // when not charging, set the first LED to green to indicate it is powered up (similar to the actual TWC)
		LED2: 0x000000,
		LED3: 0x000000, // plugstate color to indicate if a car is plugged in or not etc
		LED4: 0x000000,
		LED5: 0x000000, // full vin number check (green 3 parts, orange 2 parts, red 1 part, off no vin)
		LED6: 0x000000,
		LED7: 0x000000,
		LED8: 0x000000, // the status of connected TWCs (green good, red bad, orange none found)
	}
	return &primary, nil
}

func (p *TWCPrimary) writeConfig() error {
	d, _ := yaml.Marshal(p)
	err := ioutil.WriteFile(p.ConfigPath, d, 0644)
	if err != nil {
		return fmt.Errorf(`{"error":"unable to write config file: %v"}`, err)
	}
	return nil
}

func (p *TWCPrimary) sendPrimaryLinkReady1() (int64, error) {
	if p.DebugLevel >= 9 {
		log.Println(log2JSONString(LogData{
			Type:    "INFO",
			Source:  "primary",
			Sender:  fmt.Sprintf("%x", p.ID),
			Message: "Sending primary linkready1",
		}))
	}
	msg := append(
		append(
			[]byte{0xFC, 0xE1},
			p.ID...),
		p.sign...)
	padBytes(&msg)
	return SendMessage(p.DebugLevel, p.port, msg)
}

func (p *TWCPrimary) sendPrimaryLinkReady2() (int64, error) {
	if p.DebugLevel >= 9 {
		log.Println(log2JSONString(LogData{
			Type:    "INFO",
			Source:  "primary",
			Sender:  fmt.Sprintf("%x", p.ID),
			Message: "Sending primary linkready2",
		}))
	}
	msg := append(
		append(
			[]byte{0xFB, 0xE2},
			p.ID...),
		p.sign...)
	padBytes(&msg)
	return SendMessage(p.DebugLevel, p.port, msg)
}

// sendChargeRate sends the desiredcharge rate to the receiver
func (p *TWCPrimary) sendChargeRate(secondaryID []byte, chargeRate []byte, cmd byte) (int64, error) {
	if p.DebugLevel >= 9 {
		// displaying the chargerate we need to divide the given value by 100
		cr := float64(Bytes2Dec2(chargeRate, true) / 100)
		log.Println(log2JSONString(LogData{
			Type:     "INFO",
			Source:   "primary",
			Sender:   fmt.Sprintf("%x", p.ID),
			Receiver: fmt.Sprintf("%x", secondaryID),
			Message:  fmt.Sprintf("Sending charge rate %05.2fA to secondary", cr),
		}))
	}
	msg := append(
		append(
			append(
				[]byte{0xFB, 0xE0},
				p.ID...),
			secondaryID...),
		[]byte{cmd, chargeRate[1], chargeRate[0]}...)
	padBytes(&msg)
	return SendMessage(p.DebugLevel, p.port, msg)
}

// sendStopCommand sends the desiredcharge rate to the receiver
func (p *TWCPrimary) sendStopCommand(secondaryID []byte) (int64, error) {
	if p.DebugLevel >= 9 {
		log.Println(log2JSONString(LogData{
			Type:     "INFO",
			Source:   "primary",
			Sender:   fmt.Sprintf("%x", p.ID),
			Receiver: fmt.Sprintf("%x", secondaryID),
			Message:  "Sending stop command to secondary",
		}))
	}
	msg := append(
		append(
			[]byte{0xFC, 0xB2},
			p.ID...),
		secondaryID...)
	padBytes(&msg)
	return SendMessage(p.DebugLevel, p.port, msg)
}

// sendStartCommand sends the desiredcharge rate to the receiver
func (p *TWCPrimary) sendStartCommand(secondaryID []byte) (int64, error) {
	if p.DebugLevel >= 9 {
		log.Println(log2JSONString(LogData{
			Type:     "INFO",
			Source:   "primary",
			Sender:   fmt.Sprintf("%x", p.ID),
			Receiver: fmt.Sprintf("%x", secondaryID),
			Message:  "Sending start command to secondary",
		}))
	}
	msg := append(
		append(
			[]byte{0xFC, 0xB1},
			p.ID...),
		secondaryID...)
	padBytes(&msg)
	return SendMessage(p.DebugLevel, p.port, msg)
}

// HasTWC checks if the primary has a TWC already
func (p *TWCPrimary) HasTWC(id []byte) (int, bool) {
	for i, item := range p.knownTWCs {
		if bytes.Compare(item.TWCID, id) == 0 {
			return i, true
		}
	}
	return 0, false
}

// AddSecondary adds a secondary TWC to the primary.
func (p *TWCPrimary) AddSecondary(secondaryTWC *TWCSecondary, secondaryID []byte) {
	_, ok := p.HasTWC(secondaryID)
	if !ok {
		if p.DebugLevel >= 12 {
			log.Println(log2JSONString(LogData{
				Type:     "INFO",
				Source:   "primary",
				Sender:   fmt.Sprintf("%x", p.ID),
				Receiver: fmt.Sprintf("%x", secondaryID),
				Message:  "Secondary TWC is a new TWC",
			}))
		}
		secondaryTWC, _ = NewTWCSecondary(secondaryID, p.WiringMaxAmpsPerTWC, p.port, p.WiringMaxAmpsAllTWC, p.DebugLevel)
		p.knownTWCs = append(p.knownTWCs, secondaryTWC)
	}
}

// GetSecondary returns a secondary TWC if one is already connected.
func (p *TWCPrimary) GetSecondary(secondaryID []byte) (*TWCSecondary, bool) {
	idx, ok := p.HasTWC(secondaryID)
	if ok {
		if p.DebugLevel >= 12 {
			log.Println(log2JSONString(LogData{
				Type:     "INFO",
				Source:   "primary",
				Sender:   fmt.Sprintf("%x", p.ID),
				Receiver: fmt.Sprintf("%x", secondaryID),
				Message:  "Secondary TWC already found",
			}))
		}
		return p.knownTWCs[idx], true
	}
	return &TWCSecondary{}, false
}

// RemoveSecondary removes a secondary TWC.
func (p *TWCPrimary) RemoveSecondary(i int) {
	p.knownTWCs[i] = p.knownTWCs[len(p.knownTWCs)-1]
	p.knownTWCs = p.knownTWCs[:len(p.knownTWCs)-1]
}

// PreStart pre-starts the controller by running the link ready process initially
func (p *TWCPrimary) PreStart() {
	time.Sleep(2 * time.Second)

	if p.numInitMsgsToSend > 5 {
		p.timeLastTx, _ = p.sendPrimaryLinkReady1()
		time.Sleep(100 * time.Millisecond)
		p.numInitMsgsToSend--
	} else if p.numInitMsgsToSend > 0 {
		p.timeLastTx, _ = p.sendPrimaryLinkReady2()
		time.Sleep(100 * time.Millisecond)
		p.numInitMsgsToSend = p.numInitMsgsToSend - 1
	}

	time.Sleep(2 * time.Second)
}

// Run runs the primary controller.
func (p *TWCPrimary) Run() {
	var idxSecondaryToSendNextHeartbeat int

	var vinSCount = 0
	var vinMCount = 0
	var vinECount = 0
	var kwhCount = 0
	var plugCount = 0

	for {
		time.Sleep(25 * time.Millisecond)
		now := time.Now().UTC().Unix()

		if p.numInitMsgsToSend > 5 {
			p.timeLastTx, _ = p.sendPrimaryLinkReady1()
			time.Sleep(100 * time.Millisecond)
			p.numInitMsgsToSend--
			continue
		} else if p.numInitMsgsToSend > 0 {
			p.timeLastTx, _ = p.sendPrimaryLinkReady2()
			time.Sleep(100 * time.Millisecond)
			p.numInitMsgsToSend = p.numInitMsgsToSend - 1
			continue
		} else {
			// @TODO: remove this after testing that it works in cron.go
			// After finishing the 5 startup linkready1 and linkready2
			if (now - p.timeLastTx) > 0 {
				if len(p.knownTWCs) > 0 {
					secondaryTWC := p.knownTWCs[idxSecondaryToSendNextHeartbeat]
					if (now - secondaryTWC.TimeLastRx) >= 26 {
						if p.DebugLevel >= 12 {
							log.Println(log2JSONString(LogData{
								Type:     "INFO",
								Source:   "primary",
								Sender:   fmt.Sprintf("%x", p.ID),
								Receiver: fmt.Sprintf("%x", secondaryTWC.TWCID),
								Message:  "Have not heard from secondary TWC for 26 seconds, removing.",
							}))
						}
						p.RemoveSecondary(idxSecondaryToSendNextHeartbeat)
					} else {
						if p.DebugLevel >= 12 {
							log.Println(log2JSONString(LogData{
								Type:     "INFO",
								Source:   "primary",
								Sender:   fmt.Sprintf("%x", p.ID),
								Receiver: fmt.Sprintf("%x", secondaryTWC.TWCID),
								Message:  "Sending heartbeat to secondary TWC",
							}))
						}
						p.timeLastTx, _ = secondaryTWC.sendPrimaryHeartbeat(p.port, p.ID)
						time.Sleep(200 * time.Millisecond)
					}
					idxSecondaryToSendNextHeartbeat++
					if idxSecondaryToSendNextHeartbeat >= len(p.knownTWCs) {
						idxSecondaryToSendNextHeartbeat = 0
					}
				}
			}
		}
		p.ReadMessage()

		if vinSCount == 9 {
			for _, twc := range p.knownTWCs {
				if p.DebugLevel >= 15 {
					log.Println(log2JSONString(LogData{
						Type:     "INFO",
						Source:   "polling",
						Sender:   fmt.Sprintf("%x", p.ID),
						Receiver: fmt.Sprintf("%x", twc.TWCID),
						Message:  "Poll Secondary for VIN start",
					}))
				}
				msg := append(append([]byte{0xFB, 0xEE}, p.ID...), twc.TWCID...)
				padBytes(&msg)
				_, _ = SendMessage(p.DebugLevel, p.port, msg)
				time.Sleep(200 * time.Millisecond)
				p.ReadMessage()
			}
			vinSCount = 0
		}
		if vinMCount == 10 {
			for _, twc := range p.knownTWCs {
				if p.DebugLevel >= 15 {
					log.Println(log2JSONString(LogData{
						Type:     "INFO",
						Source:   "polling",
						Sender:   fmt.Sprintf("%x", p.ID),
						Receiver: fmt.Sprintf("%x", twc.TWCID),
						Message:  "Poll Secondary for VIN middle",
					}))
				}
				msg := append(append([]byte{0xFB, 0xEF}, p.ID...), twc.TWCID...)
				padBytes(&msg)
				_, _ = SendMessage(p.DebugLevel, p.port, msg)
				time.Sleep(200 * time.Millisecond)
				p.ReadMessage()
			}
			vinMCount = 0
		}
		if vinECount == 11 {
			for _, twc := range p.knownTWCs {
				if p.DebugLevel >= 15 {
					log.Println(log2JSONString(LogData{
						Type:     "INFO",
						Source:   "polling",
						Sender:   fmt.Sprintf("%x", p.ID),
						Receiver: fmt.Sprintf("%x", twc.TWCID),
						Message:  "Poll Secondary for VIN end",
					}))
				}
				msg := append(append([]byte{0xFB, 0xF1}, p.ID...), twc.TWCID...)
				padBytes(&msg)
				_, _ = SendMessage(p.DebugLevel, p.port, msg)
				time.Sleep(200 * time.Millisecond)
				p.ReadMessage()
			}
			vinECount = 0
		}
		if kwhCount == 12 {
			for _, twc := range p.knownTWCs {
				if p.DebugLevel >= 15 {
					log.Println(log2JSONString(LogData{
						Type:     "INFO",
						Source:   "polling",
						Sender:   fmt.Sprintf("%x", p.ID),
						Receiver: fmt.Sprintf("%x", twc.TWCID),
						Message:  "Poll secondary for stats",
					}))
				}
				msg := append(append([]byte{0xFB, 0xEB}, p.ID...), twc.TWCID...)
				padBytes(&msg)
				_, _ = SendMessage(p.DebugLevel, p.port, msg)
				time.Sleep(200 * time.Millisecond)
				p.ReadMessage()
			}
			kwhCount = 0
		}
		if plugCount == 5 {
			for _, twc := range p.knownTWCs {
				if p.DebugLevel >= 15 {
					log.Println(log2JSONString(LogData{
						Type:     "INFO",
						Source:   "polling",
						Sender:   fmt.Sprintf("%x", p.ID),
						Receiver: fmt.Sprintf("%x", twc.TWCID),
						Message:  "Poll Secondary for plug state",
					}))
				}
				msg := append(append([]byte{0xFB, 0xB4}, p.ID...), twc.TWCID...)
				padBytes(&msg)
				_, _ = SendMessage(p.DebugLevel, p.port, msg)
				time.Sleep(200 * time.Millisecond)
				p.ReadMessage()
			}
			plugCount = 0
		}

		plugCount++
		vinSCount++
		vinMCount++
		vinECount++
		kwhCount++
	}
}

// ReadMessage reads message from serial port
func (p *TWCPrimary) ReadMessage() {
	var ignoredData []byte
	var lastTWCResponseMsg []byte
	var msgRxCount = 0
	var msg []byte
	var msgLen int
	// get the message
	for {
		// this section is where we read in the bytes that may be on the serial line until we have our data
		var dataLen int
		var data []byte
		dataLen = 1
		buf := make([]byte, dataLen)
		n, _ := p.port.Read(buf[:])
		data = buf[:n]
		if len(data) == 0 {
			break
		}
		if msgLen == 0 && data[0] != 0xC0 {
			if p.DebugLevel >= 12 {
				log.Println(log2JSONString(LogData{
					Type:    "INFO",
					Source:  "primary",
					Sender:  fmt.Sprintf("%x", p.ID),
					Message: "Ignoring data if length is 0 or first byte is not C0",
				}))
			}
			ignoredData = append(ignoredData, data...)
		} else if msgLen > 0 && msgLen < 15 && data[0] == 0xC0 {
			if p.DebugLevel >= 12 {
				log.Println(log2JSONString(LogData{
					Type:    "INFO",
					Source:  "primary",
					Sender:  fmt.Sprintf("%x", p.ID),
					Message: "Found end of message before full message received",
				}))
				// found end of message before full message received
			}
			msg = data
			msgLen = 1
		}
		if msgLen == 0 {
			msg = []byte{}
		}
		msg = append(msg, data...)
		msgLen++
		if msgLen >= 16 && data[0] == 0xC0 {
			break
		}
	}

	if msgLen >= 16 {
		msg = unescapeMessage(msg, msgLen)
		msgLen = 0
		msgRxCount++
		if bytes.Compare(lastTWCResponseMsg, []byte{}) == 0 &&
			bytes.Compare(msg[0:2], []byte{0xFB, 0xE0}) == 0 && bytes.Compare(msg[0:2], []byte{0xFD, 0xE0}) == 0 &&
			bytes.Compare(msg[0:2], []byte{0xFC, 0xE1}) == 0 && bytes.Compare(msg[0:2], []byte{0xFB, 0xE2}) == 0 &&
			bytes.Compare(msg[0:2], []byte{0xFD, 0xE2}) == 0 && bytes.Compare(msg[0:2], []byte{0xFB, 0xEB}) == 0 &&
			bytes.Compare(msg[0:2], []byte{0xFD, 0xEB}) == 0 && bytes.Compare(msg[0:2], []byte{0xFD, 0xE0}) == 0 {
			lastTWCResponseMsg = msg
		}
		var debugBytes []byte
		for _, dByte := range msg {
			debugByte := []byte(fmt.Sprintf("%0X ", dByte))
			debugBytes = append(debugBytes, debugByte...)
		}
		if p.DebugLevel >= 1 {
			if p.DebugLevel > 1 {
				log.Println(log2JSONString(LogData{
					Type:    "DEBUG",
					Source:  "primary",
					Message: fmt.Sprintf("Rx@: (%0X) %s", ignoredData, debugBytes),
				}))
			} else {
				log.Println(log2JSONString(LogData{
					Type:    "DEBUG",
					Source:  "primary",
					Message: fmt.Sprintf("Rx@: %s", debugBytes),
				}))
			}
		}
		ignoredData = []byte{}
		if len(msg) != 14 && len(msg) != 16 && len(msg) != 20 {
			// ignoring message of unexpected length
			var debugBytes []byte
			for _, debugB := range msg {
				debubByte := []byte(fmt.Sprintf("%X ", debugB))
				debugBytes = append(debugBytes, debubByte...)
			}
			if p.DebugLevel >= 2 {
				log.Println(log2JSONString(LogData{
					Type:    "DEBUG",
					Source:  "primary",
					Sender:  fmt.Sprintf("%x", p.ID),
					Message: fmt.Sprintf("Ignoring message of unexpected length, msg: %s", debugBytes),
				}))
			}
			return
		}
		checksumExpected := msg[len(msg)-1]
		checksum := 0
		for b := 1; b < len(msg)-1; b++ {
			checksum = checksum + int(msg[b])
		}
		if byte(checksum&0xFF) != checksumExpected {
			// checksum does not match
			var debugBytes []byte
			for _, debugB := range msg {
				debubByte := []byte(fmt.Sprintf("%X ", debugB))
				debugBytes = append(debugBytes, debubByte...)
			}
			if p.DebugLevel >= 2 {
				log.Println(log2JSONString(LogData{
					Type:    "DEBUG",
					Source:  "primary",
					Sender:  fmt.Sprintf("%x", p.ID),
					Message: fmt.Sprintf("Checksum does not match %x %x expected: %x; msg: %s", checksum&0xFF, byte(checksum&0xFF), checksumExpected, debugBytes),
				}))
			}
			return
		}
		foundMsgMatch := false
		p.isSecondaryReadyToLink(msg, &foundMsgMatch)
		p.receiveSecondaryHeartbeatData(msg, &foundMsgMatch)
		p.receivePeriodicPollData(msg, &foundMsgMatch)
		p.receiveVinStart(msg, &foundMsgMatch)
		p.receiveVinMiddle(msg, &foundMsgMatch)
		p.receiveVinEnd(msg, &foundMsgMatch)
		p.receivePlugState(msg, &foundMsgMatch)
		p.isPrimaryTWC(msg, &foundMsgMatch)
	}
}

// helper to generate the states for all connected TWCs
func (p *TWCPrimary) getStats() []TWCSecondary {
	allSecondaries := []TWCSecondary{}
	for _, twc := range p.knownTWCs {
		allSecondaries = append(allSecondaries, *twc)
	}
	return allSecondaries
}

func (p *TWCPrimary) getTWCStats(twcid []byte) TWCSecondary {
	twc := &TWCSecondary{}
	twc, ok := p.GetSecondary(twcid)
	if ok {
		return *twc
	}
	return *twc
}

// CustomMessage send a msg directly to the serial port, will pad it to the required length and add any padding
// Be careful
func (p *TWCPrimary) CustomMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	msg := vars["msg"]
	data, _ := hex.DecodeString(msg)
	padBytes(&data)
	_, _ = SendMessage(p.DebugLevel, p.port, data)
	fmt.Fprintln(w, "")
}

// SetMaxAmpsHandler is the actual function that sets the maximum amps that all wall connectors can use
func (p *TWCPrimary) SetMaxAmpsHandler(intAmps int) error {
	totalAmps := intAmps * 100 // multiply by 100 to get use in the byte value
	if totalAmps > (p.WiringMaxAmpsAllTWC * 100) {
		// if the given amps is more than the number of available amps, then set total amps to max available
		totalAmps = p.WiringMaxAmpsAllTWC * 100
	}
	if totalAmps == 0 {
		p.AvailableAmps = 0
	} else {
		p.AvailableAmps = totalAmps / 100
	}
	err := p.writeConfig()
	if err != nil {
		return err
	}
	if p.knownTWCs != nil {
		splitAmps := totalAmps / len(p.knownTWCs)
		for _, twc := range p.knownTWCs {
			// set the twc to have the number of amps available to it to use in the heartbeat
			twc.AvailableAmps = Dec2Bytes(uint16(splitAmps))
			if twc.AllowCharge {
				_, err := p.sendChargeRate(twc.TWCID, Dec2Bytes(uint16(splitAmps)), byte(0x09))
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
