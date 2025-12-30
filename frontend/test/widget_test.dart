import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:frontend/main.dart';

void main() {
  testWidgets('App loads and shows main layout', (WidgetTester tester) async {
    // Build our app and trigger a frame.
    await tester.pumpWidget(const ProviderScope(child: MyApp()));

    // Verify that the Dashboard screen is shown (text appears in AppBar and bottom nav)
    expect(find.text('Dashboard'), findsAtLeastNWidgets(1));
    expect(find.text('Net Worth'), findsOneWidget);

    // Verify that bottom navigation is present
    expect(find.byIcon(Icons.dashboard), findsOneWidget);
    expect(find.byIcon(Icons.add_chart), findsOneWidget);
    expect(find.byIcon(Icons.checklist), findsOneWidget);
    expect(find.byIcon(Icons.shopping_bag), findsOneWidget);
  });
}
