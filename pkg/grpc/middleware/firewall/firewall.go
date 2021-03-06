package firewall

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

var (
	ErrInvalidRole = status.Errorf(codes.PermissionDenied, "Invalid role")
)

const mdIdentityKey = "identity"

// AppendIdentityToOutgoingUnaryContext appends identity to outgoing context
//
// https://godoc.org/google.golang.org/grpc#WithUnaryInterceptor
//
// conn, err := grpc.Dial("localhost:5000", grpc.WithUnaryInterceptor(AppendIdentityToOutgoingUnaryContext()))
func AppendIdentityToOutgoingUnaryContext() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		i, ok := identity.FromContext(ctx)
		if ok {
			jsn, err := json.Marshal(i)
			if err != nil {
				return err
			}

			ctx = metadata.AppendToOutgoingContext(ctx, mdIdentityKey, string(jsn))
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// AppendIdentityToOutgoingStreamContext appends identity to outgoing context
//
// https://godoc.org/google.golang.org/grpc#WithStreamInterceptor
//
// conn, err := grpc.Dial("localhost:5000", grpc.WithStreamInterceptor(AppendIdentityToOutgoingStreamContext()))
func AppendIdentityToOutgoingStreamContext() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		i, ok := identity.FromContext(ctx)
		if ok {
			jsn, err := json.Marshal(i)
			if err != nil {
				return nil, err
			}

			ctx = metadata.AppendToOutgoingContext(ctx, mdIdentityKey, string(jsn))
		}

		return streamer(ctx, desc, cc, method, opts...)
	}
}

// GrantAccessForStreamRequest returns error if Identity not set within context or user does not have required role
//
// 	https://godoc.org/google.golang.org/grpc#StreamInterceptor
//
// opts := []grpc.ServerOption{
// 	grpc.StreamInterceptor(GrantAccessForStreamRequest("admin")),
// }
// s := grpc.NewServer(opts...)
// pb.RegisterGreeterServer(s, &server{})
func GrantAccessForStreamRequest(role string) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if md, ok := metadata.FromIncomingContext(ss.Context()); ok {
			var i identity.Identity
			if values := md.Get(mdIdentityKey); len(values) > 0 {
				err := json.Unmarshal([]byte(values[0]), &i)
				if err != nil {
					return err
				}

				// TODO: update server stream context
				// ctx := identity.ContextWithIdentity(ss.Context(), i)

				for _, userRole := range i.Roles {
					if userRole == role {
						return handler(srv, ss)
					}
				}
			}
		}

		return ErrInvalidRole
	}
}

// CheckAccessForUnaryRequest returns error if Identity not set within context or user does not have required role
//
// 	https://godoc.org/google.golang.org/grpc#UnaryInterceptor
//
// opts := []grpc.ServerOption{
// 	grpc.UnaryInterceptor(CheckAccessForUnaryRequest("admin")),
// }
// s := grpc.NewServer(opts...)
// pb.RegisterGreeterServer(s, &server{})
func GrantAccessForUnaryRequest(role string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			var i identity.Identity
			if values := md.Get(mdIdentityKey); len(values) > 0 {
				err := json.Unmarshal([]byte(values[0]), &i)
				if err != nil {
					return nil, err
				}

				ctx = identity.ContextWithIdentity(ctx, i)

				for _, userRole := range i.Roles {
					if userRole == role {
						return handler(ctx, req)
					}
				}
			}
		}

		return nil, ErrInvalidRole
	}
}
