// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'grpc_client.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

String _$grpcChannelHash() => r'1ca0d6e21c0876a75f844edb02fd77eaef40a1b5';

/// Riverpod provider for the gRPC client channel
///
/// Copied from [grpcChannel].
@ProviderFor(grpcChannel)
final grpcChannelProvider = AutoDisposeProvider<ClientChannel>.internal(
  grpcChannel,
  name: r'grpcChannelProvider',
  debugGetCreateSourceHash: const bool.fromEnvironment('dart.vm.product')
      ? null
      : _$grpcChannelHash,
  dependencies: null,
  allTransitiveDependencies: null,
);

@Deprecated('Will be removed in 3.0. Use Ref instead')
// ignore: unused_element
typedef GrpcChannelRef = AutoDisposeProviderRef<ClientChannel>;
String _$wealthFlowClientHash() => r'13293e870062a031972f7dfbbb9e0259b4ab0a03';

/// Riverpod provider for the WealthFlowService client
///
/// Copied from [wealthFlowClient].
@ProviderFor(wealthFlowClient)
final wealthFlowClientProvider =
    AutoDisposeProvider<WealthFlowServiceClient>.internal(
      wealthFlowClient,
      name: r'wealthFlowClientProvider',
      debugGetCreateSourceHash: const bool.fromEnvironment('dart.vm.product')
          ? null
          : _$wealthFlowClientHash,
      dependencies: null,
      allTransitiveDependencies: null,
    );

@Deprecated('Will be removed in 3.0. Use Ref instead')
// ignore: unused_element
typedef WealthFlowClientRef = AutoDisposeProviderRef<WealthFlowServiceClient>;
// ignore_for_file: type=lint
// ignore_for_file: subtype_of_sealed_class, invalid_use_of_internal_member, invalid_use_of_visible_for_testing_member, deprecated_member_use_from_same_package
