package controller

import (
	"fmt"
	"net/http"
	"text/template"

	"github.com/gorilla/mux"
	"github.com/shreddedbacon/twcmanager/internal/ui"
)

func getBreadCrumbs(name string) []BreadCrumb {
	for i, bc := range breadcrumbMenu {
		breadcrumbMenu[i].Active = ""
		if bc.Name == name {
			breadcrumbMenu[i].Active = "active"
		}
	}
	return breadcrumbMenu
}

var breadcrumbMenu = []BreadCrumb{
	{
		Name:    "Wall Connectors",
		NavName: "Wall Connectors",
		Path:    "/",
		Active:  "",
	},
	// {
	// 	Name:    "Vehicles",
	// 	NavName: "Vehicles",
	// 	Path:    "/",
	// 	Active:  "",
	// },
	{
		Name:    "Powerwall",
		NavName: "Powerwall",
		Path:    "/powerwall",
		Active:  "",
	},
	{
		Name:    "Accounts",
		NavName: "Accounts",
		Path:    "/accounts",
		Active:  "",
	},
	{
		Name:    "Settings",
		NavName: "Settings",
		Path:    "/settings",
		Active:  "",
	},
}

// StatsPage .
type StatsPage struct {
	PageName    string
	BreadCrumbs []BreadCrumb
	StatsData   []TWCSecondary
	PrimaryData TWCPrimary
}

// SettingsPage .
type SettingsPage struct {
	PageName    string
	BreadCrumbs []BreadCrumb
	PrimaryData TWCPrimary
}

// TWCInfoPage .
type TWCInfoPage struct {
	PageName    string
	BreadCrumbs []BreadCrumb
	StatsData   TWCSecondary
	PrimaryData TWCPrimary
}

// BreadCrumb .
type BreadCrumb struct {
	ID      int
	Name    string
	Active  string
	NavName string
	Path    string
}

// GetWallConnectors .
func (p *TWCPrimary) GetWallConnectors(w http.ResponseWriter, r *http.Request) {
	pageData := StatsPage{
		BreadCrumbs: getBreadCrumbs("Wall Connectors"),
		PageName:    "Wall Connectors",
		StatsData:   p.getStats(),
		PrimaryData: *p,
	}
	tpl1, _ := ui.Asset("templates/stats.html")
	tpl2, _ := ui.Asset("templates/home.html")
	tpl3, _ := ui.Asset("templates/base.html")
	tpl := append(tpl1, tpl2...)
	tpl = append(tpl, tpl3...)
	tmpl, _ := template.New("").Funcs(funcMap).Parse(string(tpl))
	tmpl.ExecuteTemplate(w, "base", pageData)
}

// GetWallConnectorInfo gets a specific TWC info.
func (p *TWCPrimary) GetWallConnectorInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	twcid := vars["twcid"]
	bTWCID, err := TWCIDStr2Byte(twcid)
	if err != nil {
		httpError(w, fmt.Errorf("%v", err))
		return
	}
	pageData := TWCInfoPage{
		BreadCrumbs: getBreadCrumbs("Wall Connectors"),
		PageName:    "Wall Connector",
		StatsData:   p.getTWCStats(bTWCID),
		PrimaryData: *p,
	}
	tpl1, _ := ui.Asset("templates/wcinfo.html")
	tpl2, _ := ui.Asset("templates/home.html")
	tpl3, _ := ui.Asset("templates/base.html")
	tpl := append(tpl1, tpl2...)
	tpl = append(tpl, tpl3...)
	tmpl, _ := template.New("").Funcs(funcMap).Parse(string(tpl))
	tmpl.ExecuteTemplate(w, "base", pageData)
}
