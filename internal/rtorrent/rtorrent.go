package rtorrent

import (
	"fmt"

	"github.com/jdelkins/pia-tools/internal/httpclient"
	"github.com/kolo/xmlrpc"
)

var transport = httpclient.WithTimeout(nil, httpclient.Timeout)

func Notify(url string, port int) error {
	client, err := xmlrpc.NewClient(url+"/RPC2", transport)
	if err != nil {
		return err
	}
	defer client.Close()
	p := []interface{}{
		"",
		fmt.Sprintf("%d-%d", port, port),
	}
	if err := client.Call("network.port_range.set", p, nil); err != nil {
		return err
	}
	// apparently a second call is necessary to re-bind the listeners
	p = []interface{}{
		"",
		"0.0.0.0",
	}
	if err := client.Call("network.bind_address.set", p, nil); err != nil {
		return err
	}
	return nil
}

func Confirm(url string) (int, error) {
	client, err := xmlrpc.NewClient(url+"/RPC2", transport)
	if err != nil {
		return -1, err
	}
	defer client.Close()
	var resp string
	if err := client.Call("network.port_range", "", &resp); err != nil {
		return -1, err
	}
	var port, port2 int
	_, err = fmt.Sscanf(resp, "%d-%d", &port, &port2)
	if err != nil {
		return -1, err
	}
	return port, nil
}
