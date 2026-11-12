package transmission

import (
	"context"
	"net/url"

	xm "github.com/hekmon/transmissionrpc/v3"
	"github.com/jdelkins/pia-tools/internal/httpclient"
)

var httpClient = httpclient.New()

func Notify(server, username, password string, port int) error {
	endpoint, err := url.Parse(server)
	if err != nil {
		return err
	}

	if username != "" {
		endpoint.User = url.UserPassword(username, password)
	}

	config := xm.Config{CustomClient: httpClient}

	client, err := xm.New(endpoint, &config)
	if err != nil {
		return err
	}
	var port64 = int64(port)
	args := xm.SessionArguments{
		PeerPort: &port64,
	}
	ctx, cancel := context.WithTimeout(context.Background(), httpclient.Timeout)
	defer cancel()
	if err = client.SessionArgumentsSet(ctx, args); err != nil {
		return err
	}
	return nil
}

func Confirm(server, username, password string) (int, error) {
	endpoint, err := url.Parse(server)
	if err != nil {
		return 0, err
	}

	if username != "" {
		endpoint.User = url.UserPassword(username, password)
	}

	config := xm.Config{CustomClient: httpClient}

	client, err := xm.New(endpoint, &config)
	if err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), httpclient.Timeout)
	defer cancel()
	args, err := client.SessionArgumentsGet(ctx, []string{"peer-port"})
	if err != nil {
		return 0, err
	}

	return int(*args.PeerPort), nil
}
