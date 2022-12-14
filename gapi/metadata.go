package gapi

import (
	"context"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

// const variables we will use to identify the metadata we are looking for
// the metadata is a map with string keys and values which are slices of strings
const (
	grpcGatewayUserAgentHeader = "grpcgateway-user-agent" // HTTP metadata (goes through the gRPC gateway)
	userAgentHeader            = "user-agent"             // gRPC metadata differs from HTTP metadata
	xForwardedForHeader        = "x-forwarded-for"        // HTTP metadata (goes through the gRPC gateway)
)

// all metadata that we want to extract from the context
type Metadata struct {
	UserAgent string
	ClientIP  string
}

func (server *Server) extractMetadata(ctx context.Context) *Metadata {
	mtdt := &Metadata{}

	// metadata is a subpackage of gRPC which provides the FromIncomingContext function
	// FromIncomingContext returns metadata in context if it exists
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		// grab the User Agent data and, if it is not empty (length greater than 0), the user agent data will be the first
		// element in the slice - HTTP
		if userAgents := md.Get(grpcGatewayUserAgentHeader); len(userAgents) > 0 {
			mtdt.UserAgent = userAgents[0]
		}

		// grab the User Agent data - this applies when gRPC is used. The md.Get(grpcGatewayUserAgentHeader) would produce a
		// slice (userAgents) with length equal to zero - in which case, the if statement below should produce a slice of
		// length greater than zero and thus execute
		if userAgents := md.Get(userAgentHeader); len(userAgents) > 0 {
			mtdt.UserAgent = userAgents[0]
		}

		// grab the Client IP data and, if it is not empty (length greater than 0), the client IP address will be the first
		// element in the slice - HTTP
		if clientIPs := md.Get(xForwardedForHeader); len(clientIPs) > 0 {
			mtdt.ClientIP = clientIPs[0]
		}
	}

	// gRPC metadata doesn't include the client's IP address - it is in the context, but it isn't obtained using
	// metadata.FromIncomingContext - must use peer.FromContext (peer is another subpackage of gRPC)
	if p, ok := peer.FromContext(ctx); ok {
		mtdt.ClientIP = p.Addr.String()
	}

	return mtdt
}
