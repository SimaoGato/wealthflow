import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:grpc/grpc.dart';
import 'package:intl/intl.dart';
import 'package:frontend/core/utils/currency_formatter.dart';
import 'package:frontend/presentation/providers/dashboard_provider.dart';
import 'package:frontend/generated/wealthflow/v1/service.pb.dart';

class DashboardScreen extends ConsumerWidget {
  const DashboardScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final netWorthAsync = ref.watch(netWorthProvider);
    final transactionsAsync = ref.watch(recentTransactionsProvider);

    return Scaffold(
      appBar: AppBar(title: const Text('Dashboard')),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Net Worth Card
            Card(
              child: Padding(
                padding: const EdgeInsets.all(24),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'Net Worth',
                      style: Theme.of(
                        context,
                      ).textTheme.titleMedium?.copyWith(color: Colors.grey),
                    ),
                    const SizedBox(height: 8),
                    netWorthAsync.when(
                      data: (response) => Text(
                        formatCurrency(response.totalNetWorth),
                        style: Theme.of(context).textTheme.displayLarge
                            ?.copyWith(
                              fontWeight: FontWeight.bold,
                              color: Colors.white,
                            ),
                      ),
                      loading: () => const SizedBox(
                        height: 48,
                        child: Center(child: CircularProgressIndicator()),
                      ),
                      error: (error, stack) => Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            'Connection Error',
                            style: Theme.of(context).textTheme.displayLarge
                                ?.copyWith(
                                  fontWeight: FontWeight.bold,
                                  color: Colors.red,
                                ),
                          ),
                          const SizedBox(height: 8),
                          Text(
                            _formatGrpcError(error),
                            style: Theme.of(context).textTheme.bodyMedium
                                ?.copyWith(color: Colors.red.shade300),
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 24),
            // Recent Transactions Section
            Text(
              'Recent Transactions',
              style: Theme.of(
                context,
              ).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 16),
            transactionsAsync.when(
              data: (data) {
                final transactions = data.$1;
                final bucketNames = data.$2;

                if (transactions.isEmpty) {
                  return const Card(
                    child: Padding(
                      padding: EdgeInsets.all(24),
                      child: Center(child: Text('No transactions yet')),
                    ),
                  );
                }

                return Card(
                  child: Column(
                    children: transactions.map((transaction) {
                      return _buildTransactionTile(
                        context,
                        transaction,
                        bucketNames,
                      );
                    }).toList(),
                  ),
                );
              },
              loading: () => const Card(
                child: Padding(
                  padding: EdgeInsets.all(24),
                  child: Center(child: CircularProgressIndicator()),
                ),
              ),
              error: (error, stack) => Card(
                child: Padding(
                  padding: const EdgeInsets.all(24),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'Connection Error',
                        style: Theme.of(context).textTheme.titleMedium
                            ?.copyWith(
                              fontWeight: FontWeight.bold,
                              color: Colors.red,
                            ),
                      ),
                      const SizedBox(height: 8),
                      Text(
                        _formatGrpcError(error),
                        style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                          color: Colors.red.shade300,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildTransactionTile(
    BuildContext context,
    Transaction transaction,
    Map<String, String> bucketNames,
  ) {
    final dateFormat = DateFormat('dd MMM, HH:mm');
    final date = transaction.hasDate()
        ? dateFormat.format(transaction.date.toDateTime())
        : 'Unknown date';

    return ListTile(
      title: Text(transaction.description),
      subtitle: Text(date),
      trailing: Text(
        formatCurrency(transaction.amount),
        style: Theme.of(
          context,
        ).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold),
      ),
    );
  }

  /// Formats gRPC errors for display
  String _formatGrpcError(Object error) {
    if (error is GrpcError) {
      final buffer = StringBuffer();
      buffer.writeln('Code: ${error.code} (${error.codeName})');
      final message = error.message;
      if (message != null && message.isNotEmpty) {
        buffer.writeln('Message: $message');
      }
      if (error.rawResponse != null) {
        buffer.writeln('Response: ${error.rawResponse}');
      }
      return buffer.toString();
    }
    return error.toString();
  }
}
