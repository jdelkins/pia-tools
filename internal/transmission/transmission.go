package transmission

import (
	"context"

	xm "github.com/hekmon/transmissionrpc/v2"
)

func Notify(server, username, password string, port int) error {
	client, err := xm.New(server, username, password, nil)
	if err != nil {
		return err
	}
	var port64 int64 = int64(port)
	args := xm.SessionArguments{
		PeerPort: &port64,
	}
	ctx := context.Background()
	if err = client.SessionArgumentsSet(ctx, args); err != nil {
		return err
	}
	return nil
}

func Confirm(server, username, password string) (int, error) {
	client, err := xm.New(server, username, password, nil)
	if err != nil {
		return 0, err
	}
	ctx := context.Background()
	args, err := client.SessionArgumentsGet(ctx, []string{"peer-port"})
	if err != nil {
		return 0, err
	}

	return int(*args.PeerPort), nil
}
