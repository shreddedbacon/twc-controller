package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"text/template"

	"github.com/shreddedbacon/twcmanager/internal/ui"
)

// APISettings returns the settings for the TWC controller
func (p *TWCPrimary) APISettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		wiringMaxAmpsAllTWC := r.FormValue("wiringMaxAmpsAllTWC")
		wiringMaxAmpsPerTWC := r.FormValue("wiringMaxAmpsPerTWC")
		minAmpsPerTWC := r.FormValue("minAmpsPerTWC")
		supplyVoltage := r.FormValue("supplyVoltage")
		supplyPhases := r.FormValue("supplyPhases")
		debugLevel := r.FormValue("debugLevel")
		baudRate := r.FormValue("baudRate")
		devicePath := r.FormValue("devicePath")
		enableLed := r.FormValue("enableLed")
		err := r.ParseForm()
		if err != nil {
			httpError(w, fmt.Errorf("%v", err))
			return
		}
		if wiringMaxAmpsAllTWC != "" && wiringMaxAmpsPerTWC != "" && debugLevel != "" && supplyPhases != "" && supplyVoltage != "" {
			wmaat, err := strconv.Atoi(wiringMaxAmpsAllTWC)
			if err != nil {
				httpError(w, fmt.Errorf(`{"error":"wiring max amps all twc is not a number: %v"}`, err))
				return
			}
			wmapt, err := strconv.Atoi(wiringMaxAmpsPerTWC)
			if err != nil {
				httpError(w, fmt.Errorf(`{"error":"wiring max amps per twc is not a number: %v"}`, err))
				return
			}
			mapt, err := strconv.Atoi(minAmpsPerTWC)
			if err != nil {
				httpError(w, fmt.Errorf(`{"error":"min amps per twc is not a number: %v"}`, err))
				return
			}
			sv, err := strconv.Atoi(supplyVoltage)
			if err != nil {
				httpError(w, fmt.Errorf(`{"error":"supply voltage is not a number: %v"}`, err))
				return
			}
			sp, err := strconv.Atoi(supplyPhases)
			if err != nil {
				httpError(w, fmt.Errorf(`{"error":"supply phase is not a number: %v"}`, err))
				return
			}
			dl, err := strconv.Atoi(debugLevel)
			if err != nil {
				httpError(w, fmt.Errorf(`{"error":"debug level is not a number: %v"}`, err))
				return
			}
			p.WiringMaxAmpsAllTWC = wmaat
			p.WiringMaxAmpsPerTWC = wmapt
			p.MinAmpsPerTWC = mapt
			p.DebugLevel = dl
			for _, twc := range p.knownTWCs {
				twc.DebugLevel = dl
			}
			if sv >= 100 && sv <= 260 {
				// only set the supply voltage is between 100 and 260 is defined
				p.SupplyVoltage = sv
			}
			if sp == 1 || sp == 3 {
				// only set the supply phase if 1 or 3 is defined
				p.SupplyPhases = sp
			}
		}
		if baudRate != "" && devicePath != "" {
			br, err := strconv.Atoi(baudRate)
			if err != nil {
				httpError(w, fmt.Errorf(`{"error":"baud rate is not a number: %v"}`, err))
				return
			}
			p.SerialConfig.BaudRate = br
			p.SerialConfig.DevicePath = devicePath
		}
		if enableLed == "on" {
			p.LEDSOn = true
		} else {
			p.LEDSOn = false
		}
		err = p.writeConfig()
		if err != nil {
			httpError(w, err)
			return
		}
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
		return
	} else if r.Method == http.MethodGet {
		strB := fmt.Sprintf(`{"wiringMaxAmpsAllTWC": %d, "wiringMaxAmpsPerTWC": %d, "debugLevel": %d}`, p.WiringMaxAmpsAllTWC, p.WiringMaxAmpsPerTWC, p.DebugLevel)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%s", strB)
		return
	}
	httpError(w, fmt.Errorf(`{"error":"unknown command"}`))
}

// GetPrimarySettings .
func (p *TWCPrimary) GetPrimarySettings(w http.ResponseWriter, r *http.Request) {
	pageData := SettingsPage{
		BreadCrumbs: getBreadCrumbs("Settings"),
		PageName:    "Settings",
		PageData:    *p,
	}
	tpl1, _ := ui.Asset("templates/settings.html")
	tpl2, _ := ui.Asset("templates/home.html")
	tpl3, _ := ui.Asset("templates/base.html")
	tpl := append(tpl1, tpl2...)
	tpl = append(tpl, tpl3...)
	tmpl, _ := template.New("").Funcs(funcMap).Parse(string(tpl))
	tmpl.ExecuteTemplate(w, "base", pageData)
}
