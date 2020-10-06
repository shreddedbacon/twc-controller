package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"text/template"

	powerwall "github.com/shreddedbacon/fake-powerwall/api"
	"github.com/shreddedbacon/twcmanager/internal/ui"
)

// PowerwallSettingsPage .
type PowerwallSettingsPage struct {
	PageName      string
	BreadCrumbs   []BreadCrumb
	PageData      TWCPrimary
	PowerwallInfo powerwall.MetersAggregates
}

// GetPowerwallSettings .
func (p *TWCPrimary) GetPowerwallSettings(w http.ResponseWriter, r *http.Request) {
	tpl1, _ := ui.Asset("templates/powerwall.html")
	tpl2, _ := ui.Asset("templates/home.html")
	tpl3, _ := ui.Asset("templates/base.html")
	tpl := append(tpl1, tpl2...)
	tpl = append(tpl, tpl3...)
	tmpl, _ := template.New("").Funcs(funcMap).Parse(string(tpl))
	d := &powerwall.MetersAggregates{}
	if p.EnablePowerwall && p.Powerwall != "" {
		pw := powerwall.FakePowerwall{
			Inverter: p.Powerwall,
		}
		b, err := pw.Request("/api/meters/aggregates")
		if err == nil {
			json.Unmarshal(b, d)
		}
	}
	pageData := PowerwallSettingsPage{
		BreadCrumbs:   getBreadCrumbs("Powerwall"),
		PageName:      "Powerwall",
		PageData:      *p,
		PowerwallInfo: *d,
	}
	tmpl.ExecuteTemplate(w, "base", pageData)
}

// GetPowerwallSiteUsage get the usage meter value from the Powerwall.
func (p *TWCPrimary) GetPowerwallSiteUsage(w http.ResponseWriter, r *http.Request) {
	if p.EnablePowerwall && p.Powerwall != "" {
		pw := powerwall.FakePowerwall{
			Inverter: p.Powerwall,
		}
		b, err := pw.Request("/api/meters/aggregates")
		if err != nil {
			httpError(w, err)
			return
		}
		fmt.Fprintln(w, fmt.Sprintf(`%s`, b))
		return
	}
	httpError(w, fmt.Errorf("not configured"))
}

// APIPowerwallSettings returns the settings for the TWC controller
func (p *TWCPrimary) APIPowerwallSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		powerwall := r.FormValue("powerwall")
		enablePowerwall := r.FormValue("enablePowerwall")
		autoStartStopInterval := r.FormValue("autoStartStopInterval")
		powerOffset := r.FormValue("powerOffset")
		powerOffsetAmps := r.FormValue("powerOffsetAmps")
		powerwallCheckInterval := r.FormValue("powerwallCheckInterval")
		err := r.ParseForm()
		if err != nil {
			httpError(w, fmt.Errorf("%v", err))
			return
		}
		if powerOffset != "" && powerwallCheckInterval != "" {
			po, err := strconv.Atoi(powerOffset)
			if err != nil {
				httpError(w, fmt.Errorf(`{"error":"power offset (watts) is not a number: %v"}`, err))
				return
			}
			pwci, err := strconv.Atoi(powerwallCheckInterval)
			if err != nil {
				httpError(w, fmt.Errorf(`{"error":"powerwall check interval is not a number: %v"}`, err))
				return
			}
			p.PowerOffset = po
			p.PowerwallCheckInterval = pwci
		}
		if powerOffsetAmps != "" {
			poa, err := strconv.Atoi(powerOffsetAmps)
			if err != nil {
				httpError(w, fmt.Errorf(`{"error":"power offset (amps) is not a number: %v"}`, err))
				return
			}
			p.PowerOffset = ampsToWatts(p.SupplyPhases, p.SupplyVoltage, poa)

		}
		if powerwall != "" {
			p.Powerwall = powerwall
		}
		if autoStartStopInterval == "on" {
			p.AutoStartStopInterval = true
		} else {
			p.AutoStartStopInterval = false
		}
		if enablePowerwall == "on" {
			p.EnablePowerwall = true
		} else {
			p.EnablePowerwall = false
		}
		err = p.writeConfig()
		if err != nil {
			httpError(w, err)
			return
		}
		http.Redirect(w, r, "/powerwall", http.StatusSeeOther)
		return
	} else if r.Method == http.MethodGet {
		strB := fmt.Sprintf(`{"powerwall": "%s", "enablePowerwall": %v}`, p.Powerwall, p.EnablePowerwall)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%s", strB)
		return
	}
	httpError(w, fmt.Errorf("unknown command"))
}
