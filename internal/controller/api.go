package controller

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// APISetDebugLevel sets the debug level for all TWCs
func (p *TWCPrimary) APISetDebugLevel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dl := vars["debugLevel"]
	debugLevel, err := strconv.ParseInt(dl, 10, 32)
	if err != nil {
		httpError(w, fmt.Errorf("%v", err))
		return
	}
	p.DebugLevel = int(debugLevel)
	if len(p.knownTWCs) > 0 {
		for _, twc := range p.knownTWCs {
			twc.DebugLevel = int(debugLevel)
		}
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"DebugLevel": %v}`, vars["DebugLevel"])
}

// APISetMaxAmpsHandler handles setting the max charge available to all TWCs
func (p *TWCPrimary) APISetMaxAmpsHandler(w http.ResponseWriter, r *http.Request) {
	var amps string
	if r.Method == http.MethodPost {
		amps = r.FormValue("availableAmps")
		err := r.ParseForm()
		if err != nil {
			httpError(w, fmt.Errorf("%v", err))
			return
		}
	} else {
		vars := mux.Vars(r)
		amps = vars["availableAmps"]
	}
	if amps != "" {
		intAmps, err := strconv.ParseFloat(amps, 10)
		if err != nil {
			httpError(w, fmt.Errorf("%v", err))
			return
		}
		err = p.SetMaxAmpsHandler(int(intAmps))
		if err != nil {
			httpError(w, fmt.Errorf("%v", err))
			return
		}
	}
	if r.Method == http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"maxAmps": %v}`, amps)
}

// APISetMaxWattsHandler handles setting the max charge available to all TWCs
func (p *TWCPrimary) APISetMaxWattsHandler(w http.ResponseWriter, r *http.Request) {
	var watts string
	var intAmps int
	if r.Method == http.MethodPost {
		watts = r.FormValue("availableWatts")
		err := r.ParseForm()
		if err != nil {
			httpError(w, fmt.Errorf("%v", err))
			return
		}
	} else {
		vars := mux.Vars(r)
		watts = vars["availableWatts"]
	}
	if watts != "" {
		intWatts, err := strconv.ParseFloat(watts, 10)
		if err != nil {
			httpError(w, fmt.Errorf("%v", err))
			return
		}
		phases := p.SupplyPhases
		if phases != 1 && phases != 3 {
			httpError(w, fmt.Errorf(`{"error":"not valid number of phases: %d"}`, phases))
			return
		}
		intAmps = int(intWatts) / p.SupplyVoltage
		if phases == 3 {
			volts := p.SupplyVoltage * 3
			intAmps = int(intWatts) / volts
		}
		err = p.SetMaxAmpsHandler(int(intAmps))
		if err != nil {
			httpError(w, fmt.Errorf("%v", err))
			return
		}
	}
	if r.Method == http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"maxAmps": %v}`, intAmps)
}

// APIDisableTWC handles disabling a specific TWC
func (p *TWCPrimary) APIDisableTWC(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		twcid := r.FormValue("twcid")
		bTWCID, err := TWCIDStr2Byte(twcid)
		if err != nil {
			httpError(w, fmt.Errorf("%v", err))
			return
		}
		twc, ok := p.GetSecondary(bTWCID)
		if ok {
			if p.DebugLevel >= 9 {
				log.Println(fmt.Sprintf("API: Disabling secondary TWC %x%x ", bTWCID[0], bTWCID[1]))
			}
			twc.AllowCharge = false
			// _, err = p.sendChargeRate(twc.TWCID, []byte{0x00, 0x00}, byte(0x05))
			// if err != nil {
			// 	httpError(w, fmt.Errorf("%v", err))
			// 	return
			// }
			_, err := p.sendStopCommand(twc.TWCID)
			if err != nil {
				httpError(w, fmt.Errorf("%v", err))
				return
			}
			// @TODO: redistribute available amps over available TWCs
		}
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// APIEnableTWC handles disabling a specific TWC
func (p *TWCPrimary) APIEnableTWC(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		twcid := r.FormValue("twcid")
		bTWCID, err := TWCIDStr2Byte(twcid)
		if err != nil {
			httpError(w, fmt.Errorf("%v", err))
			return
		}
		twc, ok := p.GetSecondary(bTWCID)
		if ok {
			if p.DebugLevel >= 9 {
				log.Println(fmt.Sprintf("API: Enabling secondary TWC %x%x ", bTWCID[0], bTWCID[1]))
			}
			twc.AllowCharge = true
			// redistribute available amps over available TWCs
			if p.knownTWCs != nil {
				splitAmps := p.AvailableAmps / len(p.knownTWCs)
				for _, twc := range p.knownTWCs {
					if twc.AllowCharge {
						p.numInitMsgsToSend = 10
						// for i := 0; i < 6; i++ {
						// 	_, _ = p.sendPrimaryLinkReady1()
						// 	time.Sleep(100 * time.Millisecond)
						// }
						// for i := 0; i < 6; i++ {
						// 	_, _ = p.sendPrimaryLinkReady2()
						// 	time.Sleep(100 * time.Millisecond)
						// }
						_, err := p.sendChargeRate(twc.TWCID, Dec2Bytes(uint16(splitAmps*100)), byte(0x09))
						if err != nil {
							httpError(w, fmt.Errorf("%v", err))
							return
						}
						_, err = p.sendStartCommand(twc.TWCID)
						if err != nil {
							httpError(w, fmt.Errorf("%v", err))
							return
						}
					}
				}
			}
		}
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// APIGetStats just marshal the TWCs that we know about into json and return it
func (p *TWCPrimary) APIGetStats(w http.ResponseWriter, r *http.Request) {
	strB, err := json.Marshal(p.getStats())
	if err != nil {
		httpError(w, fmt.Errorf("%v", err))
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%s", string(strB))
}

// APIPollVIN gets the vin number from the TWC.
func (p *TWCPrimary) APIPollVIN(w http.ResponseWriter, r *http.Request) {
	_, _ = p.PollVINStart()
	_, _ = p.PollVINMiddle()
	_, _ = p.PollVINEnd()
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok"}`)
}

// APIPollPlugState gets the plug state from the TWC.
func (p *TWCPrimary) APIPollPlugState(w http.ResponseWriter, r *http.Request) {
	_, _ = p.PollPlugState()
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok"}`)
}

// APIPollFirmwareInfo gets the firmware info from the TWC.
func (p *TWCPrimary) APIPollFirmwareInfo(w http.ResponseWriter, r *http.Request) {
	_, _ = p.PollFirmwareVersion()
	_, _ = p.PollModel()
	_, _ = p.PollSerialNumber()
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok"}`)
}
