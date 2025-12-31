import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:frontend/generated/wealthflow/v1/service.pb.dart';
import 'package:frontend/presentation/providers/bucket_provider.dart';
import 'package:frontend/presentation/providers/expense_provider.dart';

class ExpenseForm extends ConsumerStatefulWidget {
  const ExpenseForm({super.key});

  @override
  ConsumerState<ExpenseForm> createState() => _ExpenseFormState();
}

class _ExpenseFormState extends ConsumerState<ExpenseForm> {
  final _amountController = TextEditingController();
  final _descriptionController = TextEditingController();
  Bucket? _selectedVirtualBucket;
  Bucket? _selectedCategoryBucket;
  Bucket? _selectedPhysicalOverrideBucket;
  bool _showAdvanced = false;

  @override
  void dispose() {
    _amountController.dispose();
    _descriptionController.dispose();
    super.dispose();
  }

  bool get _isFormValid {
    return _selectedVirtualBucket != null &&
        _selectedCategoryBucket != null &&
        _amountController.text.trim().isNotEmpty;
  }

  void _handleSubmit() async {
    if (!_isFormValid) return;

    final amount = _amountController.text.trim();
    final description = _descriptionController.text.trim().isEmpty
        ? 'Expense'
        : _descriptionController.text.trim();

    try {
      await ref
          .read(expenseControllerProvider.notifier)
          .logExpense(
            amount: amount,
            description: description,
            virtualBucketId: _selectedVirtualBucket!.id,
            categoryBucketId: _selectedCategoryBucket!.id,
            physicalBucketOverrideId:
                _showAdvanced && _selectedPhysicalOverrideBucket != null
                ? _selectedPhysicalOverrideBucket!.id
                : null,
          );

      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text('Expense logged successfully!'),
            backgroundColor: Colors.green,
          ),
        );

        // Clear form
        _amountController.clear();
        _descriptionController.clear();
        setState(() {
          _selectedVirtualBucket = null;
          _selectedCategoryBucket = null;
          _selectedPhysicalOverrideBucket = null;
          _showAdvanced = false;
        });

        // Close bottom sheet
        Navigator.of(context).pop();
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
    final expenseState = ref.watch(expenseControllerProvider);

    return Container(
      padding: EdgeInsets.only(
        bottom: MediaQuery.of(context).viewInsets.bottom,
      ),
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Header
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Text(
                  'Log Expense',
                  style: Theme.of(context).textTheme.headlineSmall?.copyWith(
                    fontWeight: FontWeight.bold,
                  ),
                ),
                IconButton(
                  icon: const Icon(Icons.close),
                  onPressed: () => Navigator.of(context).pop(),
                ),
              ],
            ),
            const SizedBox(height: 16),
            // Virtual Bucket Dropdown
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'From Virtual Bucket',
                      style: Theme.of(context).textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                    const SizedBox(height: 8),
                    organizedBucketsAsync.when(
                      data: (organizedBuckets) {
                        final virtualBuckets = organizedBuckets.virtualBuckets;
                        if (virtualBuckets.isEmpty) {
                          return const Text(
                            'No virtual buckets available',
                            style: TextStyle(color: Colors.grey),
                          );
                        }
                        return DropdownButtonFormField<Bucket>(
                          value: _selectedVirtualBucket,
                          decoration: const InputDecoration(
                            border: OutlineInputBorder(),
                            hintText: 'Select virtual bucket',
                          ),
                          items: virtualBuckets.map((bucket) {
                            return DropdownMenuItem<Bucket>(
                              value: bucket,
                              child: Text(bucket.name),
                            );
                          }).toList(),
                          onChanged: (bucket) {
                            setState(() {
                              _selectedVirtualBucket = bucket;
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
            // Category/Expense Dropdown
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'Category/Expense',
                      style: Theme.of(context).textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                    const SizedBox(height: 8),
                    organizedBucketsAsync.when(
                      data: (organizedBuckets) {
                        final expenseBuckets = organizedBuckets.expenseBuckets;
                        if (expenseBuckets.isEmpty) {
                          return const Text(
                            'No expense categories available',
                            style: TextStyle(color: Colors.grey),
                          );
                        }
                        return DropdownButtonFormField<Bucket>(
                          value: _selectedCategoryBucket,
                          decoration: const InputDecoration(
                            border: OutlineInputBorder(),
                            hintText: 'Select expense category',
                          ),
                          items: expenseBuckets.map((bucket) {
                            return DropdownMenuItem<Bucket>(
                              value: bucket,
                              child: Text(bucket.name),
                            );
                          }).toList(),
                          onChanged: (bucket) {
                            setState(() {
                              _selectedCategoryBucket = bucket;
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
                        hintText: 'e.g., Groceries at supermarket',
                      ),
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 16),
            // Advanced: Physical Bucket Override
            Card(
              child: ExpansionTile(
                title: Text(
                  'Advanced: Different Physical Card?',
                  style: Theme.of(context).textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.bold,
                  ),
                ),
                subtitle: const Text(
                  'Use if you paid with a different card than the virtual bucket\'s parent',
                ),
                onExpansionChanged: (expanded) {
                  setState(() {
                    _showAdvanced = expanded;
                    if (!expanded) {
                      _selectedPhysicalOverrideBucket = null;
                    }
                  });
                },
                children: [
                  Padding(
                    padding: const EdgeInsets.all(16),
                    child: organizedBucketsAsync.when(
                      data: (organizedBuckets) {
                        final physicalBuckets =
                            organizedBuckets.physicalBuckets;
                        if (physicalBuckets.isEmpty) {
                          return const Text(
                            'No physical buckets available',
                            style: TextStyle(color: Colors.grey),
                          );
                        }
                        return DropdownButtonFormField<Bucket>(
                          value: _selectedPhysicalOverrideBucket,
                          decoration: const InputDecoration(
                            border: OutlineInputBorder(),
                            hintText: 'Select physical bucket',
                            labelText: 'Actual Physical Bucket Used',
                          ),
                          items: physicalBuckets.map((bucket) {
                            return DropdownMenuItem<Bucket>(
                              value: bucket,
                              child: Text(bucket.name),
                            );
                          }).toList(),
                          onChanged: (bucket) {
                            setState(() {
                              _selectedPhysicalOverrideBucket = bucket;
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
                  ),
                ],
              ),
            ),
            const SizedBox(height: 24),
            // Submit Button
            SizedBox(
              width: double.infinity,
              height: 56,
              child: FilledButton(
                onPressed: _isFormValid && !expenseState.isLoading
                    ? _handleSubmit
                    : null,
                style: FilledButton.styleFrom(
                  backgroundColor: Theme.of(context).colorScheme.primary,
                  foregroundColor: Theme.of(context).colorScheme.onPrimary,
                  shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(12),
                  ),
                ),
                child: expenseState.isLoading
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
                        'Log Expense',
                        style: TextStyle(
                          fontSize: 16,
                          fontWeight: FontWeight.bold,
                        ),
                      ),
              ),
            ),
            const SizedBox(height: 16),
          ],
        ),
      ),
    );
  }
}
