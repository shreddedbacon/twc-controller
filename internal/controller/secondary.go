package controller

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/tarm/serial"
)

// TWCSecondary .
type TWCSecondary struct {
	port *serial.Port

	TWCID      []byte `json:"ID"`
	MaxAmps    int    `json:"maxAmps"`
	DebugLevel int    `json:"debugLevel"`

	// Protocol 2 TWCs tend to respond to commands sent using protocol 1, so
	// default to that till we know for sure we're talking to protocol 2.
	ProtocolVersion      int `json:"protocolVersion"` // 1
	MinAmpsTWCSupports   int `json:"minAmpsTWC"`      // 6
	primaryHeartbeatData []byte
	TimeLastRx           int64 `json:"timeLastRX"`

	reportedAmpsActualSignificantChangeMonitor []byte
	timeReportedAmpsActualChangedSignificantly int64

	wiringMaxAmps int    // wiringMaxAmpsPerTWC
	AvailableAmps []byte // the number of amps available to this twc

	// reported* vars below are reported to us in heartbeat messages from a Secondary
	// TWC.
	ReportedAmpsMax    []byte `json:"reportedAmpsMax"`
	ReportedAmpsActual []byte `json:"reportedAmpsActual"`
	LastAmpsOffered    []byte `json:"lastAmpsOffered"`
	ReportedState      byte   `json:"reportedState"`
	StatsCurrentWatts  uint32 `json:"currentkWh"` // calculated from the p1-3 voltages * reportedampsactual
	StatsKWH           uint32 `json:"kWh"`
	StatsP1Volts       uint16 `json:"phase1volts"`
	StatsP2Volts       uint16 `json:"phase2volts"`
	StatsP3Volts       uint16 `json:"phase3volts"`
	StatsP1Amps        int    `json:"phase1amps"`
	StatsP2Amps        int    `json:"phase2amps"`
	StatsP3Amps        int    `json:"phase3amps"`
	AllowCharge        bool   `json:"allowCharge"` // stop the primary talking to this secondary
	ChargeState        bool   `json:"chargeState"`
	VINStart           string `json:"vinStart"`
	VINMiddle          string `json:"vinMiddle"`
	VINEnd             string `json:"vinEnd"`
	PlugState          int    `json:"plugState"`
}

// NewTWCSecondary creates a new secondary TWC.
func NewTWCSecondary(newSecondaryID []byte, maxAmps int, port *serial.Port, wiringMaxAmpsPerTWC int, debugLevel int) (*TWCSecondary, error) {
	now := time.Now().UTC().Unix()
	return &TWCSecondary{
		TimeLastRx:         now,
		port:               port,
		TWCID:              newSecondaryID,
		MaxAmps:            maxAmps,
		ProtocolVersion:    1,
		MinAmpsTWCSupports: 6,
		DebugLevel:         debugLevel,
		wiringMaxAmps:      wiringMaxAmpsPerTWC,
		AvailableAmps:      []byte{0x00, 0x00},
		LastAmpsOffered:    []byte{0x00, 0x00},
		ReportedAmpsActual: []byte{0x00, 0x00},
		ReportedAmpsMax:    []byte{0x00, 0x00},
		reportedAmpsActualSignificantChangeMonitor: []byte{0x00, 0x00},
		primaryHeartbeatData:                       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		AllowCharge:                                true, // default all new TWCs to allow charging
	}, nil
}

// ReceiveSecondaryHeartbeat parses the received heartbeat from the secondary TWC
func (t *TWCSecondary) ReceiveSecondaryHeartbeat(heartbeatData []byte) {
	now := time.Now().UTC().Unix()
	t.TimeLastRx = now
	if t.DebugLevel >= 9 {
		log.Println(log2JSONString(LogData{
			Type:     "INFO",
			Source:   "secondary",
			Receiver: fmt.Sprintf("%x", t.TWCID),
			Message:  "Secondary heartbeat received",
		}))
	}

	t.ReportedAmpsMax = []byte{heartbeatData[1], heartbeatData[2]}
	t.ReportedAmpsActual = []byte{heartbeatData[3], heartbeatData[4]}
	t.ReportedState = heartbeatData[0]

	lastOffered := uint16(0)
	if bytes.Compare(t.LastAmpsOffered, []byte{}) != 0 {
		lastOffered = Bytes2Dec2(t.LastAmpsOffered, false)
	}
	if lastOffered < 0 {
		t.LastAmpsOffered = t.ReportedAmpsMax
		if t.DebugLevel >= 9 {
			log.Println(log2JSONString(LogData{
				Type:     "INFO",
				Source:   "secondary",
				Receiver: fmt.Sprintf("%x", t.TWCID),
				Message:  fmt.Sprintf("Secondary was last offered %x, reported maximum %x", t.LastAmpsOffered, t.ReportedAmpsMax),
			}))
		}
	}

	// fmt.Println("ampsMax", Bytes2Dec2(t.reportedAmpsMax), t.reportedAmpsMax)
	// fmt.Println("ampsActual", Bytes2Dec2(t.reportedAmpsActual), t.reportedAmpsActual)
	// fmt.Println("state", t.reportedState)

	reportedSignificant := uint16(0)
	if bytes.Compare(t.reportedAmpsActualSignificantChangeMonitor, []byte{}) != 0 {
		reportedSignificant = Bytes2Dec2(t.reportedAmpsActualSignificantChangeMonitor, false)
	}
	if reportedSignificant < 0 || Bytes2Dec2(t.ReportedAmpsActual, false)-reportedSignificant > 80 {
		if t.DebugLevel >= 9 {
			log.Println(log2JSONString(LogData{
				Type:     "INFO",
				Source:   "secondary",
				Receiver: fmt.Sprintf("%x", t.TWCID),
				Message:  fmt.Sprintf("Secondary reported actual charging amperage of %x", t.ReportedAmpsActual),
			}))
		}
		t.timeReportedAmpsActualChangedSignificantly = now
		t.reportedAmpsActualSignificantChangeMonitor = t.ReportedAmpsActual
	}

	// fmt.Println("timeReported", t.timeReportedAmpsActualChangedSignificantly)
	// fmt.Println("reportAmpsActual", Bytes2Dec2(t.reportedAmpsActualSignificantChangeMonitor), t.reportedAmpsActualSignificantChangeMonitor)

}

func (t *TWCSecondary) sendPrimaryHeartbeat(port *serial.Port, primaryID []byte) (int64, error) {
	if t.AllowCharge {
		if t.DebugLevel >= 9 {
			log.Println(log2JSONString(LogData{
				Type:     "INFO",
				Source:   "secondary",
				Sender:   fmt.Sprintf("%x", primaryID),
				Receiver: fmt.Sprintf("%x", t.TWCID),
				Message:  "Sending hearbeat to secondary TWC",
			}))
		}
		msg := append(append(append([]byte{0xFB, 0xE0}, primaryID...), t.TWCID...), t.primaryHeartbeatData...)
		// send heartbeat with the available amperage to this twc
		// msg := append(
		// 	append(
		// 		append(
		// 			[]byte{0xFB, 0xE0},
		// 			primaryID...),
		// 		t.TWCID...),
		// 	[]byte{0x05, t.AvailableAmps[1], t.AvailableAmps[0]}...)
		// padBytes(&msg)
		return SendMessage(t.DebugLevel, port, msg)
	}
	return time.Now().UTC().Unix(), nil
}
