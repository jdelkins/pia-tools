package pia

import (
	"encoding/json"
	"net/http"
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

func (self *Region) setPingTime(done *sync.WaitGroup) {
	defer done.Done()
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
	regions := _r.Regions

	var done sync.WaitGroup
	for i := range regions {
		r := &regions[i]
		done.Add(1)
		go r.setPingTime(&done)
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
