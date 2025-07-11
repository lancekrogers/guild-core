package logging

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// HTTPMiddleware provides HTTP request logging
func HTTPMiddleware(logger Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Generate request ID if not present
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = generateRequestID()
			}

			// Enrich context
			ctx := WithRequestID(r.Context(), requestID)
			r = r.WithContext(ctx)

			// Create logger for this request
			reqLogger := logger.WithContext(ctx).With(
				String("method", r.Method),
				String("path", r.URL.Path),
				String("remote_addr", r.RemoteAddr),
			)

			// Log request start
			reqLogger.Info("request started")

			// Wrap response writer to capture status
			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Set request ID header
			w.Header().Set("X-Request-ID", requestID)

			// Handle panic recovery
			defer func() {
				if err := recover(); err != nil {
					duration := time.Since(start)
					reqLogger.Error("request panicked",
						Any("panic", err),
						Duration("duration", duration),
						Int("status", http.StatusInternalServerError),
					)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()

			// Process request
			next.ServeHTTP(wrapped, r)

			// Log request completion
			duration := time.Since(start)
			fields := []Field{
				Int("status", wrapped.statusCode),
				Int("bytes", wrapped.bytesWritten),
				Duration("duration", duration),
			}

			// Add query parameters if present
			if r.URL.RawQuery != "" {
				fields = append(fields, String("query", r.URL.RawQuery))
			}

			// Log based on status code
			switch {
			case wrapped.statusCode >= 500:
				reqLogger.Error("request completed", fields...)
			case wrapped.statusCode >= 400:
				reqLogger.Warn("request completed", fields...)
			default:
				reqLogger.Info("request completed", fields...)
			}
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code and bytes written
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
	written      bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.ResponseWriter.WriteHeader(code)
		rw.written = true
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

// GRPCUnaryServerInterceptor provides gRPC unary request logging
func GRPCUnaryServerInterceptor(logger Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// Generate request ID if not present
		requestID, ok := RequestID(ctx)
		if !ok {
			requestID = generateRequestID()
			ctx = WithRequestID(ctx, requestID)
		}

		// Create logger for this RPC
		rpcLogger := logger.WithContext(ctx).With(
			String("grpc.method", info.FullMethod),
			String("grpc.type", "unary"),
		)

		// Log RPC start
		rpcLogger.Info("grpc request started")

		// Handle panic recovery
		var resp interface{}
		var err error
		panicked := true
		defer func() {
			if r := recover(); r != nil || panicked {
				duration := time.Since(start)
				rpcLogger.Error("grpc request panicked",
					Any("panic", r),
					Duration("duration", duration),
				)
				err = status.Errorf(codes.Internal, "Internal Server Error")
			}
		}()

		// Process request
		resp, err = handler(ctx, req)
		panicked = false

		// Log request completion
		duration := time.Since(start)
		code := codes.OK
		if err != nil {
			if s, ok := status.FromError(err); ok {
				code = s.Code()
			} else {
				code = codes.Unknown
			}
		}

		fields := []Field{
			String("grpc.code", code.String()),
			Duration("duration", duration),
		}

		if err != nil {
			fields = append(fields, ErrorField(err))
		}

		// Log based on error status
		switch code {
		case codes.OK:
			rpcLogger.Info("grpc request completed", fields...)
		case codes.Canceled, codes.InvalidArgument, codes.NotFound, codes.AlreadyExists,
			codes.PermissionDenied, codes.Unauthenticated, codes.FailedPrecondition,
			codes.OutOfRange:
			rpcLogger.Warn("grpc request completed", fields...)
		default:
			rpcLogger.Error("grpc request completed", fields...)
		}

		return resp, err
	}
}

// GRPCStreamServerInterceptor provides gRPC stream logging
func GRPCStreamServerInterceptor(logger Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		ctx := ss.Context()

		// Generate request ID if not present
		requestID, ok := RequestID(ctx)
		if !ok {
			requestID = generateRequestID()
			ctx = WithRequestID(ctx, requestID)
		}

		// Create logger for this stream
		streamLogger := logger.WithContext(ctx).With(
			String("grpc.method", info.FullMethod),
			String("grpc.type", "stream"),
			Bool("grpc.client_stream", info.IsClientStream),
			Bool("grpc.server_stream", info.IsServerStream),
		)

		// Log stream start
		streamLogger.Info("grpc stream started")

		// Wrap the stream
		wrapped := &loggingServerStream{
			ServerStream: ss,
			ctx:          ctx,
			logger:       streamLogger,
		}

		// Handle panic recovery
		var err error
		panicked := true
		defer func() {
			if r := recover(); r != nil || panicked {
				duration := time.Since(start)
				streamLogger.Error("grpc stream panicked",
					Any("panic", r),
					Duration("duration", duration),
				)
				err = status.Errorf(codes.Internal, "Internal Server Error")
			}
		}()

		// Process stream
		err = handler(srv, wrapped)
		panicked = false

		// Log stream completion
		duration := time.Since(start)
		code := codes.OK
		if err != nil {
			if s, ok := status.FromError(err); ok {
				code = s.Code()
			} else {
				code = codes.Unknown
			}
		}

		fields := []Field{
			String("grpc.code", code.String()),
			Duration("duration", duration),
			Int("messages_sent", wrapped.msgSent),
			Int("messages_received", wrapped.msgRecv),
		}

		if err != nil {
			fields = append(fields, ErrorField(err))
		}

		// Log based on error status
		if err == nil {
			streamLogger.Info("grpc stream completed", fields...)
		} else {
			streamLogger.Error("grpc stream completed", fields...)
		}

		return err
	}
}

// loggingServerStream wraps grpc.ServerStream to count messages
type loggingServerStream struct {
	grpc.ServerStream
	ctx     context.Context
	logger  Logger
	msgSent int
	msgRecv int
}

func (s *loggingServerStream) Context() context.Context {
	return s.ctx
}

func (s *loggingServerStream) SendMsg(m interface{}) error {
	err := s.ServerStream.SendMsg(m)
	if err == nil {
		s.msgSent++
	}
	return err
}

func (s *loggingServerStream) RecvMsg(m interface{}) error {
	err := s.ServerStream.RecvMsg(m)
	if err == nil {
		s.msgRecv++
	}
	return err
}

// generateRequestID generates a unique request ID
var requestIDCounter uint64

func generateRequestID() string {
	// Use nanosecond timestamp + atomic counter for uniqueness
	counter := atomic.AddUint64(&requestIDCounter, 1)
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), counter)
}
