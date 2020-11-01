package controller

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/tarm/serial"
)

// SendMessage sends a message to the serial port
func SendMessage(debugLevel int, port *serial.Port, msg []byte) (int64, error) {
	// calculate checksum of message
	checksum := 0
	for b := 1; b < len(msg)-1; b++ {
		checksum = checksum + int(msg[b])
	}
	// add the checksum
	msg = append(msg, byte(checksum&0xFF))

	// escape special chars
	msg = bytes.Replace(bytes.Replace(msg, []byte{0xDB}, []byte{0xDB, 0xDD}, 1), []byte{0xC0}, []byte{0xDB, 0xDC}, 1)

	var debugBytes []byte
	for _, debugB := range msg {
		debubByte := []byte(fmt.Sprintf("%X ", debugB))
		debugBytes = append(debugBytes, debubByte...)
	}
	if debugLevel >= 1 {
		log.Println(log2JSONString(LogData{
			Type:    "DEBUG",
			Source:  "primary",
			Message: fmt.Sprintf("Tx@: %s", debugBytes),
		}))
	}

	// wrap the message with c0
	msg = append([]byte{0xC0}, append(msg, 0xC0)...)

	// actually send the message to the serial port
	_, err := port.Write(msg)
	if err != nil {
		return time.Now().Unix(), err
	}
	return time.Now().Unix(), nil
}

// Dec2Bytes convert uint16 to bytes
func Dec2Bytes(i uint16) []byte {
	bs := make([]byte, 2)
	binary.LittleEndian.PutUint16(bs, i)
	return bs
}

// Bytes2Dec4 convert a byte array to uint32
func Bytes2Dec4(mySlice []byte, little bool) uint32 {
	if little {
		data := binary.LittleEndian.Uint32(mySlice)
		return data
	}
	data := binary.BigEndian.Uint32(mySlice)
	return data
}

// Bytes2Dec2 convert a byte array to uint16
func Bytes2Dec2(mySlice []byte, little bool) uint16 {
	if little {
		data := binary.LittleEndian.Uint16(mySlice)
		return data
	}
	data := binary.BigEndian.Uint16(mySlice)
	return data
}

func unescapeMessage(msg []byte, msgLen int) []byte {
	unescapedMsg := msg[0:msgLen]
	unescapedMsg = bytes.Replace(msg, []byte{0xDB, 0xDC}, []byte{0xC0}, 1)
	return unescapedMsg[1 : len(unescapedMsg)-1]
}

func wattsToAmps(phases int, voltage int, watts float64) int {
	intAmps := int(math.RoundToEven(watts)) / voltage
	if phases == 3 {
		volts := voltage * 3
		intAmps = int(math.RoundToEven(watts)) / volts
	}
	return intAmps
}

func ampsToWatts(phases int, voltage int, amps int) int {
	watts := amps * voltage
	if phases == 3 {
		volts := voltage * 3
		watts = amps * volts
	}
	return watts
}

func bytesToUint16(t []byte) uint16 {
	return Bytes2Dec2(t, false)
}

func bytesToUint16Divide(t []byte) uint16 {
	return Bytes2Dec2(t, false) / 100
}

func bytesToSTring(t []byte) string {
	return fmt.Sprintf("%x", t)
}

// TWCIDStr2Byte convert a given string representation of the TWCID turn it into a byte array
func TWCIDStr2Byte(s string) ([]byte, error) {
	if len(s) == 4 {
		data, err := hex.DecodeString(s)
		return data, err
	}
	return nil, fmt.Errorf("not valid")
}

func padBytes(msg *[]byte) {
	for i := len(*msg); i < 15; i++ {
		*msg = append(*msg, byte(0x00))
	}
}

func isFloatNegative(f float64) bool {
	return math.Signbit(f)
}

func roundFloat(f float64) float64 {
	return math.RoundToEven(f)
}

var funcMap = template.FuncMap{
	"ToUpper":             strings.ToUpper,
	"ToLower":             strings.ToLower,
	"GetTime":             getTime,
	"GetState":            getState,
	"BytesToString":       bytesToSTring,
	"BytesToUint16":       bytesToUint16,
	"BytesToUint16Divide": bytesToUint16Divide,
	"IsFloatNegative":     isFloatNegative,
	"RoundFloat":          roundFloat,
	"Divide": func(a, b int) int {
		return a / b
	},
}

func getTime(t int64) string {
	unixTimeUTC := time.Unix(t, 0)
	unitTimeInRFC := unixTimeUTC.Format(time.RFC822)
	return unitTimeInRFC
}

func getState(t byte) string {
	if bytes.Compare([]byte{t}, []byte{1}) == 0 {
		return `Charging`
	} else if bytes.Compare([]byte{t}, []byte{2}) == 0 {
		return `Error: Lost comms`
	} else if bytes.Compare([]byte{t}, []byte{3}) == 0 {
		return `Do not charge`
	} else if bytes.Compare([]byte{t}, []byte{4}) == 0 {
		return `Ready to charge`
	} else if bytes.Compare([]byte{t}, []byte{5}) == 0 {
		return `Busy`
	} else if bytes.Compare([]byte{t}, []byte{8}) == 0 {
		return `Preparing to charge`
	} else if bytes.Compare([]byte{t}, []byte{9}) == 0 {
		return `Adjusting charge rate`
	} else if bytes.Compare([]byte{t}, []byte{99}) == 0 {
		return `Disabled by controller`
	}
	return `Not charging`
}

func float64frombytes(bytes []byte) float64 {
	bits := binary.LittleEndian.Uint64(bytes)
	float := math.Float64frombits(bits)
	return float
}

func httpError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprintf(w, "%s", fmt.Sprintf(`{"error":"%v"}`, err))
}

func log2JSONString(log LogData) string {
	b, _ := json.Marshal(log)
	return string(b)
}
