package pia

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/go-ping/ping"
)

type Region struct {
	Id          string              `json:"id"`
	Name        string              `json:"name"`
	PortForward bool                `json:"port_forward"`
	Servers     map[string][]Server `json:"servers"`
	PingTime    time.Duration
}

func (self *Region) server(typ string) *Server {
	s, ok := self.Servers[typ]
	if !ok {
		return nil
	}
	return &s[0]
}

func (self *Region) WgServer() *Server {
	return self.server("wg")
}

func (self *Region) MetaServer() *Server {
	return self.server("meta")
}

func (self *Region) HasWg() bool {
	return self.WgServer() != nil
}

func (self *Region) setPingTime() {
	wg := self.WgServer()
	if wg == nil {
		self.PingTime = time.Duration(0)
		return
	}
	pinger, err := ping.NewPinger(wg.Ip)
	if err != nil {
		self.PingTime = time.Duration(0)
	}
	pinger.Count = 3
	pinger.Timeout = 1 * time.Second
	pinger.OnFinish = func(stats *ping.Statistics) {
		self.PingTime = stats.AvgRtt
	}
	err = pinger.Run()
	if err != nil {
		self.PingTime = time.Duration(0)
	}
	return
}

func RegionsWithPingTime() ([]Region, error) {
	regions, err := Regions()
	if err != nil {
		return nil, err
	}

	// parallelize the pings, obviously, or we'll be here all day
	var done sync.WaitGroup
	for i := range regions {
		r := &regions[i]
		done.Add(1)
		go func(done *sync.WaitGroup) {
			r.setPingTime()
			done.Done()
		}(&done)
	}
	done.Wait()

	// sort by increasing ping time
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

func Regions() ([]Region, error) {
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
	return _r.Regions, nil
}

func FindRegion(id string) (*Region, error) {
	regions, err := Regions()
	if err != nil {
		return nil, err
	}
	for i := range regions {
		r := &regions[i]
		if r.Id == id {
			if r.WgServer() == nil {
				return nil, fmt.Errorf("Region %s (%s) was found but does not have a WireGuard server", r.Id, r.Name)
			}
			r.setPingTime()
			if r.PingTime == 0 {
				fmt.Fprintf(os.Stderr, "Warning: WireGuard server for region %s (%s) is not currently reachable at %s", r.Id, r.Name, r.WgServer().Ip)
			}
			return r, nil
		}
	}
	return nil, fmt.Errorf("Could not find region %s", id)
}
