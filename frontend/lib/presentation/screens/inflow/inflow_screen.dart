import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:frontend/generated/wealthflow/v1/service.pb.dart';
import 'package:frontend/presentation/providers/bucket_provider.dart';
import 'package:frontend/presentation/providers/inflow_provider.dart';

class InflowScreen extends ConsumerStatefulWidget {
  const InflowScreen({super.key});

  @override
  ConsumerState<InflowScreen> createState() => _InflowScreenState();
}

class _InflowScreenState extends ConsumerState<InflowScreen> {
  final _amountController = TextEditingController();
  final _descriptionController = TextEditingController();
  Bucket? _selectedSourceBucket;
  bool _isExternal = true;

  @override
  void dispose() {
    _amountController.dispose();
    _descriptionController.dispose();
    super.dispose();
  }

  bool get _isFormValid {
    return _selectedSourceBucket != null &&
        _amountController.text.trim().isNotEmpty;
  }

  void _handleSubmit() async {
    if (!_isFormValid) return;

    final amount = _amountController.text.trim();
    final description = _descriptionController.text.trim().isEmpty
        ? 'Inflow'
        : _descriptionController.text.trim();

    try {
      await ref
          .read(inflowControllerProvider.notifier)
          .recordInflow(
            amount: amount,
            description: description,
            sourceBucketId: _selectedSourceBucket!.id,
            isExternal: _isExternal,
          );

      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text('Inflow recorded successfully!'),
            backgroundColor: Colors.green,
          ),
        );

        // Clear form
        _amountController.clear();
        _descriptionController.clear();
        setState(() {
          _selectedSourceBucket = null;
          _isExternal = true;
        });
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('Error: ${e.toString()}'),
            backgroundColor: Colors.red,
          ),
        );
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final organizedBucketsAsync = ref.watch(organizedBucketsProvider);
    final inflowState = ref.watch(inflowControllerProvider);

    return Scaffold(
      appBar: AppBar(title: const Text('Record Inflow')),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Source Bucket Dropdown
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'Source Bucket',
                      style: Theme.of(context).textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                    const SizedBox(height: 8),
                    organizedBucketsAsync.when(
                      data: (organizedBuckets) {
                        final incomeBuckets = organizedBuckets.incomeBuckets;
                        if (incomeBuckets.isEmpty) {
                          return const Text(
                            'No income buckets available',
                            style: TextStyle(color: Colors.grey),
                          );
                        }
                        return DropdownButtonFormField<Bucket>(
                          value: _selectedSourceBucket,
                          decoration: const InputDecoration(
                            border: OutlineInputBorder(),
                            hintText: 'Select source bucket',
                          ),
                          items: incomeBuckets.map((bucket) {
                            return DropdownMenuItem<Bucket>(
                              value: bucket,
                              child: Text(bucket.name),
                            );
                          }).toList(),
                          onChanged: (bucket) {
                            setState(() {
                              _selectedSourceBucket = bucket;
                            });
                          },
                        );
                      },
                      loading: () => const Center(
                        child: Padding(
                          padding: EdgeInsets.all(16),
                          child: CircularProgressIndicator(),
                        ),
                      ),
                      error: (error, stackTrace) => Text(
                        'Error loading buckets: $error',
                        style: const TextStyle(color: Colors.red),
                      ),
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 16),
            // Amount Field
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'Amount',
                      style: Theme.of(context).textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                    const SizedBox(height: 8),
                    TextField(
                      controller: _amountController,
                      keyboardType: const TextInputType.numberWithOptions(
                        decimal: true,
                      ),
                      inputFormatters: [
                        FilteringTextInputFormatter.allow(
                          RegExp(r'^\d+\.?\d{0,2}'),
                        ),
                      ],
                      decoration: const InputDecoration(
                        border: OutlineInputBorder(),
                        hintText: '0.00',
                        prefixText: '€ ',
                      ),
                      onChanged: (_) => setState(() {}),
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 16),
            // Description Field
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'Description',
                      style: Theme.of(context).textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                    const SizedBox(height: 8),
                    TextField(
                      controller: _descriptionController,
                      decoration: const InputDecoration(
                        border: OutlineInputBorder(),
                        hintText: 'e.g., Monthly Salary',
                      ),
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 16),
            // External Inflow Switch
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            'External Inflow',
                            style: Theme.of(context).textTheme.titleMedium
                                ?.copyWith(fontWeight: FontWeight.bold),
                          ),
                          const SizedBox(height: 4),
                          Text(
                            'Triggers the split rule engine',
                            style: Theme.of(
                              context,
                            ).textTheme.bodySmall?.copyWith(color: Colors.grey),
                          ),
                        ],
                      ),
                    ),
                    Switch(
                      value: _isExternal,
                      onChanged: (value) {
                        setState(() {
                          _isExternal = value;
                        });
                      },
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 24),
            // Submit Button
            SizedBox(
              width: double.infinity,
              height: 56,
              child: FilledButton(
                onPressed: _isFormValid && !inflowState.isLoading
                    ? _handleSubmit
                    : null,
                style: FilledButton.styleFrom(
                  backgroundColor: Theme.of(context).colorScheme.primary,
                  foregroundColor: Theme.of(context).colorScheme.onPrimary,
                  shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(12),
                  ),
                ),
                child: inflowState.isLoading
                    ? const SizedBox(
                        height: 24,
                        width: 24,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          valueColor: AlwaysStoppedAnimation<Color>(
                            Colors.white,
                          ),
                        ),
                      )
                    : const Text(
                        'Record Inflow',
                        style: TextStyle(
                          fontSize: 16,
                          fontWeight: FontWeight.bold,
                        ),
                      ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
