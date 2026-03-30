package grpcclient

import (
	"crypto/tls"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

const dialOptionCapacity = 5

func DialOptions(timeout time.Duration, useTLS bool, serverName, bearer string, allowInsecureTLS bool) []grpc.DialOption {
	options := make([]grpc.DialOption, 0, dialOptionCapacity)

	unaryInterceptors := []grpc.UnaryClientInterceptor{UnaryTimeoutInterceptor(timeout)}
	streamInterceptors := []grpc.StreamClientInterceptor{StreamTimeoutInterceptor(timeout)}

	if useTLS {
		tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
		if serverName != "" {
			tlsConfig.ServerName = serverName
		}

		tlsConfig.InsecureSkipVerify = allowInsecureTLS

		options = append(options, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	} else {
		options = append(options, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	if bearer != "" {
		unaryInterceptors = append(unaryInterceptors, UnaryBearerInterceptor(bearer))
		streamInterceptors = append(streamInterceptors, StreamBearerInterceptor(bearer))
	}

	options = append(options,
		grpc.WithChainUnaryInterceptor(unaryInterceptors...),
		grpc.WithChainStreamInterceptor(streamInterceptors...),
	)

	return options
}
