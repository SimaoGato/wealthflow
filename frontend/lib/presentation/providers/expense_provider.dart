import 'dart:developer' as developer;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:frontend/core/networking/grpc_client.dart';
import 'package:frontend/generated/wealthflow/v1/service.pb.dart';
import 'package:frontend/generated/wealthflow/v1/service.pbgrpc.dart';
import 'package:frontend/presentation/providers/bucket_provider.dart';
import 'package:frontend/presentation/providers/dashboard_provider.dart';

part 'expense_provider.g.dart';

/// Controller for logging expense transactions
@riverpod
class ExpenseController extends _$ExpenseController {
  @override
  FutureOr<void> build() {
    // Initial state is empty - controller is ready to use
  }

  /// Logs an expense transaction
  Future<void> logExpense({
    required String amount,
    required String description,
    required String virtualBucketId,
    required String categoryBucketId,
    String? physicalBucketOverrideId,
  }) async {
    state = const AsyncValue.loading();
    try {
      developer.log(
        '💸 [Expense] Logging expense: $amount from virtual bucket $virtualBucketId',
        name: 'ExpenseController',
      );
      final client = ref.read(wealthFlowClientProvider);
      final request = LogExpenseRequest()
        ..amount = amount
        ..description = description
        ..virtualBucketId = virtualBucketId
        ..categoryBucketId = categoryBucketId;

      if (physicalBucketOverrideId != null &&
          physicalBucketOverrideId.isNotEmpty) {
        request.physicalBucketOverrideId = physicalBucketOverrideId;
      }

      final response = await client.logExpense(request);
      developer.log(
        '✅ [Expense] Expense logged successfully: ${response.transactionId}',
        name: 'ExpenseController',
      );

      // Invalidate providers to refresh data
      ref.invalidate(bucketsProvider);
      ref.invalidate(netWorthProvider);
      ref.invalidate(recentTransactionsProvider);

      state = const AsyncValue.data(null);
    } catch (e, stackTrace) {
      developer.log(
        '❌ [Expense] Error logging expense: $e',
        name: 'ExpenseController',
        error: e,
        stackTrace: stackTrace,
      );
      state = AsyncValue.error(e, stackTrace);
      rethrow;
    }
  }
}
