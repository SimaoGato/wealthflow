package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAuthInterceptor(t *testing.T) {
	validToken := "test-token-123"
	interceptor := AuthInterceptor(validToken)

	tests := []struct {
		name           string
		ctx            context.Context
		handlerCalled  bool
		expectedCode   codes.Code
		expectedErrMsg string
	}{
		{
			name: "Valid Token",
			ctx: metadata.NewIncomingContext(
				context.Background(),
				metadata.Pairs("authorization", validToken),
			),
			handlerCalled:  true,
			expectedCode:   codes.OK,
			expectedErrMsg: "",
		},
		{
			name: "Invalid Token",
			ctx: metadata.NewIncomingContext(
				context.Background(),
				metadata.Pairs("authorization", "wrong-token"),
			),
			handlerCalled:  false,
			expectedCode:   codes.Unauthenticated,
			expectedErrMsg: "invalid token",
		},
		{
			name:           "Missing Token",
			ctx:            context.Background(),
			handlerCalled:  false,
			expectedCode:   codes.Unauthenticated,
			expectedErrMsg: "missing metadata",
		},
		{
			name: "Missing Authorization Header",
			ctx: metadata.NewIncomingContext(
				context.Background(),
				metadata.Pairs("other-header", "value"),
			),
			handlerCalled:  false,
			expectedCode:   codes.Unauthenticated,
			expectedErrMsg: "missing authorization header",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled := false
			handler := func(ctx context.Context, req interface{}) (interface{}, error) {
				handlerCalled = true
				return "success", nil
			}

			info := &grpc.UnaryServerInfo{
				FullMethod: "/test.Service/Method",
			}

			resp, err := interceptor(tt.ctx, "test-request", info, handler)

			assert.Equal(t, tt.handlerCalled, handlerCalled, "handler called status mismatch")

			if tt.expectedCode == codes.OK {
				assert.NoError(t, err)
				assert.Equal(t, "success", resp)
			} else {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok, "error should be a gRPC status")
				assert.Equal(t, tt.expectedCode, st.Code())
				assert.Contains(t, st.Message(), tt.expectedErrMsg)
			}
		})
	}
}
