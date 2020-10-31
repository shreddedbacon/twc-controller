package controller

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"gopkg.in/matryer/try.v1"
)

// APIStopCharging stop charging
func (p *TWCPrimary) APIStopCharging(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		twcid := r.FormValue("twcid")
		bTWCID, err := TWCIDStr2Byte(twcid)
		if err != nil {
			httpError(w, fmt.Errorf("%v", err))
			return
		}
		err = p.StopCharging(bTWCID)
		if err != nil {
			httpError(w, fmt.Errorf("%v", err))
			return
		}
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// APIStartCharging stop charging
func (p *TWCPrimary) APIStartCharging(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		twcid := r.FormValue("twcid")
		bTWCID, err := TWCIDStr2Byte(twcid)
		if err != nil {
			httpError(w, fmt.Errorf("%v", err))
			return
		}
		err = p.StartCharging(bTWCID)
		if err != nil {
			httpError(w, fmt.Errorf("%v", err))
			return
		}
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// StopCharging attempts to stop charging the car
func (p *TWCPrimary) StopCharging(TWCID []byte) error {
	twc, ok := p.GetSecondary(TWCID)
	if ok {
		// if the twc is reporting that a car is plugged in, then attempt to stop charging it
		if twc.PlugState == 1 || twc.PlugState == 3 {
			if p.DebugLevel >= 9 {
				log.Println(log2JSONString(LogData{
					Type:     "DEBUG",
					Source:   "charging",
					Receiver: fmt.Sprintf("%x", TWCID),
					Message:  "Received stop command for TWC",
				}))
			}
			if len(p.TeslaAPITokens) > 0 {
				vin := fmt.Sprintf("%s%s%s", twc.VINStart, twc.VINMiddle, twc.VINEnd)
				if len(vin) == 17 {
					twc.AllowCharge = true
					twc.ChargeState = false
					if p.DebugLevel >= 9 {
						log.Println(log2JSONString(LogData{
							Type:     "DEBUG",
							Source:   "charging",
							Receiver: fmt.Sprintf("%x", TWCID),
							Message:  fmt.Sprintf("Attempting to stop charging car via API, vin: %s ", vin),
						}))
					}
					// try and stop charging 10 times before giving up
					err := try.Do(func(attempt int) (bool, error) {
						var err error
						err = p.TeslaAPIChargeByVIN(vin, false)
						if err != nil {
							if p.DebugLevel >= 9 {
								log.Println(log2JSONString(LogData{
									Type:     "ERROR",
									Source:   "charging",
									Receiver: fmt.Sprintf("%x", TWCID),
									Message:  fmt.Sprintf("Unable to stop charging vin %s, trying again: %v", vin, err),
								}))
							}
							time.Sleep(2 * time.Second)
						}
						return attempt < 10, err
					})
					if err != nil {
						return err
					}
				}
			} else {
				if p.DebugLevel >= 9 {
					log.Println(log2JSONString(LogData{
						Type:     "DEBUG",
						Source:   "charging",
						Receiver: fmt.Sprintf("%x", TWCID),
						Message:  "No accounts known by controller, proceeding to stop charging the brutal way",
					}))
				}
				twc.AllowCharge = false
				twc.ChargeState = false
				if p.DebugLevel >= 9 {
					log.Println(log2JSONString(LogData{
						Type:     "DEBUG",
						Source:   "charging",
						Receiver: fmt.Sprintf("%x", TWCID),
						Message:  "Disabling secondary TWC",
					}))
				}
				_, err := p.sendChargeRate(twc.TWCID, []byte{0x00, 0x00}, byte(0x05))
				if err != nil {
					return err
				}
				_, err = p.sendStopCommand(twc.TWCID)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// StartCharging attempts to start charging the car
func (p *TWCPrimary) StartCharging(TWCID []byte) error {
	twc, ok := p.GetSecondary(TWCID)
	if ok {
		// if the twc is reporting that a car is plugged in, then attempt to start charging it
		if twc.PlugState == 1 || twc.PlugState == 3 {
			if p.DebugLevel >= 9 {
				log.Println(log2JSONString(LogData{
					Type:     "DEBUG",
					Source:   "charging",
					Receiver: fmt.Sprintf("%x", TWCID),
					Message:  "Received start command for TWC",
				}))
			}
			if len(p.TeslaAPITokens) > 0 {
				vin := fmt.Sprintf("%s%s%s", twc.VINStart, twc.VINMiddle, twc.VINEnd)
				if len(vin) == 17 {
					if p.DebugLevel >= 9 {
						log.Println(log2JSONString(LogData{
							Type:     "DEBUG",
							Source:   "charging",
							Receiver: fmt.Sprintf("%x", TWCID),
							Message:  fmt.Sprintf("Attempting to start charging car via API, vin: %s ", vin),
						}))
					}
					splitAmps := p.AvailableAmps / len(p.knownTWCs)
					twc.AllowCharge = true
					twc.ChargeState = true
					// try and start charging 10 times before giving up
					err := try.Do(func(attempt int) (bool, error) {
						var err error
						err = p.TeslaAPIChargeByVIN(vin, true)
						if err != nil {
							if p.DebugLevel >= 9 {
								log.Println(log2JSONString(LogData{
									Type:     "ERROR",
									Source:   "charging",
									Receiver: fmt.Sprintf("%x", TWCID),
									Message:  fmt.Sprintf("Unable to start charging vin %s, trying again: %v", vin, err),
								}))
							}
							time.Sleep(2 * time.Second)
						}
						return attempt < 10, err
					})
					if err != nil {
						return err
					}
					_, err = p.sendChargeRate(twc.TWCID, Dec2Bytes(uint16(splitAmps*100)), byte(0x09))
					if err != nil {
						return err
					}
				}
			} else {
				if p.DebugLevel >= 9 {
					log.Println(log2JSONString(LogData{
						Type:     "DEBUG",
						Source:   "charging",
						Receiver: fmt.Sprintf("%x", TWCID),
						Message:  "No accounts known by controller, proceeding to start charging the brutal way",
					}))
				}
				splitAmps := p.AvailableAmps / len(p.knownTWCs)
				twc.AllowCharge = true
				twc.ChargeState = true
				if p.DebugLevel >= 9 {
					log.Println(log2JSONString(LogData{
						Type:     "DEBUG",
						Source:   "charging",
						Receiver: fmt.Sprintf("%x", TWCID),
						Message:  "Enabling secondary TWC",
					}))
				}
				_, err := p.sendStartCommand(twc.TWCID)
				if err != nil {
					return err
				}
				_, err = p.sendChargeRate(twc.TWCID, Dec2Bytes(uint16(splitAmps*100)), byte(0x09))
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// @TODO: need an option to check if we should actually stop/start connected cars, to support scheduled charging/departure in the car

// StopConnectedCars loop over all the twcs that have a known charge state and check if they need to be stopped
func (p *TWCPrimary) StopConnectedCars() {
	if p.knownTWCs != nil {
		for _, twc := range p.knownTWCs {
			if twc.ChargeState == true {
				err := p.StopCharging(twc.TWCID)
				if err != nil {
					fmt.Println(fmt.Sprintf("error stopping charging: %v", err))
				}
			}
		}
	}
}

// StartConnectedCars loop over all the twcs that have a known charge state and check if they need to be started
func (p *TWCPrimary) StartConnectedCars() {
	if p.knownTWCs != nil {
		for _, twc := range p.knownTWCs {
			if twc.ChargeState == false {
				err := p.StartCharging(twc.TWCID)
				if err != nil {
					fmt.Println(fmt.Sprintf("error starting charging: %v", err))
				}
			}
		}
	}
}
