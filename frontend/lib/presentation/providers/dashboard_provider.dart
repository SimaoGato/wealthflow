import 'dart:developer' as developer;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:frontend/core/networking/grpc_client.dart';
import 'package:frontend/generated/wealthflow/v1/service.pb.dart';
import 'package:frontend/generated/wealthflow/v1/service.pbgrpc.dart';

/// Provider that fetches the net worth from the backend
final netWorthProvider = FutureProvider<GetNetWorthResponse>((ref) async {
  try {
    developer.log(
      '📊 [Dashboard] Fetching net worth...',
      name: 'DashboardProvider',
    );
    final client = ref.watch(wealthFlowClientProvider);
    developer.log(
      '📊 [Dashboard] Client obtained, making GetNetWorth request...',
      name: 'DashboardProvider',
    );
    final request = GetNetWorthRequest();
    final response = await client.getNetWorth(request);
    developer.log(
      '✅ [Dashboard] Net worth fetched successfully: ${response.totalNetWorth}',
      name: 'DashboardProvider',
    );
    return response;
  } catch (e, stackTrace) {
    developer.log(
      '❌ [Dashboard] Error fetching net worth: $e',
      name: 'DashboardProvider',
      error: e,
      stackTrace: stackTrace,
    );
    rethrow;
  }
});

/// Provider that fetches recent transactions from the backend
final recentTransactionsProvider =
    FutureProvider<(List<Transaction>, Map<String, String>)>((ref) async {
      try {
        developer.log(
          '📋 [Dashboard] Fetching recent transactions...',
          name: 'DashboardProvider',
        );
        final client = ref.watch(wealthFlowClientProvider);
        developer.log(
          '📋 [Dashboard] Client obtained, making ListTransactions request...',
          name: 'DashboardProvider',
        );
        final request = ListTransactionsRequest()..limit = 10;
        final response = await client.listTransactions(request);
        developer.log(
          '✅ [Dashboard] Transactions fetched successfully: ${response.transactions.length} transactions',
          name: 'DashboardProvider',
        );
        return (response.transactions, response.bucketNames);
      } catch (e, stackTrace) {
        developer.log(
          '❌ [Dashboard] Error fetching transactions: $e',
          name: 'DashboardProvider',
          error: e,
          stackTrace: stackTrace,
        );
        rethrow;
      }
    });
