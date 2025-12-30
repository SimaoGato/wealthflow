import 'package:intl/intl.dart';

/// Formats a decimal string as EUR currency
///
/// Example: formatCurrency("1234.56") returns "€1,234.56"
String formatCurrency(String amount) {
  final value = double.tryParse(amount) ?? 0.0;
  return NumberFormat.simpleCurrency(name: 'EUR').format(value);
}
