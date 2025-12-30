import 'dart:developer' as developer;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:frontend/core/networking/grpc_client.dart';
import 'package:frontend/generated/wealthflow/v1/service.pb.dart';
import 'package:frontend/generated/wealthflow/v1/service.pbgrpc.dart';
import 'package:frontend/presentation/providers/bucket_provider.dart';
import 'package:frontend/presentation/providers/dashboard_provider.dart';

part 'inflow_provider.g.dart';

/// Controller for recording inflow transactions
@riverpod
class InflowController extends _$InflowController {
  @override
  FutureOr<void> build() {
    // Initial state is empty - controller is ready to use
  }

  /// Records an inflow transaction
  Future<void> recordInflow({
    required String amount,
    required String description,
    required String sourceBucketId,
    required bool isExternal,
  }) async {
    state = const AsyncValue.loading();
    try {
      developer.log(
        '💰 [Inflow] Recording inflow: $amount from $sourceBucketId',
        name: 'InflowController',
      );
      final client = ref.read(wealthFlowClientProvider);
      final request = RecordInflowRequest()
        ..amount = amount
        ..description = description
        ..sourceBucketId = sourceBucketId
        ..isExternal = isExternal;

      final response = await client.recordInflow(request);
      developer.log(
        '✅ [Inflow] Inflow recorded successfully: ${response.transactionId}',
        name: 'InflowController',
      );

      // Invalidate providers to refresh data
      ref.invalidate(bucketsProvider);
      ref.invalidate(netWorthProvider);
      ref.invalidate(recentTransactionsProvider);

      state = const AsyncValue.data(null);
    } catch (e, stackTrace) {
      developer.log(
        '❌ [Inflow] Error recording inflow: $e',
        name: 'InflowController',
        error: e,
        stackTrace: stackTrace,
      );
      state = AsyncValue.error(e, stackTrace);
      rethrow;
    }
  }
}
