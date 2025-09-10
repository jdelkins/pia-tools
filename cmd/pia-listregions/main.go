package main

import (
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/jdelkins/pia-tools/internal/pia"
)

func main() {
	regions, err := pia.RegionsWithPingTime()
	if err != nil {
		log.Fatalf("%v", err)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", "ID",             "NAME",                    "PING",      "WG?", "PF?")
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", "==============", "=======================", "=========", "===", "===")
	for i := range regions {
		r := &regions[i]
		wg := ""
		if r.HasWg() {
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
