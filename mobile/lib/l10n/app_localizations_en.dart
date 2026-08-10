// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for English (`en`).
class AppLocalizationsEn extends AppLocalizations {
  AppLocalizationsEn([String locale = 'en']) : super(locale);

  @override
  String get pairHelp =>
      'Enter the URL and token shown under Mobile Pairing in LuCI.';

  @override
  String get routerUrl => 'Router URL';

  @override
  String get pairToken => 'Pairing token';

  @override
  String get connect => 'Connect';

  @override
  String get traffic => 'Traffic';

  @override
  String get test => 'Test';

  @override
  String get account => 'Account';

  @override
  String get authenticating => 'Authenticating';

  @override
  String get switching => 'Switching server';

  @override
  String get connecting => 'Connecting';

  @override
  String get networkConfig => 'Configuring network';

  @override
  String get interfaceProcessing => 'Configuring interface';

  @override
  String get disconnecting => 'Disconnecting';

  @override
  String get connected => 'Connected';

  @override
  String get disconnected => 'Disconnected';

  @override
  String get routingMode => 'Routing mode';

  @override
  String get smartRouting => 'Smart routing';

  @override
  String get smartDescription =>
      'Route only China-bound traffic through OpenSocks';

  @override
  String get globalRouting => 'Full China routing';

  @override
  String get globalDescription => 'Route all LAN web traffic through OpenSocks';

  @override
  String get changingMode => 'Changing routing mode';

  @override
  String get modeChanged => 'Routing mode changed';

  @override
  String get protected => '回国ing';

  @override
  String get notConnected => 'Not connected';

  @override
  String get tapToConnect => 'Tap to connect';

  @override
  String get selectServer => 'Select server';

  @override
  String get routeStopped => 'Network route is stopped';

  @override
  String get routeHealthy => 'Network route is healthy';

  @override
  String get routeRestoring => 'Restoring network route';

  @override
  String get notSignedIn => 'Not signed in';

  @override
  String daysLeft(Object days) {
    return '$days days remaining';
  }

  @override
  String serversCount(int count) {
    return 'Servers ($count)';
  }

  @override
  String historyCount(int count) {
    return 'History ($count)';
  }

  @override
  String get serverList => 'Server list';

  @override
  String get measuringAllPing => 'Measuring latency for all servers…';

  @override
  String get serversUnavailable => 'Could not load servers';

  @override
  String get reconnected => 'Reconnected';

  @override
  String get chinaServiceTraffic => 'Traffic by China service';

  @override
  String cumulativeTraffic(Object up, Object down, Object total) {
    return 'Total ↑ $up / ↓ $down / Σ $total';
  }

  @override
  String get chinaRouteTest => 'China route & IP health test';

  @override
  String get health => 'Health';

  @override
  String get tapToTest => 'Tap to test';

  @override
  String get notSelected => 'Not selected';

  @override
  String get changeServer => 'Change server';

  @override
  String get searchTestNode => 'Search node, region, or carrier';

  @override
  String serverFetchError(Object error) {
    return 'Server error: $error';
  }

  @override
  String get completed => 'Completed';

  @override
  String get email => 'Email';

  @override
  String get phone => 'Phone';

  @override
  String get expires => 'Expires';

  @override
  String get remainingDays => 'Days remaining';

  @override
  String get credentials => 'Credentials';

  @override
  String get saved => 'Saved';

  @override
  String get notSaved => 'Not saved';

  @override
  String get loginManagement => 'Sign-in management';

  @override
  String get username => 'Email / username';

  @override
  String get password => 'Password';

  @override
  String get login => 'Sign in';

  @override
  String get anonymousRegister => 'Register free account';

  @override
  String get logout => 'Sign out';

  @override
  String get freeAccount => 'FREE ACCOUNT';

  @override
  String get vipAccount => 'VIP ACCOUNT';
}
