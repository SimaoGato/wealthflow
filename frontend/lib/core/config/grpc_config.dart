/// gRPC connection configuration
///
/// IMPORTANT: For physical devices, use your development machine's IP address, not 'localhost'
///
/// To find your IP:
///   - macOS/Linux: ifconfig | grep "inet " | grep -v 127.0.0.1
///   - Windows: ipconfig (look for IPv4 Address)
///
/// Switch between:
///   - 'localhost' or '127.0.0.1' for emulator/simulator
///   - Your machine's IP (e.g., '192.168.1.17') for physical devices
class GrpcConfig {
  /// Server host address
  ///
  /// Change this based on your target:
  ///   - Emulator/Simulator: 'localhost'
  ///   - Physical Device: Your machine's IP (e.g., '192.168.1.17')
  static const String host =
      '192.168.1.17'; // Detected IP - change to 'localhost' for emulator

  /// Server port
  static const int port = 8080;

  /// Get the full server address for logging
  static String get serverAddress => '$host:$port';

  /// Check if using localhost (will fail on physical devices)
  static bool get isLocalhost => host == 'localhost' || host == '127.0.0.1';
}
