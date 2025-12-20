/// API error response from server
class APIError {
  final String code;
  final String message;
  final Map<String, dynamic>? details;
  final String? traceId;

  const APIError({
    required this.code,
    required this.message,
    this.details,
    this.traceId,
  });

  factory APIError.fromJson(Map<String, dynamic> json) => APIError(
        code: json['code'] as String,
        message: json['message'] as String,
        details: json['details'] as Map<String, dynamic>?,
        traceId: json['trace_id'] as String?,
      );

  @override
  String toString() => 'APIError($code): $message';
}

/// Mizu client errors
class MizuError implements Exception {
  final String type;
  final String message;
  final Object? cause;

  const MizuError._(this.type, this.message, [this.cause]);

  factory MizuError.invalidResponse() =>
      const MizuError._('invalid_response', 'Invalid server response');

  factory MizuError.http(int statusCode, String body) =>
      MizuError._('http', 'HTTP error $statusCode', body);

  factory MizuError.api(APIError error) =>
      MizuError._('api', error.message, error);

  factory MizuError.network(Object error) =>
      MizuError._('network', 'Network error', error);

  factory MizuError.encoding(Object error) =>
      MizuError._('encoding', 'Encoding error', error);

  factory MizuError.decoding(Object error) =>
      MizuError._('decoding', 'Decoding error', error);

  factory MizuError.unauthorized() =>
      const MizuError._('unauthorized', 'Unauthorized');

  factory MizuError.tokenExpired() =>
      const MizuError._('token_expired', 'Token expired');

  bool get isInvalidResponse => type == 'invalid_response';
  bool get isHttp => type == 'http';
  bool get isApi => type == 'api';
  bool get isNetwork => type == 'network';
  bool get isUnauthorized => type == 'unauthorized';
  bool get isTokenExpired => type == 'token_expired';

  APIError? get apiError => cause is APIError ? cause as APIError : null;

  @override
  String toString() => 'MizuError($type): $message';
}
