import 'package:flutter/foundation.dart';

/// App configuration
class AppConfig {
  /// Base URL for API requests
  static String get baseURL {
    if (kDebugMode) {
      // Use 10.0.2.2 for Android emulator, localhost for iOS simulator
      return const String.fromEnvironment(
        'MIZU_BASE_URL',
        defaultValue: 'http://10.0.2.2:3000',
      );
    }
    return const String.fromEnvironment(
      'MIZU_BASE_URL',
      defaultValue: 'https://api.example.com',
    );
  }

  /// Request timeout
  static Duration get timeout => const Duration(
        seconds: int.fromEnvironment('MIZU_TIMEOUT', defaultValue: 30),
      );

  /// Enable debug mode
  static bool get debug => kDebugMode;
}
