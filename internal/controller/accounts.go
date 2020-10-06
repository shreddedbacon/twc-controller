package controller

import (
	"fmt"
	"log"
	"net/http"
	"text/template"
	"time"

	"github.com/shreddedbacon/tesla"
	"github.com/shreddedbacon/twcmanager/internal/ui"
)

// TeslaAPIAuth Authenticates against the telsa API to retrieve a token.
func (p *TWCPrimary) TeslaAPIAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		tUser := r.FormValue("username")
		tPass := r.FormValue("password")

		tClient, err := tesla.NewClient(
			&tesla.Auth{
				ClientID:     "81527cff06843c8634fdc09e8ac0abefb46ac849f38fe1e431c2ef2106796384",
				ClientSecret: "c7257eb71a564034f9419ee651c7d0e5f7aa6bfbd18bafb5c5c033b093bb2fa3",
				Email:        tUser,
				Password:     tPass,
			})
		if err != nil {
			http.Redirect(w, r, "/accounts", http.StatusSeeOther)
			log.Println(fmt.Sprintf("Error authenticating user: %v", err))
			return
		}
		tAPI := &TeslaAPIUser{
			Username: tUser,
			Token:    tClient.Token,
		}
		// check if we already have the user, and just update the token
		idx, ok := containsAPIUser(p.TeslaAPITokens, tAPI)
		if ok {
			p.TeslaAPITokens[idx] = tAPI
		} else {
			p.TeslaAPITokens = append(p.TeslaAPITokens, tAPI)
		}
	}
	http.Redirect(w, r, "/accounts", http.StatusSeeOther)
}

func containsAPIUser(slice []*TeslaAPIUser, s *TeslaAPIUser) (int, bool) {
	for idx, item := range slice {
		if item.Username == s.Username {
			return idx, true
		}
	}
	return 0, false
}

// TeslaAPIPage .
type TeslaAPIPage struct {
	PageName    string
	BreadCrumbs []BreadCrumb
	PageData    TWCPrimary
}

// GetTeslaAPIUsers displays
func (p *TWCPrimary) GetTeslaAPIUsers(w http.ResponseWriter, r *http.Request) {
	pageData := TeslaAPIPage{
		BreadCrumbs: getBreadCrumbs("Accounts"),
		PageName:    "Accounts",
		PageData:    *p,
	}
	tpl1, _ := ui.Asset("templates/accounts.html")
	tpl2, _ := ui.Asset("templates/home.html")
	tpl3, _ := ui.Asset("templates/base.html")
	tpl := append(tpl1, tpl2...)
	tpl = append(tpl, tpl3...)
	tmpl, _ := template.New("").Funcs(funcMap).Parse(string(tpl))
	tmpl.ExecuteTemplate(w, "base", pageData)
}

// TeslaAPIChargeByVIN check if the known tesla API users have the VIN that is being returned by the TWC,
// return the user for next steps
func (p *TWCPrimary) TeslaAPIChargeByVIN(VIN string, charge bool) error {
	for _, tAPIUser := range p.TeslaAPITokens {
		tClient, err := tesla.NewClientWithToken(&tesla.Auth{
			ClientID:     "81527cff06843c8634fdc09e8ac0abefb46ac849f38fe1e431c2ef2106796384",
			ClientSecret: "c7257eb71a564034f9419ee651c7d0e5f7aa6bfbd18bafb5c5c033b093bb2fa3",
		}, tAPIUser.Token)
		if err != nil {
			return fmt.Errorf("Error authenticating user: %v", err)
		}
		vehicles, err := tClient.Vehicles()
		if err != nil {
			return fmt.Errorf("Error authenticating user: %v", err)
		}
		for _, v := range vehicles {
			if v.Vin == VIN {
				v, err := v.Wakeup()
				if err != nil {
					return fmt.Errorf("Error waking car: %v", err)
				}
				if v.State != "online" {
					err = fmt.Errorf("Car not online yet: %v", err)
					time.Sleep(5 * time.Second)
				}
				if err != nil {
					return fmt.Errorf("Error waking car: %v", err)
				}
				if charge {
					err := v.StartCharging()
					if err != nil {
						return fmt.Errorf("Error starting charge: %v", err)
					}

				} else {
					err := v.StopCharging()
					if err != nil {
						return fmt.Errorf("Error stopping charge: %v", err)
					}
				}
			}
		}
	}
	return nil
}

// ListTeslaAPIVehicles lists all the vehicles for all known user accounts
func (p *TWCPrimary) ListTeslaAPIVehicles(w http.ResponseWriter, r *http.Request) {
	for _, tAPIUser := range p.TeslaAPITokens {
		tClient, err := tesla.NewClientWithToken(&tesla.Auth{
			ClientID:     "81527cff06843c8634fdc09e8ac0abefb46ac849f38fe1e431c2ef2106796384",
			ClientSecret: "c7257eb71a564034f9419ee651c7d0e5f7aa6bfbd18bafb5c5c033b093bb2fa3",
		}, tAPIUser.Token)
		if err != nil {
			http.Redirect(w, r, "/vehicles", http.StatusSeeOther)
			log.Println(fmt.Sprintf("Error authenticating user: %v", err))
			return
		}
		vehicles, err := tClient.Vehicles()
		if err != nil {
			http.Redirect(w, r, "/vehicles", http.StatusSeeOther)
			log.Println(fmt.Sprintf("Error authenticating user: %v", err))
			return
		}
		for _, v := range vehicles {
			fmt.Println(fmt.Sprintf("%v, %v, %v, %v", v.ID, v.DisplayName, v.State, v.Vin))
		}
	}
	http.Redirect(w, r, "/vehicles", http.StatusSeeOther)
}
