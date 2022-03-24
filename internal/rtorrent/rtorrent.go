package rtorrent

import (
	"fmt"

	"github.com/kolo/xmlrpc"
)

func Notify(url string, port int) error {
	client, err := xmlrpc.NewClient(url+"/RPC2", nil)
	if err != nil {
		return err
	}
	p := []interface{}{
		"",
		fmt.Sprintf("%d-%d", port, port),
	}
	if err := client.Call("network.port_range.set", p, nil); err != nil {
		return err
	}
	return nil
}

func Confirm(url string) (int, error) {
	client, err := xmlrpc.NewClient(url+"/RPC2", nil)
	if err != nil {
		return -1, err
	}
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
