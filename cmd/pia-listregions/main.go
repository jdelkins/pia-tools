package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/jdelkins/pia-tools/internal/pia"
)

type Region struct {
	Id          string                  `json:"id"`
	Name        string                  `json:"name"`
	PortForward bool                    `json:"port_forward"`
	Servers     map[string][]pia.Server `json:"servers"`
}

func get_regions() (*[]Region, error) {
	const pia_url_servers = "https://serverlist.piaservers.net/vpninfo/servers/v4"
	resp, err := http.Get(pia_url_servers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var _r struct {
		Regions []Region `json:"regions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&_r); err != nil {
		return nil, err
	}
	regions := _r.Regions
	sort.Slice(regions, func(i, j int) bool {
		return regions[i].Id < regions[j].Id
	})
	return &regions, nil
}

func main() {
	regions, err := get_regions()
	if err != nil {
		log.Fatalf("%v", err)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", "ID", "NAME", "WG?", "PF?")
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", "==============", "=======================", "===", "===")
	for _, r := range *regions {
		_, has_wg := r.Servers["wg"]
		wg := ""
		if has_wg {
			wg = " ✓"
		}
		pf := ""
		if r.PortForward {
			pf = " ✓"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.Id, r.Name, wg, pf)
	}
	w.Flush()
}
