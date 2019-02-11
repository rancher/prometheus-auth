package agent

import (
	"context"

	grpcproxy "github.com/mwitkow/grpc-proxy/proxy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (a *agent) grpcBackend() grpc.StreamHandler {
	return grpcproxy.TransparentHandler(func(ctx context.Context, fullMethodName string) (context.Context, *grpc.ClientConn, error) {
		con, err := grpc.DialContext(ctx, a.cfg.proxyURL.String(), grpc.WithDefaultCallOptions(grpc.CallCustomCodec(grpcproxy.Codec())))
		if err != nil {
			return ctx, nil, status.Errorf(codes.Unavailable, "Unavailable endpoint")
		}

		return ctx, con, nil
	})
}
