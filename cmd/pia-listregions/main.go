package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/go-ping/ping"
	"github.com/jdelkins/pia-tools/internal/pia"
)

type Region struct {
	Id          string                  `json:"id"`
	Name        string                  `json:"name"`
	PortForward bool                    `json:"port_forward"`
	Servers     map[string][]pia.Server `json:"servers"`
	PingTime    time.Duration
}

func (self *Region) getPingTime(done *sync.WaitGroup) {
	wg, has_wg := self.Servers["wg"]
	if !has_wg {
		self.PingTime = time.Duration(0)
		return
	}
	pinger, err := ping.NewPinger(wg[0].Ip)
	if err != nil {
		self.PingTime = time.Duration(0)
	}
	pinger.Count = 3
	pinger.Timeout = 1 * time.Second
	pinger.OnFinish = func(stats *ping.Statistics) {
		self.PingTime = stats.AvgRtt
		done.Done()
	}
	err = pinger.Run()
	if err != nil {
		self.PingTime = time.Duration(0)
	}
	return
}

func getRegions() ([]Region, error) {
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

	var done sync.WaitGroup
	for i := range regions {
		r := &regions[i]
		done.Add(1)
		go r.getPingTime(&done)
	}
	done.Wait()

	sort.Slice(regions, func(i, j int) bool {
		if regions[i].PingTime == 0 {
			return false
		}
		if regions[j].PingTime == 0 {
			return true
		}
		return regions[i].PingTime < regions[j].PingTime
	})
	return regions, nil
}

func main() {
	regions, err := getRegions()
	if err != nil {
		log.Fatalf("%v", err)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", "ID", "NAME", "PING", "WG?", "PF?")
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", "==============", "=======================", "=========", "===", "===")
	for i := range regions {
		r := &regions[i]
		_, has_wg := r.Servers["wg"]
		wg := ""
		if has_wg {
			wg = " ✓"
		}
		pf := ""
		if r.PortForward {
			pf = " ✓"
		}
		if r.PingTime == 0 {
			fmt.Fprintf(w, "%s\t%s\tN/A\t%s\t%s\n", r.Id, r.Name, wg, pf)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%d ms\t%s\t%s\n", r.Id, r.Name, r.PingTime.Milliseconds(), wg, pf)
		}
	}
	w.Flush()
}
