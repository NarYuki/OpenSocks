import 'package:flutter_test/flutter_test.dart';
import 'package:opensocks_mobile/main.dart';

void main() {
  test('traffic units scale correctly', () {
    expect(rate(125000), '1.0 Mbps');
    expect(bytes(1024), '1.0 KiB');
  });
}
