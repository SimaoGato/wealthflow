import 'dart:developer' as developer;
import 'package:grpc/grpc.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:frontend/core/config/grpc_config.dart';
import 'package:frontend/generated/wealthflow/v1/service.pbgrpc.dart';

part 'grpc_client.g.dart';

/// Interceptor that injects authorization metadata into every gRPC call
class AuthInterceptor implements ClientInterceptor {
  @override
  ResponseStream<R> interceptStreaming<Q, R>(
    ClientMethod<Q, R> method,
    Stream<Q> request,
    CallOptions options,
    ClientStreamingInvoker<Q, R> invoker,
  ) {
    developer.log(
      '🔐 [gRPC] Intercepting streaming call: ${method.path}',
      name: 'AuthInterceptor',
    );
    final metadata = Map<String, String>.from(options.metadata);
    metadata['authorization'] = 'dev-token';
    final updatedOptions = CallOptions(
      metadata: metadata,
      timeout: options.timeout,
      compression: options.compression,
    );
    return invoker(method, request, updatedOptions);
  }

  @override
  ResponseFuture<R> interceptUnary<Q, R>(
    ClientMethod<Q, R> method,
    Q request,
    CallOptions options,
    ClientUnaryInvoker<Q, R> invoker,
  ) {
    developer.log(
      '🔐 [gRPC] Intercepting unary call: ${method.path}',
      name: 'AuthInterceptor',
    );
    final metadata = Map<String, String>.from(options.metadata);
    metadata['authorization'] = 'dev-token';
    final updatedOptions = CallOptions(
      metadata: metadata,
      timeout: options.timeout,
      compression: options.compression,
    );
    return invoker(method, request, updatedOptions);
  }
}

/// Riverpod provider for the gRPC client channel
@riverpod
ClientChannel grpcChannel(GrpcChannelRef ref) {
  final host = GrpcConfig.host;
  final port = GrpcConfig.port;

  developer.log(
    '🔌 [gRPC] Creating channel to $host:$port',
    name: 'GrpcClient',
  );

  if (GrpcConfig.isLocalhost) {
    developer.log(
      '⚠️ [gRPC] WARNING: Using localhost - this will NOT work on physical devices!',
      name: 'GrpcClient',
    );
    developer.log(
      '💡 [gRPC] TIP: Update GrpcConfig.host in lib/core/config/grpc_config.dart with your machine IP',
      name: 'GrpcClient',
    );
  }

  return ClientChannel(
    host,
    port: port,
    options: const ChannelOptions(credentials: ChannelCredentials.insecure()),
  );
}

/// Riverpod provider for the WealthFlowService client
@riverpod
WealthFlowServiceClient wealthFlowClient(WealthFlowClientRef ref) {
  developer.log(
    '🔌 [gRPC] Creating WealthFlowServiceClient',
    name: 'GrpcClient',
  );
  final channel = ref.watch(grpcChannelProvider);
  final client = WealthFlowServiceClient(
    channel,
    interceptors: [AuthInterceptor()],
  );
  developer.log('✅ [gRPC] WealthFlowServiceClient created', name: 'GrpcClient');
  return client;
}
