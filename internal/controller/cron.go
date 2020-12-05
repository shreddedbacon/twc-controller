package controller

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	powerwall "github.com/shreddedbacon/fake-powerwall/api"
)

// RunCron runs the cron scrips
func (p *TWCPrimary) RunCron() {
	now := time.Now().UTC().Unix()
	p.vinCron(now)
	p.twcStatusCron(now)
	p.powerwallCron(now)
}

func (p *TWCPrimary) twcStatusCron(now int64) {
	if (now - p.timeLastSecondaryPoll) >= 5 {
		if p.DebugLevel >= 12 {
			log.Println(log2JSONString(LogData{
				Type:    "DEBUG",
				Source:  "cron",
				Message: fmt.Sprintf("Running twcStatusCron %d, knownTWC count %d", now-p.timeLastSecondaryPoll, len(p.knownTWCs)),
			}))
		}
		if len(p.knownTWCs) == 1 {
			p.SetTWCStatusLED(0x00ff00)
		} else if len(p.knownTWCs) == 0 {
			p.SetTWCStatusLED(0xffff00)
		}
		p.timeLastSecondaryPoll = now
	}
}

// check if the full vin is collected and set the VIN LED status
// this is meant for if the controller only talks to 1 twc
func (p *TWCPrimary) vinCron(now int64) {
	if (now - p.timeLastVINCron) >= 5 {
		if p.DebugLevel >= 12 {
			log.Println(log2JSONString(LogData{
				Type:    "DEBUG",
				Source:  "cron",
				Message: fmt.Sprintf("Running vin %d, knownTWC count %d", now-p.timeLastSecondaryPoll, len(p.knownTWCs)),
			}))
		}
		if len(p.knownTWCs) == 1 {
			if p.knownTWCs[0].VINStart == "" && p.knownTWCs[0].VINMiddle == "" && p.knownTWCs[0].VINEnd == "" {
				p.SetVINLED(0x000000)
			}

			if p.knownTWCs[0].VINStart != "" && p.knownTWCs[0].VINMiddle == "" && p.knownTWCs[0].VINEnd == "" {
				p.SetVINLED(0xff0000)
			}
			if p.knownTWCs[0].VINStart == "" && p.knownTWCs[0].VINMiddle != "" && p.knownTWCs[0].VINEnd == "" {
				p.SetVINLED(0xff0000)
			}
			if p.knownTWCs[0].VINStart == "" && p.knownTWCs[0].VINMiddle == "" && p.knownTWCs[0].VINEnd != "" {
				p.SetVINLED(0xff0000)
			}

			if p.knownTWCs[0].VINStart != "" && p.knownTWCs[0].VINMiddle != "" && p.knownTWCs[0].VINEnd == "" {
				p.SetVINLED(0xffff00)
			}
			if p.knownTWCs[0].VINStart != "" && p.knownTWCs[0].VINMiddle == "" && p.knownTWCs[0].VINEnd != "" {
				p.SetVINLED(0xffff00)
			}
			if p.knownTWCs[0].VINStart == "" && p.knownTWCs[0].VINMiddle != "" && p.knownTWCs[0].VINEnd != "" {
				p.SetVINLED(0xffff00)
			}

			if p.knownTWCs[0].VINStart != "" && p.knownTWCs[0].VINMiddle != "" && p.knownTWCs[0].VINEnd != "" {
				p.SetVINLED(0x00ff00)
			}
		}
		p.timeLastVINCron = now
	}
}

// this is where we check the usage from the solar/powerwall and set the available amperage on the wall connector
func (p *TWCPrimary) powerwallCron(now int64) {
	// check the last time we checked the powerwall for its status
	if (now - p.timeLastPowerwallCheck) >= int64(p.PowerwallCheckInterval*60) {
		// if powerwall monitoring is enabled
		if p.EnablePowerwall && p.Powerwall != "" {
			// if the time checks out, then we do the thing
			if p.DebugLevel >= 12 {
				log.Println(log2JSONString(LogData{
					Type:    "DEBUG",
					Source:  "cron",
					Message: fmt.Sprintf("Running powerwallCron %d", now-p.timeLastPowerwallCheck),
				}))
			}

			d := &powerwall.MetersAggregates{}
			pw := powerwall.FakePowerwall{
				Inverter: p.Powerwall,
			}
			b, err := pw.Request("/api/meters/aggregates")
			if err == nil {
				json.Unmarshal(b, d)
			}

			// get the total watts the chargers are consuming first up
			totalWatts := 0
			for _, twc := range p.knownTWCs {
				totalWatts = totalWatts + int(twc.StatsCurrentWatts)
			}
			currentLoad := 0
			if d.Load != nil {
				currentLoad = int(d.Load.InstantPower)
			}
			nonChargerLoad := int(currentLoad)
			if int(currentLoad) > totalWatts {
				nonChargerLoad = int(currentLoad) - totalWatts
			}

			solarGeneration := float64(0)
			if d.Solar != nil {
				solarGeneration = d.Solar.InstantPower
			}
			availableWatts := float64(0)
			if int(solarGeneration) > nonChargerLoad {
				availableWatts = float64(int(solarGeneration) - nonChargerLoad)
			}

			// need to invert the site power as it shows as negative when exporting to grid
			// solarExport := d.Site.InstantPower * -1
			intAmps := wattsToAmps(p.SupplyPhases, p.SupplyVoltage, availableWatts)
			offsetAmps := wattsToAmps(p.SupplyPhases, p.SupplyVoltage, float64(p.PowerOffset))
			// @TODO: this needs to be redone at some stage to calculate if the powerwall/solar power value has the charger usage in it too
			// if intAmps >= p.MinAmpsPerTWC {
			// 	// check if the power we are getting from solar is producing enough amps
			// 	if p.DebugLevel >= 12 {
			// 		log.Println(fmt.Sprintf("set the amperage to %d", intAmps))
			// 	}
			// 	err = p.SetMaxAmpsHandler(intAmps)
			// 	if err != nil {
			// 		if p.DebugLevel >= 12 {
			// 			log.Println(fmt.Sprintf("error set the amperage to %d", intAmps))
			// 		}
			// 	}
			// 	if p.AutoStartStopInterval {
			// 		p.StartConnectedCars()
			// 	}
			// } else
			if intAmps+offsetAmps >= p.MinAmpsPerTWC {
				// check if the power we are getting from solar is producing enough amps, plus the amps we have as an offset
				if p.DebugLevel == 12 {
					log.Println(log2JSONString(LogData{
						Type:    "DEBUG",
						Source:  "cron",
						Message: fmt.Sprintf("Setting the amperage to %d, including offset of %d", intAmps+offsetAmps, offsetAmps),
					}))
				}
				if p.AutoStartStopInterval {
					err = p.SetMaxAmpsHandler(intAmps + offsetAmps)
					if err != nil {
						if p.DebugLevel >= 12 {
							log.Println(log2JSONString(LogData{
								Type:    "ERROR",
								Source:  "cron",
								Message: fmt.Sprintf("Error setting the amperage to %d, including offset of %d", intAmps+offsetAmps, offsetAmps),
							}))
						}
						return
					}
				}
				if p.AutoStartStopInterval {
					p.StartConnectedCars()
				}
			} else {
				// otherwise fall back to just our offset amps
				if offsetAmps >= p.MinAmpsPerTWC {
					if p.DebugLevel >= 12 {
						log.Println(log2JSONString(LogData{
							Type:    "DEBUG",
							Source:  "cron",
							Message: fmt.Sprintf("Setting the amperage to %d with no solar generation", offsetAmps),
						}))
					}
					if p.AutoStartStopInterval {
						err = p.SetMaxAmpsHandler(offsetAmps)
						if err != nil {
							if p.DebugLevel >= 12 {
								log.Println(log2JSONString(LogData{
									Type:    "ERROR",
									Source:  "cron",
									Message: fmt.Sprintf("Error setting the amperage to %d with no solar generation", offsetAmps),
								}))
							}
							return
						}
					}
					if p.AutoStartStopInterval {
						p.StartConnectedCars()
					}
				} else {
					if p.DebugLevel >= 12 {
						log.Println(log2JSONString(LogData{
							Type:    "DEBUG",
							Source:  "cron",
							Message: fmt.Sprintf("Not enough amps to cover minimum; Offset: %d, Minimum:%d", offsetAmps, p.MinAmpsPerTWC),
						}))
					}
					if p.AutoStartStopInterval {
						p.StopConnectedCars()
					}
				}
			}
		} else {
			// if powerwall monitoring is disabled, then just check if the available amps are enough
			// this mode is basically acting just like a normal wall connector if the available amps are higher than the minimum (default 6A)
			if p.AvailableAmps >= p.MinAmpsPerTWC {
				if p.DebugLevel >= 12 {
					log.Println(log2JSONString(LogData{
						Type:    "DEBUG",
						Source:  "cron",
						Message: fmt.Sprintf("Setting the amperage to %d", p.AvailableAmps),
					}))
				}
				if p.AutoStartStopInterval {
					err := p.SetMaxAmpsHandler(p.AvailableAmps)
					if err != nil {
						if p.DebugLevel >= 12 {
							log.Println(log2JSONString(LogData{
								Type:    "ERROR",
								Source:  "cron",
								Message: fmt.Sprintf("Error setting the amperage to %d", p.AvailableAmps),
							}))
						}
						return
					}
				}
				if p.AutoStartStopInterval {
					p.StartConnectedCars()
				}
			} else {
				if p.DebugLevel >= 12 {
					log.Println(log2JSONString(LogData{
						Type:    "DEBUG",
						Source:  "cron",
						Message: fmt.Sprintf("Not enough amps to cover minimum; Available: %d, Minimum:%d", p.AvailableAmps, p.MinAmpsPerTWC),
					}))
				}
				if p.AutoStartStopInterval {
					p.StopConnectedCars()
				}
			}
		}
		p.timeLastPowerwallCheck = now
	}
}
