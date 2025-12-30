import 'dart:developer' as developer;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:frontend/core/networking/grpc_client.dart';
import 'package:frontend/generated/wealthflow/v1/service.pb.dart';
import 'package:frontend/generated/wealthflow/v1/service.pbgrpc.dart';
import 'package:frontend/generated/wealthflow/v1/service.pbenum.dart';

/// Provider that fetches all buckets from the backend
final bucketsProvider = FutureProvider<ListBucketsResponse>((ref) async {
  try {
    developer.log(
      '🪣 [Buckets] Fetching all buckets...',
      name: 'BucketProvider',
    );
    final client = ref.watch(wealthFlowClientProvider);
    developer.log(
      '🪣 [Buckets] Client obtained, making ListBuckets request...',
      name: 'BucketProvider',
    );
    final request = ListBucketsRequest();
    final response = await client.listBuckets(request);
    developer.log(
      '✅ [Buckets] Buckets fetched successfully: ${response.buckets.length} buckets',
      name: 'BucketProvider',
    );
    return response;
  } catch (e, stackTrace) {
    developer.log(
      '❌ [Buckets] Error fetching buckets: $e',
      name: 'BucketProvider',
      error: e,
      stackTrace: stackTrace,
    );
    rethrow;
  }
});

/// Organized buckets grouped by type for easy UI consumption
class OrganizedBuckets {
  final List<Bucket> incomeBuckets;
  final List<Bucket> expenseBuckets;
  final List<Bucket> virtualBuckets;
  final List<Bucket> physicalBuckets;
  final List<Bucket> equityBuckets;

  OrganizedBuckets({
    required this.incomeBuckets,
    required this.expenseBuckets,
    required this.virtualBuckets,
    required this.physicalBuckets,
    required this.equityBuckets,
  });
}

/// Provider that organizes buckets by type
final organizedBucketsProvider = Provider<AsyncValue<OrganizedBuckets>>((ref) {
  final bucketsAsync = ref.watch(bucketsProvider);

  return bucketsAsync.when(
    data: (response) {
      final incomeBuckets = <Bucket>[];
      final expenseBuckets = <Bucket>[];
      final virtualBuckets = <Bucket>[];
      final physicalBuckets = <Bucket>[];
      final equityBuckets = <Bucket>[];

      for (final bucket in response.buckets) {
        if (bucket.type == BucketType.BUCKET_TYPE_PHYSICAL) {
          physicalBuckets.add(bucket);
        } else if (bucket.type == BucketType.BUCKET_TYPE_VIRTUAL) {
          virtualBuckets.add(bucket);
        } else if (bucket.type == BucketType.BUCKET_TYPE_INCOME) {
          incomeBuckets.add(bucket);
        } else if (bucket.type == BucketType.BUCKET_TYPE_EXPENSE) {
          expenseBuckets.add(bucket);
        } else if (bucket.type == BucketType.BUCKET_TYPE_EQUITY) {
          equityBuckets.add(bucket);
        }
      }

      return AsyncValue.data(
        OrganizedBuckets(
          incomeBuckets: incomeBuckets,
          expenseBuckets: expenseBuckets,
          virtualBuckets: virtualBuckets,
          physicalBuckets: physicalBuckets,
          equityBuckets: equityBuckets,
        ),
      );
    },
    loading: () => const AsyncValue.loading(),
    error: (error, stackTrace) => AsyncValue.error(error, stackTrace),
  );
});

/// Helper function to refresh buckets (useful after adding transactions)
void refreshBuckets(WidgetRef ref) {
  developer.log('🔄 [Buckets] Refreshing buckets...', name: 'BucketProvider');
  ref.invalidate(bucketsProvider);
}
