import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'dashboard/dashboard_screen.dart';
import '../providers/bucket_provider.dart';

class MainLayout extends ConsumerStatefulWidget {
  const MainLayout({super.key});

  @override
  ConsumerState<MainLayout> createState() => _MainLayoutState();
}

class _MainLayoutState extends ConsumerState<MainLayout> {
  int _currentIndex = 0;

  final List<Widget> _screens = [
    const DashboardScreen(),
    const _InflowPlaceholderScreen(),
    const _PlaceholderScreen(title: 'Money Moves', icon: Icons.checklist),
    const _PlaceholderScreen(title: 'Wants', icon: Icons.shopping_bag),
  ];

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: IndexedStack(index: _currentIndex, children: _screens),
      bottomNavigationBar: BottomNavigationBar(
        currentIndex: _currentIndex,
        onTap: (index) {
          setState(() {
            _currentIndex = index;
          });
        },
        items: const [
          BottomNavigationBarItem(
            icon: Icon(Icons.dashboard),
            label: 'Dashboard',
          ),
          BottomNavigationBarItem(icon: Icon(Icons.add_chart), label: 'Inflow'),
          BottomNavigationBarItem(
            icon: Icon(Icons.checklist),
            label: 'Money Moves',
          ),
          BottomNavigationBarItem(
            icon: Icon(Icons.shopping_bag),
            label: 'Wants',
          ),
        ],
      ),
    );
  }
}

class _InflowPlaceholderScreen extends ConsumerWidget {
  const _InflowPlaceholderScreen();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final organizedBucketsAsync = ref.watch(organizedBucketsProvider);

    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          const Icon(Icons.add_chart, size: 64, color: Colors.grey),
          const SizedBox(height: 16),
          Text(
            'Inflow',
            style: Theme.of(
              context,
            ).textTheme.headlineMedium?.copyWith(color: Colors.grey),
          ),
          const SizedBox(height: 24),
          organizedBucketsAsync.when(
            data: (organizedBuckets) => Text(
              'Income Buckets: ${organizedBuckets.incomeBuckets.length}',
              style: Theme.of(context).textTheme.bodyLarge,
            ),
            loading: () => const CircularProgressIndicator(),
            error: (error, stackTrace) => Text(
              'Error: $error',
              style: Theme.of(
                context,
              ).textTheme.bodyMedium?.copyWith(color: Colors.red),
            ),
          ),
        ],
      ),
    );
  }
}

class _PlaceholderScreen extends StatelessWidget {
  final String title;
  final IconData icon;

  const _PlaceholderScreen({required this.title, required this.icon});

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(icon, size: 64, color: Colors.grey),
          const SizedBox(height: 16),
          Text(
            title,
            style: Theme.of(
              context,
            ).textTheme.headlineMedium?.copyWith(color: Colors.grey),
          ),
        ],
      ),
    );
  }
}
