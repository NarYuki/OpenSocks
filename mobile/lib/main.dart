import 'dart:async';
import 'dart:math' as math;
import 'package:adaptive_platform_ui/adaptive_platform_ui.dart';
// adaptive_platform_ui does not re-export this iOS 26 scaffold yet.
// ignore: implementation_imports
import 'package:adaptive_platform_ui/src/widgets/ios26/ios26_scaffold.dart';
import 'package:flutter/cupertino.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter/services.dart';
import 'api.dart';
import 'l10n/app_localizations.dart';
import 'connection_store.dart';

extension LocalizedContext on BuildContext {
  AppLocalizations get l10n => AppLocalizations.of(this);
}

void main() => runApp(const OpenSocksApp());

class OpenSocksApp extends StatelessWidget {
  const OpenSocksApp({super.key});
  @override
  Widget build(BuildContext context) => MaterialApp(
    title: 'OpenSocks',
    themeMode: ThemeMode.system,
    debugShowCheckedModeBanner: false,
    localizationsDelegates: const [
      AppLocalizations.delegate,
      GlobalMaterialLocalizations.delegate,
      GlobalCupertinoLocalizations.delegate,
      GlobalWidgetsLocalizations.delegate,
    ],
    supportedLocales: const [Locale('ja'), Locale('en'), Locale('zh', 'CN')],
    theme: ThemeData(
      colorScheme: ColorScheme.fromSeed(
        seedColor: const Color(0xff006c51),
        brightness: Brightness.light,
        surface: const Color(0xfff6faf8),
      ),
      scaffoldBackgroundColor: const Color(0xffeef5f2),
      cardTheme: CardThemeData(
        color: Colors.white,
        elevation: 0,
        margin: const EdgeInsets.symmetric(vertical: 6),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(22),
          side: const BorderSide(color: Color(0xffcedbd6)),
        ),
      ),
      appBarTheme: const AppBarTheme(
        backgroundColor: Colors.transparent,
        surfaceTintColor: Colors.transparent,
        centerTitle: false,
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: Colors.white,
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(18),
          borderSide: const BorderSide(color: Color(0xffb8c9c2)),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(18),
          borderSide: const BorderSide(color: Color(0xffb8c9c2)),
        ),
      ),
      dividerColor: const Color(0xffccd9d4),
      useMaterial3: true,
    ),
    darkTheme: ThemeData(
      colorScheme: ColorScheme.fromSeed(
        seedColor: const Color(0xff22d3ee),
        brightness: Brightness.dark,
        surface: const Color(0xff151821),
      ),
      scaffoldBackgroundColor: const Color(0xff090b10),
      cardTheme: CardThemeData(
        color: const Color(0xff151821),
        elevation: 0,
        margin: const EdgeInsets.symmetric(vertical: 6),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(22),
          side: const BorderSide(color: Color(0xff343946)),
        ),
      ),
      appBarTheme: const AppBarTheme(
        backgroundColor: Colors.transparent,
        surfaceTintColor: Colors.transparent,
        centerTitle: false,
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: const Color(0xff1b1f29),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(18),
          borderSide: const BorderSide(color: Color(0xff434957)),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(18),
          borderSide: const BorderSide(color: Color(0xff434957)),
        ),
      ),
      dividerColor: const Color(0xff343946),
      useMaterial3: true,
    ),
    home: const BootstrapPage(),
  );
}

class BootstrapPage extends StatefulWidget {
  const BootstrapPage({super.key});
  @override
  State<BootstrapPage> createState() => _BootstrapPageState();
}

class _BootstrapPageState extends State<BootstrapPage> {
  static const _nativeChannel = MethodChannel('moe.n4tsu.opensocks/native');
  bool loading = true;
  String? url, token;
  @override
  void initState() {
    super.initState();
    if (defaultTargetPlatform == TargetPlatform.iOS) {
      WidgetsBinding.instance.addPostFrameCallback((_) async {
        try {
          await _nativeChannel.invokeMethod<void>('requestLocalNetworkAccess');
        } on PlatformException {
          // Pairing still works by URL if the native permission request fails.
        }
      });
    }
    ConnectionStore.load().then((x) {
      if (mounted) {
        setState(() {
          url = x.$1;
          token = x.$2;
          loading = false;
        });
      }
    });
  }

  @override
  Widget build(BuildContext c) {
    if (loading) {
      return const Scaffold(body: Center(child: CircularProgressIndicator()));
    }
    if (url == null || token == null) {
      return PairPage(
        onConnected: (u, t) => setState(() {
          url = u;
          token = t;
        }),
      );
    }
    return HomePage(
      api: OpenSocksApi(url!, token!),
      onForget: () async {
        await ConnectionStore.clear();
        setState(() {
          url = null;
          token = null;
        });
      },
    );
  }
}

class PairPage extends StatefulWidget {
  const PairPage({super.key, required this.onConnected});
  final void Function(String, String) onConnected;
  @override
  State<PairPage> createState() => _PairPageState();
}

class _PairPageState extends State<PairPage> {
  final url = TextEditingController(text: 'http://192.168.11.1:9092'),
      token = TextEditingController();
  bool busy = false;
  String? error;
  Future<void> connect() async {
    setState(() {
      busy = true;
      error = null;
    });
    try {
      final u = url.text.trim(), t = token.text.trim();
      await OpenSocksApi(u, t).get('status');
      await ConnectionStore.save(u, t);
      widget.onConnected(u, t);
    } catch (e) {
      setState(() => error = '$e'.replaceFirst('Exception: ', ''));
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  @override
  Widget build(BuildContext c) => Scaffold(
    body: SafeArea(
      child: Center(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 520),
            child: Column(
              children: [
                const Icon(Icons.router, size: 72),
                const SizedBox(height: 16),
                Text('OpenSocks', style: Theme.of(c).textTheme.headlineLarge),
                const SizedBox(height: 8),
                Text(c.l10n.pairHelp),
                const SizedBox(height: 28),
                TextField(
                  controller: url,
                  keyboardType: TextInputType.url,
                  decoration: InputDecoration(
                    labelText: c.l10n.routerUrl,
                    prefixIcon: const Icon(Icons.link),
                    border: const OutlineInputBorder(),
                  ),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: token,
                  obscureText: true,
                  decoration: InputDecoration(
                    labelText: c.l10n.pairToken,
                    prefixIcon: const Icon(Icons.key),
                    border: const OutlineInputBorder(),
                  ),
                ),
                if (error != null)
                  Padding(
                    padding: const EdgeInsets.only(top: 12),
                    child: Text(
                      error!,
                      style: TextStyle(color: Theme.of(c).colorScheme.error),
                    ),
                  ),
                const SizedBox(height: 20),
                FilledButton.icon(
                  onPressed: busy ? null : connect,
                  icon: busy
                      ? const SizedBox.square(
                          dimension: 18,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        )
                      : const Icon(Icons.link),
                  label: Text(c.l10n.connect),
                ),
              ],
            ),
          ),
        ),
      ),
    ),
  );
}

class HomePage extends StatefulWidget {
  const HomePage({super.key, required this.api, required this.onForget});
  final OpenSocksApi api;
  final VoidCallback onForget;
  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  int index = 0;
  int overlayDepth = 0;
  late final ValueNotifier<int> selectedTab;
  late final List<Widget> pages;

  @override
  void initState() {
    super.initState();
    selectedTab = ValueNotifier<int>(0);
    pages = <Widget>[
      OverviewPage(
        api: widget.api,
        selectedTab: selectedTab,
        onOverlayVisibilityChanged: _setOverlayVisible,
      ),
      TrafficPage(api: widget.api, selectedTab: selectedTab),
      TestsPage(
        api: widget.api,
        onOverlayVisibilityChanged: _setOverlayVisible,
      ),
      AccountPage(api: widget.api),
    ];
  }

  @override
  void dispose() {
    selectedTab.dispose();
    widget.api.close();
    super.dispose();
  }

  void _selectTab(int value) {
    if (value == index) return;
    selectedTab.value = value;
    setState(() => index = value);
  }

  void _setOverlayVisible(bool visible) {
    if (!mounted) return;
    setState(() {
      overlayDepth = visible ? overlayDepth + 1 : math.max(0, overlayDepth - 1);
    });
  }

  @override
  Widget build(BuildContext c) {
    final overlayVisible = overlayDepth > 0;
    final body = IndexedStack(index: index, children: pages);
    final isIOS = Theme.of(c).platform == TargetPlatform.iOS;

    if (!isIOS) {
      return Scaffold(
        extendBody: true,
        appBar: AppBar(
          title: const Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text('OpenSocks'),
              Text('CHINA ROUTE CONTROLLER', style: TextStyle(fontSize: 12)),
            ],
          ),
          actions: [
            IconButton(
              onPressed: widget.onForget,
              icon: const Icon(Icons.phonelink_erase),
            ),
          ],
        ),
        body: body,
        bottomNavigationBar: overlayVisible ? null : _androidNavigation(c),
      );
    }

    final adaptiveAppBar = AdaptiveAppBar(
      title: 'OpenSocks',
      subtitle: 'CHINA ROUTE CONTROLLER',
      useNativeToolbar: true,
      actions: [
        AdaptiveAppBarAction(
          onPressed: widget.onForget,
          iosSymbol: 'iphone.slash',
          icon: Icons.phonelink_erase,
        ),
      ],
    );
    final adaptiveBottomBar = AdaptiveBottomNavigationBar(
      useNativeBottomBar: true,
      selectedIndex: index,
      onTap: _selectTab,
      selectedItemColor: Theme.of(c).colorScheme.primary,
      bottomNavigationBar: _androidNavigation(c),
      items: _navigationItems(c),
    );

    if (PlatformInfo.isIOS26OrHigher()) {
      // Use the native scaffold itself so the UIKit status-bar material,
      // translucent blur and scroll-edge gradient remain intact.  Calling it
      // directly also avoids AdaptiveScaffold's selected-index ValueKey, which
      // recreated the entire page whenever the active tab changed.
      return IOS26Scaffold(
        title: adaptiveAppBar.title,
        titleWidget: _iosTitle(c),
        actions: adaptiveAppBar.actions,
        tintColor: Theme.of(c).colorScheme.primary,
        bottomNavigationBar: adaptiveBottomBar,
        tabBarHidden: overlayVisible,
        minimizeBehavior: TabBarMinimizeBehavior.never,
        enableBlur: true,
        children: [
          ColoredBox(color: Theme.of(c).scaffoldBackgroundColor, child: body),
        ],
      );
    }

    return AdaptiveScaffold(
      appBar: adaptiveAppBar,
      body: SafeArea(top: true, bottom: false, child: body),
      tabBarHidden: overlayVisible,
      minimizeBehavior: TabBarMinimizeBehavior.never,
      enableBlur: true,
      bottomNavigationBar: adaptiveBottomBar,
    );
  }

  Widget _iosTitle(BuildContext context) => Column(
    mainAxisSize: MainAxisSize.min,
    crossAxisAlignment: CrossAxisAlignment.center,
    children: [
      Text(
        'OpenSocks',
        style: TextStyle(
          fontSize: 17,
          fontWeight: FontWeight.w600,
          color: CupertinoColors.label.resolveFrom(context),
          decoration: TextDecoration.none,
        ),
      ),
      Text(
        'CHINA ROUTE CONTROLLER',
        style: TextStyle(
          fontSize: 12,
          fontWeight: FontWeight.normal,
          color: CupertinoColors.secondaryLabel.resolveFrom(context),
          decoration: TextDecoration.none,
        ),
      ),
    ],
  );

  List<AdaptiveNavigationDestination> _navigationItems(BuildContext c) => [
    AdaptiveNavigationDestination(
      icon: 'house',
      selectedIcon: 'house.fill',
      label: c.l10n.connect,
    ),
    AdaptiveNavigationDestination(
      icon: 'chart.bar',
      selectedIcon: 'chart.bar.fill',
      label: c.l10n.traffic,
    ),
    AdaptiveNavigationDestination(
      icon: 'speedometer',
      selectedIcon: 'speedometer',
      label: c.l10n.test,
    ),
    AdaptiveNavigationDestination(
      icon: 'person',
      selectedIcon: 'person.fill',
      label: c.l10n.account,
    ),
  ];

  Widget _androidNavigation(BuildContext c) {
    final colors = Theme.of(c).colorScheme;
    return SafeArea(
      minimum: const EdgeInsets.fromLTRB(16, 0, 16, 12),
      child: DecoratedBox(
        decoration: BoxDecoration(
          color: colors.surface,
          borderRadius: BorderRadius.circular(28),
          border: Border.all(color: colors.outlineVariant),
          boxShadow: [
            BoxShadow(
              color: colors.shadow.withValues(alpha: .18),
              blurRadius: 24,
              offset: const Offset(0, 8),
            ),
          ],
        ),
        child: ClipRRect(
          borderRadius: BorderRadius.circular(27),
          child: NavigationBar(
            height: 70,
            backgroundColor: colors.surface,
            indicatorColor: colors.primaryContainer,
            selectedIndex: index,
            onDestinationSelected: _selectTab,
            destinations: [
              NavigationDestination(
                icon: Icon(Icons.power_settings_new_outlined),
                selectedIcon: Icon(Icons.power_settings_new_rounded),
                label: c.l10n.connect,
              ),
              NavigationDestination(
                icon: Icon(Icons.bar_chart_outlined),
                selectedIcon: Icon(Icons.bar_chart_rounded),
                label: c.l10n.traffic,
              ),
              NavigationDestination(
                icon: Icon(Icons.speed_outlined),
                selectedIcon: Icon(Icons.speed_rounded),
                label: c.l10n.test,
              ),
              NavigationDestination(
                icon: Icon(Icons.person_outline_rounded),
                selectedIcon: Icon(Icons.person_rounded),
                label: c.l10n.account,
              ),
            ],
          ),
        ),
      ),
    );
  }
}

double pageTopPadding(BuildContext context, double base) {
  if (Theme.of(context).platform == TargetPlatform.iOS &&
      PlatformInfo.isIOS26OrHigher()) {
    return MediaQuery.paddingOf(context).top + base;
  }
  return base;
}

String bytes(num n) {
  const u = ['B', 'KiB', 'MiB', 'GiB', 'TiB'];
  double v = n.toDouble();
  int i = 0;
  while (v >= 1024 && i < u.length - 1) {
    v /= 1024;
    i++;
  }
  return '${v >= 100 ? v.toStringAsFixed(0) : v.toStringAsFixed(1)} ${u[i]}';
}

String rate(num n) {
  const u = ['bps', 'Kbps', 'Mbps', 'Gbps'];
  double v = n.toDouble() * 8;
  int i = 0;
  while (v >= 1000 && i < u.length - 1) {
    v /= 1000;
    i++;
  }
  return '${v >= 100 ? v.toStringAsFixed(0) : v.toStringAsFixed(1)} ${u[i]}';
}

void toast(BuildContext c, Object v, [bool bad = false]) {
  ScaffoldMessenger.of(c).showSnackBar(
    SnackBar(
      content: Text('$v'.replaceFirst('Exception: ', '')),
      backgroundColor: bad ? Theme.of(c).colorScheme.error : null,
    ),
  );
}

String apiErrorMessage(BuildContext c, Object error) {
  if (error is OpenSocksApiException) {
    return switch (error.kind) {
      OpenSocksApiErrorKind.connection => c.l10n.routerConnectionError,
      OpenSocksApiErrorKind.timeout => c.l10n.routerTimeoutError,
      OpenSocksApiErrorKind.response => c.l10n.invalidResponseError,
      OpenSocksApiErrorKind.server => c.l10n.serverResponseError,
    };
  }
  return c.l10n.serverResponseError;
}

class AppErrorPanel extends StatelessWidget {
  const AppErrorPanel({
    super.key,
    required this.title,
    required this.message,
    required this.onRetry,
    this.icon = Icons.error_outline_rounded,
  });

  final String title;
  final String message;
  final VoidCallback onRetry;
  final IconData icon;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    return Card(
      color: colors.errorContainer,
      child: Padding(
        padding: const EdgeInsets.all(18),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, size: 34, color: colors.onErrorContainer),
            const SizedBox(height: 10),
            Text(
              title,
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.titleMedium?.copyWith(
                color: colors.onErrorContainer,
                fontWeight: FontWeight.w800,
              ),
            ),
            const SizedBox(height: 6),
            Text(
              message,
              textAlign: TextAlign.center,
              style: TextStyle(color: colors.onErrorContainer),
            ),
            const SizedBox(height: 12),
            FilledButton.tonalIcon(
              onPressed: onRetry,
              icon: const Icon(Icons.refresh_rounded),
              label: Text(context.l10n.retry),
            ),
          ],
        ),
      ),
    );
  }
}

class OverviewPage extends StatefulWidget {
  const OverviewPage({
    super.key,
    required this.api,
    required this.selectedTab,
    required this.onOverlayVisibilityChanged,
  });
  final OpenSocksApi api;
  final ValueListenable<int> selectedTab;
  final ValueChanged<bool> onOverlayVisibilityChanged;
  @override
  State<OverviewPage> createState() => _OverviewPageState();
}

class _OverviewPageState extends State<OverviewPage> {
  Map<String, dynamic>? s, t;
  Timer? timer;
  bool trafficPending = false;
  bool busy = false;
  String operation = '';
  @override
  void initState() {
    super.initState();
    refresh();
    timer = Timer.periodic(const Duration(seconds: 1), (_) {
      if (widget.selectedTab.value == 0) traffic();
    });
  }

  @override
  void dispose() {
    timer?.cancel();
    super.dispose();
  }

  Future<void> refresh() async {
    try {
      final x = await widget.api.get('status');
      if (mounted) setState(() => s = x);
    } catch (e) {
      if (mounted) toast(context, e, true);
    }
  }

  Future<void> traffic() async {
    if (trafficPending) return;
    trafficPending = true;
    try {
      final x = await widget.api.get('traffic');
      if (mounted) setState(() => t = x);
    } catch (_) {
      // The next active-tab tick retries.
    } finally {
      trafficPending = false;
    }
  }

  Future<void> refreshUntilSettled({required bool expectRunning}) async {
    for (var attempt = 0; attempt < 20; attempt++) {
      final value = await widget.api.get('status');
      if (!mounted) return;
      setState(() => s = value);
      final running = value['running'] == true;
      final routed = value['routingApplied'] == true;
      if (running == expectRunning && (!expectRunning || routed)) return;
      await Future<void>.delayed(const Duration(milliseconds: 400));
    }
  }

  Future<void> action(String p, [Map<String, dynamic>? b]) async {
    final connecting = p == 'connect';
    final switching = connecting && s?['running'] == true;
    final l = context.l10n;
    final stages = connecting
        ? [
            l.authenticating,
            switching ? l.switching : l.connecting,
            l.networkConfig,
            l.interfaceProcessing,
          ]
        : [l.disconnecting, l.interfaceProcessing];
    var stage = 0;
    setState(() {
      busy = true;
      operation = stages.first;
    });
    final progress = Timer.periodic(const Duration(milliseconds: 700), (_) {
      if (!mounted || stage >= stages.length - 1) {
        return;
      }
      setState(() => operation = stages[++stage]);
    });
    try {
      await widget.api.post(p, b);
      await refreshUntilSettled(expectRunning: connecting);
      if (mounted) {
        setState(() => operation = connecting ? l.connected : l.disconnected);
        toast(context, operation);
        await Future<void>.delayed(const Duration(milliseconds: 900));
      }
    } catch (e) {
      if (mounted) toast(context, e, true);
    } finally {
      progress.cancel();
      if (mounted) {
        setState(() {
          busy = false;
          operation = '';
        });
      }
    }
  }

  Future<void> chooseServer() async {
    widget.onOverlayVisibilityChanged(true);
    await WidgetsBinding.instance.endOfFrame;
    if (!mounted) {
      widget.onOverlayVisibilityChanged(false);
      return;
    }
    int? selected;
    try {
      selected = await showModalBottomSheet<int>(
        context: context,
        isScrollControlled: true,
        useSafeArea: true,
        showDragHandle: true,
        builder: (sheetContext) => FractionallySizedBox(
          heightFactor: .94,
          child: LinesPage(
            api: widget.api,
            onSelect: (id) => Navigator.pop(sheetContext, id),
          ),
        ),
      );
    } finally {
      widget.onOverlayVisibilityChanged(false);
    }
    if (selected != null && mounted) {
      await action('connect', {'line_id': selected});
    }
  }

  Future<void> chooseRoutingMode() async {
    if (busy || s == null) return;
    final currentMode = s!['mode'] == 'global' ? 'global' : 'smart';
    final currentSessions = (s!['sessionCount'] as num?)?.toInt() ?? 1;
    widget.onOverlayVisibilityChanged(true);
    await WidgetsBinding.instance.endOfFrame;
    if (!mounted) {
      widget.onOverlayVisibilityChanged(false);
      return;
    }
    Map<String, dynamic>? selected;
    var selectedMode = currentMode;
    var selectedSessions = currentSessions;
    try {
      selected = await showModalBottomSheet<Map<String, dynamic>>(
        context: context,
        useSafeArea: true,
        showDragHandle: true,
        builder: (sheetContext) => StatefulBuilder(
          builder: (sheetContext, setSheetState) {
            return SafeArea(
              child: SingleChildScrollView(
                child: Padding(
                  padding: const EdgeInsets.fromLTRB(16, 0, 16, 18),
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      Text(
                        sheetContext.l10n.routingMode,
                        style: Theme.of(sheetContext).textTheme.titleLarge
                            ?.copyWith(fontWeight: FontWeight.w800),
                      ),
                      const SizedBox(height: 8),
                      ListTile(
                        selected: selectedMode == 'smart',
                        title: Text(sheetContext.l10n.smartRouting),
                        subtitle: Text(sheetContext.l10n.smartDescription),
                        leading: const Icon(Icons.alt_route_rounded),
                        trailing: selectedMode == 'smart'
                            ? const Icon(Icons.check_circle_rounded)
                            : null,
                        onTap: () =>
                            setSheetState(() => selectedMode = 'smart'),
                      ),
                      ListTile(
                        selected: selectedMode == 'global',
                        title: Text(sheetContext.l10n.globalRouting),
                        subtitle: Text(sheetContext.l10n.globalDescription),
                        leading: const Icon(Icons.public_rounded),
                        trailing: selectedMode == 'global'
                            ? const Icon(Icons.check_circle_rounded)
                            : null,
                        onTap: () =>
                            setSheetState(() => selectedMode = 'global'),
                      ),
                      const Divider(),
                      Text(
                        sheetContext.l10n.sessionMode,
                        style: Theme.of(sheetContext).textTheme.titleMedium
                            ?.copyWith(fontWeight: FontWeight.w800),
                      ),
                      for (final option in [
                        (1, sheetContext.l10n.singleMode),
                        (2, sheetContext.l10n.dualMode),
                        (3, sheetContext.l10n.tripleMode),
                      ])
                        ListTile(
                          selected: selectedSessions == option.$1,
                          leading: Icon(
                            option.$1 == 1
                                ? Icons.looks_one_rounded
                                : option.$1 == 2
                                ? Icons.looks_two_rounded
                                : Icons.looks_3_rounded,
                          ),
                          title: Text(option.$2),
                          trailing: selectedSessions == option.$1
                              ? const Icon(Icons.check_circle_rounded)
                              : null,
                          onTap: () =>
                              setSheetState(() => selectedSessions = option.$1),
                        ),
                      const SizedBox(height: 8),
                      FilledButton(
                        onPressed: () => Navigator.pop(sheetContext, {
                          'mode': selectedMode,
                          'sessions': selectedSessions,
                        }),
                        child: Text(sheetContext.l10n.apply),
                      ),
                    ],
                  ),
                ),
              ),
            );
          },
        ),
      );
      if (selected == null || !mounted) return;
      final mode = selected['mode'] as String;
      final sessions = selected['sessions'] as int;
      if (mode == currentMode && sessions == currentSessions) return;
      await applyRoutingMode(mode, sessions);
    } finally {
      widget.onOverlayVisibilityChanged(false);
    }
  }

  Future<void> applyRoutingMode(String mode, int sessionCount) async {
    final current = s!;
    final l = context.l10n;
    setState(() {
      busy = true;
      operation = l.changingMode;
    });
    var stage = 0;
    final stages = [l.changingMode, l.networkConfig, l.interfaceProcessing];
    final progress = Timer.periodic(const Duration(milliseconds: 700), (_) {
      if (!mounted || stage >= stages.length - 1) {
        return;
      }
      setState(() => operation = stages[++stage]);
    });
    try {
      await widget.api.post('settings', {
        'mode': mode,
        'tun': current['tun'] == true,
        'free_only': current['freeOnly'] == true,
        'auto_connect': current['autoConnect'] == true,
        'auto_route': current['autoRoute'] == true,
        'session_count': sessionCount,
        'region': current['region'] ?? '',
        'exclude_regions': current['excludeRegions'] ?? '',
        'include_domains': current['includeDomains'] ?? '',
        'exclude_domains': current['excludeDomains'] ?? '',
        'include_cidrs': current['includeCIDRs'] ?? '',
        'exclude_cidrs': current['excludeCIDRs'] ?? '',
      });
      await refreshUntilSettled(expectRunning: current['running'] == true);
      if (mounted) toast(context, l.modeChanged);
    } catch (e) {
      if (mounted) toast(context, e, true);
    } finally {
      progress.cancel();
      if (mounted) {
        setState(() {
          busy = false;
          operation = '';
        });
      }
    }
  }

  @override
  Widget build(BuildContext c) {
    final x = s;
    if (x == null) return const Center(child: CircularProgressIndicator());
    return RefreshIndicator(
      onRefresh: refresh,
      child: ListView(
        padding: EdgeInsets.fromLTRB(16, pageTopPadding(c, 16), 16, 16),
        children: [
          Center(
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 7),
              decoration: BoxDecoration(
                color:
                    (x['running'] == true
                            ? Theme.of(c).colorScheme.primary
                            : Theme.of(c).colorScheme.onSurfaceVariant)
                        .withValues(alpha: .12),
                borderRadius: BorderRadius.circular(30),
              ),
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Container(
                    width: 8,
                    height: 8,
                    decoration: BoxDecoration(
                      shape: BoxShape.circle,
                      color: x['running'] == true
                          ? Theme.of(c).colorScheme.primary
                          : Theme.of(c).colorScheme.onSurfaceVariant,
                    ),
                  ),
                  const SizedBox(width: 8),
                  Text(
                    x['running'] == true
                        ? c.l10n.protected
                        : c.l10n.notConnected,
                    style: const TextStyle(fontWeight: FontWeight.w700),
                  ),
                ],
              ),
            ),
          ),
          const SizedBox(height: 28),
          Center(
            child: GestureDetector(
              onTap: busy
                  ? null
                  : () => action(
                      x['running'] == true ? 'disconnect' : 'connect',
                      x['running'] == true ? null : {'line_id': -1},
                    ),
              child: AnimatedContainer(
                duration: const Duration(milliseconds: 250),
                width: 216,
                height: 216,
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  color: (x['running'] == true ? Colors.green : Colors.blueGrey)
                      .withValues(alpha: .16),
                  gradient: RadialGradient(
                    colors: x['running'] == true
                        ? [
                            Theme.of(c).colorScheme.primaryContainer,
                            Theme.of(c).colorScheme.surface,
                          ]
                        : [
                            Theme.of(c).colorScheme.surfaceContainerHighest,
                            Theme.of(c).colorScheme.surface,
                          ],
                  ),
                  border: Border.all(
                    color: x['running'] == true
                        ? Theme.of(c).colorScheme.primary
                        : Theme.of(c).colorScheme.outline,
                    width: 3,
                  ),
                  boxShadow: [
                    BoxShadow(
                      color:
                          (x['running'] == true
                                  ? Theme.of(c).colorScheme.primary
                                  : Theme.of(c).colorScheme.shadow)
                              .withValues(alpha: .22),
                      blurRadius: 42,
                      spreadRadius: 6,
                    ),
                  ],
                ),
                child: busy
                    ? const Padding(
                        padding: EdgeInsets.all(80),
                        child: CircularProgressIndicator(strokeWidth: 4),
                      )
                    : Icon(
                        Icons.power_settings_new,
                        size: 88,
                        color: x['running'] == true
                            ? Theme.of(c).colorScheme.primary
                            : Theme.of(c).colorScheme.onSurfaceVariant,
                      ),
              ),
            ),
          ),
          const SizedBox(height: 22),
          Center(
            child: Text(
              busy
                  ? operation
                  : (x['running'] == true
                        ? c.l10n.connected
                        : c.l10n.tapToConnect),
              style: Theme.of(
                c,
              ).textTheme.headlineSmall?.copyWith(fontWeight: FontWeight.w800),
            ),
          ),
          const SizedBox(height: 6),
          Center(
            child: TextButton.icon(
              onPressed: busy ? null : chooseServer,
              icon: const Icon(Icons.dns_rounded, size: 18),
              label: Text(x['lineName'] ?? c.l10n.selectServer),
            ),
          ),
          const SizedBox(height: 28),
          Card(
            child: InkWell(
              onTap: busy ? null : chooseRoutingMode,
              borderRadius: BorderRadius.circular(22),
              child: Padding(
                padding: const EdgeInsets.all(18),
                child: Row(
                  children: [
                    Container(
                      width: 46,
                      height: 46,
                      decoration: BoxDecoration(
                        color: Theme.of(
                          c,
                        ).colorScheme.primary.withValues(alpha: .12),
                        borderRadius: BorderRadius.circular(14),
                      ),
                      child: const Icon(Icons.route_rounded),
                    ),
                    const SizedBox(width: 14),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            '${x['mode'] == 'global' ? c.l10n.globalRouting : c.l10n.smartRouting} · ${(x['sessionCount'] ?? 1) == 3
                                ? c.l10n.tripleMode
                                : (x['sessionCount'] ?? 1) == 2
                                ? c.l10n.dualMode
                                : c.l10n.singleMode}',
                            style: const TextStyle(fontWeight: FontWeight.w700),
                          ),
                          const SizedBox(height: 3),
                          Text(
                            x['running'] != true
                                ? c.l10n.routeStopped
                                : x['routingApplied'] == true
                                ? c.l10n.routeHealthy
                                : c.l10n.routeRestoring,
                            style: TextStyle(
                              fontSize: 12,
                              color: x['running'] != true
                                  ? Theme.of(c).colorScheme.onSurfaceVariant
                                  : x['routingApplied'] == true
                                  ? Theme.of(c).colorScheme.primary
                                  : Theme.of(c).colorScheme.tertiary,
                            ),
                          ),
                        ],
                      ),
                    ),
                    Icon(
                      x['running'] != true
                          ? Icons.pause_circle_outline_rounded
                          : x['routingApplied'] == true
                          ? Icons.check_circle
                          : Icons.sync,
                      color: x['running'] != true
                          ? Theme.of(c).colorScheme.onSurfaceVariant
                          : x['routingApplied'] == true
                          ? Theme.of(c).colorScheme.primary
                          : Theme.of(c).colorScheme.tertiary,
                    ),
                  ],
                ),
              ),
            ),
          ),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(18),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.spaceAround,
                children: [
                  _metric(
                    'UPLOAD',
                    rate(t?['up_bps'] ?? 0),
                    Icons.north_rounded,
                  ),
                  Container(
                    width: 1,
                    height: 50,
                    color: Theme.of(c).dividerColor,
                  ),
                  _metric(
                    'DOWNLOAD',
                    rate(t?['down_bps'] ?? 0),
                    Icons.south_rounded,
                  ),
                  Container(
                    width: 1,
                    height: 50,
                    color: Theme.of(c).dividerColor,
                  ),
                  _metric(
                    'TOTAL',
                    bytes(t?['total_bytes'] ?? 0),
                    Icons.data_usage,
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}

Widget _metric(String a, String b, IconData i) => Column(
  children: [
    Icon(i),
    const SizedBox(height: 5),
    Padding(
      padding: const EdgeInsets.symmetric(horizontal: 3),
      child: FittedBox(
        fit: BoxFit.scaleDown,
        child: Text(b, style: const TextStyle(fontWeight: FontWeight.bold)),
      ),
    ),
    Text(a, style: const TextStyle(fontSize: 12)),
  ],
);

class AccountCard extends StatelessWidget {
  const AccountCard({super.key, this.account});
  final Map<String, dynamic>? account;
  @override
  Widget build(BuildContext c) => Card(
    child: ListTile(
      leading: const Icon(Icons.account_circle),
      title: Text(account?['nick'] ?? c.l10n.notSignedIn),
      subtitle: Text(
        account == null
            ? '-'
            : '${account!['email'] ?? account!['phone'] ?? ''} · ${c.l10n.daysLeft('${account!['remaining_days'] ?? '-'}')}',
      ),
    ),
  );
}

class LinesPage extends StatefulWidget {
  const LinesPage({super.key, required this.api, this.onSelect});
  final OpenSocksApi api;
  final ValueChanged<int>? onSelect;
  @override
  State<LinesPage> createState() => _LinesPageState();
}

class _LinesPageState extends State<LinesPage> {
  List<dynamic> lines = [], history = [];
  bool loading = false;
  int? currentID, switchingID;
  String operation = '';
  @override
  void initState() {
    super.initState();
    load();
  }

  Future<void> load() async {
    setState(() => loading = true);
    try {
      final result = await Future.wait([
        widget.api.get('lines?sort=ping', timeout: 90),
        widget.api.get('history'),
        widget.api.get('status'),
      ]);
      if (mounted) {
        setState(() {
          lines = result[0]['lines'] ?? [];
          history = result[1]['history'] ?? [];
          currentID = result[2]['lineID'];
        });
      }
    } catch (e) {
      if (mounted) toast(context, e, true);
    } finally {
      if (mounted) setState(() => loading = false);
    }
  }

  Future<void> connect(dynamic id) async {
    final switching = currentID != null;
    final l = context.l10n;
    final stages = [
      l.authenticating,
      switching ? l.switching : l.connecting,
      l.networkConfig,
      l.interfaceProcessing,
    ];
    var stage = 0;
    setState(() {
      switchingID = id as int?;
      operation = stages.first;
    });
    final progress = Timer.periodic(const Duration(milliseconds: 700), (_) {
      if (!mounted || stage >= stages.length - 1) {
        return;
      }
      setState(() => operation = stages[++stage]);
    });
    try {
      final result = await widget.api.post('connect', {'line_id': id});
      final h = await widget.api.get('history');
      if (mounted) {
        setState(() {
          currentID = result['lineID'];
          history = h['history'] ?? history;
        });
      }
      if (mounted) {
        setState(() => operation = l.connected);
        toast(context, l.connected);
        await Future<void>.delayed(const Duration(milliseconds: 900));
      }
    } catch (e) {
      if (mounted) toast(context, e, true);
    } finally {
      progress.cancel();
      if (mounted) {
        setState(() {
          switchingID = null;
          operation = '';
        });
      }
    }
  }

  @override
  Widget build(BuildContext c) => DefaultTabController(
    length: 2,
    child: Column(
      children: [
        TabBar(
          dividerColor: Colors.transparent,
          tabs: [
            Tab(text: c.l10n.serversCount(lines.length)),
            Tab(text: c.l10n.historyCount(history.length)),
          ],
        ),
        if (switchingID != null)
          Container(
            width: double.infinity,
            margin: const EdgeInsets.fromLTRB(16, 10, 16, 2),
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
            decoration: BoxDecoration(
              color: Theme.of(c).colorScheme.primaryContainer,
              borderRadius: BorderRadius.circular(16),
            ),
            child: Row(
              children: [
                const SizedBox.square(
                  dimension: 18,
                  child: CircularProgressIndicator(strokeWidth: 2),
                ),
                const SizedBox(width: 12),
                Text(
                  operation,
                  style: const TextStyle(fontWeight: FontWeight.w700),
                ),
              ],
            ),
          ),
        Expanded(
          child: TabBarView(
            children: [
              RefreshIndicator(
                onRefresh: load,
                child: ListView(
                  padding: const EdgeInsets.fromLTRB(16, 10, 16, 24),
                  children: [
                    Row(
                      children: [
                        Expanded(
                          child: Text(
                            c.l10n.serverList,
                            style: TextStyle(
                              fontSize: 18,
                              fontWeight: FontWeight.w800,
                            ),
                          ),
                        ),
                        IconButton(
                          onPressed: loading ? null : load,
                          icon: const Icon(Icons.refresh_rounded),
                        ),
                      ],
                    ),
                    if (loading) ...[
                      const SizedBox(height: 10),
                      const LinearProgressIndicator(
                        borderRadius: BorderRadius.all(Radius.circular(8)),
                      ),
                      Padding(
                        padding: EdgeInsets.symmetric(vertical: 8),
                        child: Text(
                          c.l10n.measuringAllPing,
                          style: TextStyle(
                            fontSize: 12,
                            color: Theme.of(c).colorScheme.onSurfaceVariant,
                          ),
                        ),
                      ),
                    ],
                    const SizedBox(height: 6),
                    if (!loading && lines.isEmpty)
                      Padding(
                        padding: const EdgeInsets.all(40),
                        child: Center(child: Text(c.l10n.serversUnavailable)),
                      ),
                    ...lines.map((l) {
                      final latency = (l['latency_ms'] as num?)?.toDouble();
                      final selected = currentID == l['id'];
                      final switching = switchingID == l['id'];
                      final pingColor = latency == null
                          ? Theme.of(c).colorScheme.outline
                          : latency < 100
                          ? Theme.of(c).colorScheme.primary
                          : latency < 220
                          ? (Theme.of(c).brightness == Brightness.dark
                                ? const Color(0xffffc107)
                                : const Color(0xffa85d00))
                          : Theme.of(c).colorScheme.error;
                      return Card(
                        color: selected
                            ? Theme.of(c).colorScheme.primaryContainer
                            : null,
                        child: Padding(
                          padding: const EdgeInsets.symmetric(
                            horizontal: 6,
                            vertical: 5,
                          ),
                          child: ListTile(
                            leading: Container(
                              width: 46,
                              height: 46,
                              decoration: BoxDecoration(
                                color: pingColor.withValues(alpha: .12),
                                borderRadius: BorderRadius.circular(15),
                              ),
                              child: Icon(
                                Icons.cell_tower_rounded,
                                color: pingColor,
                              ),
                            ),
                            title: Row(
                              children: [
                                Expanded(
                                  child: Text(
                                    '${l['name'] ?? '#${l['id']}'}',
                                    maxLines: 1,
                                    overflow: TextOverflow.ellipsis,
                                    style: const TextStyle(
                                      fontWeight: FontWeight.w700,
                                    ),
                                  ),
                                ),
                                if (selected)
                                  Icon(
                                    Icons.check_circle,
                                    size: 18,
                                    color: Theme.of(c).colorScheme.primary,
                                  ),
                              ],
                            ),
                            subtitle: Padding(
                              padding: const EdgeInsets.only(top: 5),
                              child: Text(
                                '${l['location'] ?? '-'}  •  ${l['isFree'] == true ? 'FREE' : 'VIP'}',
                                style: TextStyle(
                                  fontSize: 12,
                                  color: Theme.of(
                                    c,
                                  ).colorScheme.onSurfaceVariant,
                                ),
                              ),
                            ),
                            trailing: SizedBox(
                              width: 72,
                              child: Column(
                                mainAxisAlignment: MainAxisAlignment.center,
                                crossAxisAlignment: CrossAxisAlignment.end,
                                children: [
                                  if (switching)
                                    const SizedBox.square(
                                      dimension: 20,
                                      child: CircularProgressIndicator(
                                        strokeWidth: 2,
                                      ),
                                    )
                                  else
                                    Text(
                                      latency == null
                                          ? '—'
                                          : '${latency.toStringAsFixed(0)} ms',
                                      style: TextStyle(
                                        color: pingColor,
                                        fontWeight: FontWeight.w800,
                                      ),
                                    ),
                                  if (!switching)
                                    Text(
                                      selected
                                          ? c.l10n.connecting
                                          : c.l10n.connect,
                                      style: TextStyle(
                                        fontSize: 11,
                                        color: Theme.of(
                                          c,
                                        ).colorScheme.onSurfaceVariant,
                                      ),
                                    ),
                                ],
                              ),
                            ),
                            onTap: switchingID != null || selected
                                ? null
                                : () {
                                    final id = l['id'] as int;
                                    if (widget.onSelect != null) {
                                      widget.onSelect!(id);
                                    } else {
                                      connect(id);
                                    }
                                  },
                          ),
                        ),
                      );
                    }),
                  ],
                ),
              ),
              ListView(
                padding: const EdgeInsets.all(12),
                children: history
                    .map(
                      (h) => Card(
                        child: ListTile(
                          title: Text(h['line_name'] ?? '#${h['line_id']}'),
                          subtitle: Text(
                            '${h['location'] ?? '-'} · ${h['connected_at'] ?? ''}',
                          ),
                          trailing: IconButton(
                            icon: const Icon(Icons.replay),
                            onPressed: () async {
                              try {
                                await widget.api.post('reconnect', {
                                  'id': h['id'],
                                });
                                if (mounted) {
                                  toast(context, context.l10n.reconnected);
                                }
                              } catch (e) {
                                if (mounted) toast(context, e, true);
                              }
                            },
                          ),
                        ),
                      ),
                    )
                    .toList(),
              ),
            ],
          ),
        ),
      ],
    ),
  );
}

class TrafficPage extends StatefulWidget {
  const TrafficPage({super.key, required this.api, required this.selectedTab});
  final OpenSocksApi api;
  final ValueListenable<int> selectedTab;
  @override
  State<TrafficPage> createState() => _TrafficPageState();
}

class _TrafficPageState extends State<TrafficPage> {
  Map<String, dynamic>? data;
  Object? loadError;
  Timer? timer;
  bool loadPending = false;
  @override
  void initState() {
    super.initState();
    load();
    timer = Timer.periodic(const Duration(seconds: 1), (_) {
      if (widget.selectedTab.value == 1) load();
    });
  }

  @override
  void dispose() {
    timer?.cancel();
    super.dispose();
  }

  Future<void> load() async {
    if (loadPending) return;
    loadPending = true;
    try {
      final x = await widget.api.get('traffic');
      if (mounted) {
        setState(() {
          data = x;
          loadError = null;
        });
      }
    } catch (error) {
      if (mounted) setState(() => loadError = error);
    } finally {
      loadPending = false;
    }
  }

  @override
  Widget build(BuildContext c) {
    final d = data;
    if (d == null && loadError != null) {
      return ListView(
        padding: EdgeInsets.fromLTRB(16, pageTopPadding(c, 16), 16, 16),
        children: [
          AppErrorPanel(
            title: c.l10n.trafficErrorTitle,
            message:
                '${c.l10n.trafficErrorMessage}\n${apiErrorMessage(c, loadError!)}',
            onRetry: load,
            icon: Icons.bar_chart_rounded,
          ),
        ],
      );
    }
    if (d == null) return const Center(child: CircularProgressIndicator());
    final services =
        (d['services'] is Map
                ? Map<String, dynamic>.from(d['services'])
                : <String, dynamic>{})
            .entries
            .toList()
          ..sort(
            (a, b) => ((b.value['total_bytes'] ?? 0) as num).compareTo(
              (a.value['total_bytes'] ?? 0) as num,
            ),
          );
    final theme = Theme.of(c);
    final colors = theme.colorScheme;
    final panelColor = theme.brightness == Brightness.dark
        ? const Color(0xff151821)
        : Colors.white;
    final panelBorder = theme.brightness == Brightness.dark
        ? const Color(0xff343946)
        : const Color(0xffcedbd6);
    return ListView(
      padding: EdgeInsets.fromLTRB(12, pageTopPadding(c, 12), 12, 12),
      children: [
        if (loadError != null)
          AppErrorPanel(
            title: c.l10n.trafficErrorTitle,
            message: apiErrorMessage(c, loadError!),
            onRetry: load,
            icon: Icons.bar_chart_rounded,
          ),
        DecoratedBox(
          decoration: BoxDecoration(
            color: panelColor,
            borderRadius: BorderRadius.circular(22),
            border: Border.all(color: panelBorder),
          ),
          child: Padding(
            padding: const EdgeInsets.all(18),
            child: Column(
              children: [
                Text(
                  '↑ ${rate(d['up_bps'] ?? 0)}  /  ↓ ${rate(d['down_bps'] ?? 0)}',
                  style: theme.textTheme.titleLarge?.copyWith(
                    color: colors.onSurface,
                  ),
                ),
                const SizedBox(height: 8),
                Text(
                  c.l10n.cumulativeTraffic(
                    bytes(d['up_bytes'] ?? 0),
                    bytes(d['down_bytes'] ?? 0),
                    bytes(d['total_bytes'] ?? 0),
                  ),
                  style: theme.textTheme.bodyMedium?.copyWith(
                    color: colors.onSurfaceVariant,
                  ),
                ),
              ],
            ),
          ),
        ),
        Padding(
          padding: const EdgeInsets.fromLTRB(8, 22, 8, 10),
          child: Text(
            c.l10n.chinaServiceTraffic,
            style: theme.textTheme.titleMedium?.copyWith(
              color: colors.onSurface,
              fontWeight: FontWeight.w700,
            ),
          ),
        ),
        DecoratedBox(
          decoration: BoxDecoration(
            color: panelColor,
            borderRadius: BorderRadius.circular(22),
            border: Border.all(color: panelBorder),
          ),
          child: ClipRRect(
            borderRadius: BorderRadius.circular(21),
            child: Column(
              children: [
                for (var i = 0; i < services.length; i++) ...[
                  _TrafficServiceRow(
                    name: switch (services[i].key) {
                      'honor_of_kings' => c.l10n.serviceHonorOfKings,
                      'other_china' => c.l10n.serviceOtherChina,
                      _ => services[i].key.replaceAll('_', ' '),
                    },
                    up: bytes(services[i].value['up_bytes'] ?? 0),
                    down: bytes(services[i].value['down_bytes'] ?? 0),
                    total: bytes(services[i].value['total_bytes'] ?? 0),
                  ),
                  if (i != services.length - 1)
                    Divider(height: 1, indent: 16, endIndent: 16),
                ],
              ],
            ),
          ),
        ),
      ],
    );
  }
}

class _TrafficServiceRow extends StatelessWidget {
  const _TrafficServiceRow({
    required this.name,
    required this.up,
    required this.down,
    required this.total,
  });

  final String name;
  final String up;
  final String down;
  final String total;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;
    return RepaintBoundary(
      child: ColoredBox(
        color: theme.brightness == Brightness.dark
            ? const Color(0xff151821)
            : Colors.white,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
          child: Row(
            children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      name,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: theme.textTheme.titleMedium?.copyWith(
                        color: colors.onSurface,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    const SizedBox(height: 4),
                    Text(
                      '↑ $up   ↓ $down',
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: theme.textTheme.bodyMedium?.copyWith(
                        color: colors.onSurfaceVariant,
                      ),
                    ),
                  ],
                ),
              ),
              const SizedBox(width: 12),
              Text(
                total,
                style: theme.textTheme.titleSmall?.copyWith(
                  color: colors.onSurface,
                  fontWeight: FontWeight.w700,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class TestsPage extends StatefulWidget {
  const TestsPage({
    super.key,
    required this.api,
    required this.onOverlayVisibilityChanged,
  });
  final OpenSocksApi api;
  final ValueChanged<bool> onOverlayVisibilityChanged;
  @override
  State<TestsPage> createState() => _TestsPageState();
}

class _TestsPageState extends State<TestsPage> {
  Map<String, dynamic>? region, progress;
  List<dynamic> ookla = [], cn = [];
  Object? regionError, ooklaError, cnError;
  String? ooklaID, cnID;
  bool busy = false;
  bool pollingProgress = false;
  Timer? progressTimer;
  @override
  void dispose() {
    progressTimer?.cancel();
    super.dispose();
  }

  Future<void> regionTest() async {
    setState(() => busy = true);
    try {
      final x = await widget.api.get('regiontest', timeout: 45);
      if (mounted) {
        setState(() {
          region = x;
          regionError = null;
        });
      }
    } catch (e) {
      if (mounted) setState(() => regionError = e);
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  Future<void> servers(String type) async {
    setState(() => busy = true);
    try {
      final x = await widget.api.get(
        type == 'ookla' ? 'speedtest/servers' : 'speedtestcn/servers',
      );
      if (mounted) {
        setState(() {
          if (type == 'ookla') {
            ookla = x['servers'] ?? [];
            ooklaError = null;
            ooklaID = ookla.isEmpty ? null : '${ookla.first['id']}';
          } else {
            cn = x['servers'] ?? [];
            cnError = null;
            cnID = cn.isEmpty ? null : '${cn.first['id']}';
          }
        });
      }
    } catch (e) {
      if (mounted) {
        setState(() {
          if (type == 'ookla') {
            ooklaError = e;
          } else {
            cnError = e;
          }
        });
      }
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  Future<void> run(String type) async {
    final id = type == 'ookla' ? ooklaID : cnID;
    if (id == null) return;
    setState(() => busy = true);
    try {
      await widget.api.post('speedtest/job/start', {
        'provider': type == 'ookla' ? 'ookla' : 'speedtestcn',
        'id': id,
      });
      progressTimer?.cancel();
      progressTimer = Timer.periodic(
        const Duration(milliseconds: 150),
        (_) => pollProgress(),
      );
      await pollProgress();
    } catch (e) {
      if (mounted) toast(context, e, true);
    }
  }

  Future<void> pollProgress() async {
    if (pollingProgress) return;
    pollingProgress = true;
    try {
      final x = await widget.api.get('speedtest/job/status');
      if (!mounted) return;
      setState(() {
        progress = x;
        busy = x['running'] == true;
      });
      if (x['running'] != true) {
        progressTimer?.cancel();
        if (x['error'] != null && x['error'].toString().isNotEmpty) {
          toast(context, x['error'], true);
        }
      }
    } catch (_) {
      // A later serialized poll will recover transient LAN latency.
    } finally {
      pollingProgress = false;
    }
  }

  @override
  Widget build(BuildContext c) => ListView(
    padding: EdgeInsets.fromLTRB(12, pageTopPadding(c, 12), 12, 12),
    children: [
      Card(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              FilledButton.tonalIcon(
                onPressed: busy ? null : regionTest,
                icon: const Icon(Icons.location_searching),
                label: Text(c.l10n.chinaRouteTest),
              ),
              if (region != null) ...[
                const SizedBox(height: 12),
                Text(
                  '${region!['country']} / ${region!['province']} / ${region!['city'] ?? ''}',
                ),
                Text(
                  'IP ${region!['address']} · ${region!['isp']} · ${region!['asn']}',
                ),
                Text(
                  '${c.l10n.health} ${region!['health_score']}/100 · Risk ${region!['risk_score']} · Proxy ${region!['proxy']}',
                ),
                Text(
                  '${region!['provider'] ?? ''} / ${region!['organization'] ?? ''}',
                ),
              ],
              if (regionError != null) ...[
                const SizedBox(height: 12),
                AppErrorPanel(
                  title: c.l10n.chinaRouteErrorTitle,
                  message:
                      '${c.l10n.chinaRouteErrorMessage}\n${apiErrorMessage(c, regionError!)}',
                  onRetry: regionTest,
                  icon: Icons.location_off_rounded,
                ),
              ],
            ],
          ),
        ),
      ),
      _liveGauge(c),
    ],
  );

  Widget _liveGauge(BuildContext c) {
    final p = progress ?? const <String, dynamic>{};
    final current = (p['current_mbps'] as num?)?.toDouble() ?? 0;
    final stage = p['stage'] ?? 'ready';
    final running = p['running'] == true;
    final selected = _selectedServerLabel();
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          children: [
            Row(
              children: [
                Expanded(
                  child: _metric(
                    'PING',
                    '${((p['ping_ms'] as num?) ?? 0).toStringAsFixed(1)} ms',
                    Icons.network_ping,
                  ),
                ),
                Expanded(
                  child: _metric(
                    'DOWNLOAD',
                    '${((p['download_mbps'] as num?) ?? 0).toStringAsFixed(1)} Mbps',
                    Icons.download,
                  ),
                ),
                Expanded(
                  child: _metric(
                    'UPLOAD',
                    '${((p['upload_mbps'] as num?) ?? 0).toStringAsFixed(1)} Mbps',
                    Icons.upload,
                  ),
                ),
              ],
            ),
            if ((p['sessions'] ?? 1) > 1) ...[
              const SizedBox(height: 10),
              Chip(
                avatar: const Icon(Icons.call_split_rounded, size: 18),
                label: Text(
                  (p['sessions'] ?? 1) == 3
                      ? c.l10n.tripleMode
                      : c.l10n.dualMode,
                ),
              ),
            ],
            const SizedBox(height: 16),
            LayoutBuilder(
              builder: (context, constraints) {
                final gaugeSize = math.min(290.0, constraints.maxWidth);
                return GestureDetector(
                  onTap: busy
                      ? null
                      : () {
                          if (ooklaID == null && cnID == null) {
                            chooseSpeedServer();
                          } else {
                            run(cnID != null ? 'cn' : 'ookla');
                          }
                        },
                  child: SizedBox(
                    width: gaugeSize,
                    height: gaugeSize,
                    child: Stack(
                      alignment: Alignment.center,
                      children: [
                        CustomPaint(
                          size: Size.square(gaugeSize),
                          painter: SpeedGaugePainter(
                            value: current,
                            active: running,
                            color: Theme.of(c).colorScheme.primary,
                            track: Theme.of(
                              c,
                            ).colorScheme.surfaceContainerHighest,
                            textColor: Theme.of(c).colorScheme.onSurface,
                          ),
                        ),
                        Transform.translate(
                          offset: Offset(0, running ? gaugeSize * .22 : 0),
                          child: Column(
                            mainAxisSize: MainAxisSize.min,
                            children: running
                                ? [
                                    Text(
                                      stage.toString().toUpperCase(),
                                      style: TextStyle(
                                        color: Theme.of(c).colorScheme.primary,
                                        fontWeight: FontWeight.w700,
                                        letterSpacing: 1.2,
                                      ),
                                    ),
                                    Text(
                                      current.toStringAsFixed(2),
                                      style: Theme.of(c).textTheme.displaySmall
                                          ?.copyWith(
                                            fontWeight: FontWeight.w300,
                                          ),
                                    ),
                                    Text(
                                      'Mbps',
                                      style: TextStyle(
                                        color: Theme.of(
                                          c,
                                        ).colorScheme.onSurfaceVariant,
                                      ),
                                    ),
                                  ]
                                : [
                                    Text(
                                      'GO',
                                      style: Theme.of(c).textTheme.displayLarge
                                          ?.copyWith(
                                            fontWeight: FontWeight.w300,
                                          ),
                                    ),
                                    const SizedBox(height: 6),
                                    Text(
                                      ooklaID == null && cnID == null
                                          ? c.l10n.selectServer
                                          : c.l10n.tapToTest,
                                    ),
                                  ],
                          ),
                        ),
                      ],
                    ),
                  ),
                );
              },
            ),
            const SizedBox(height: 6),
            ListTile(
              contentPadding: EdgeInsets.zero,
              leading: CircleAvatar(
                backgroundColor: Theme.of(c).colorScheme.primaryContainer,
                child: const Icon(Icons.dns_rounded),
              ),
              title: Text(selected),
              subtitle: Text(
                cnID != null
                    ? 'speedtest.cn'
                    : (ooklaID != null ? 'Ookla' : c.l10n.notSelected),
              ),
              trailing: TextButton(
                onPressed: busy ? null : chooseSpeedServer,
                child: Text(c.l10n.changeServer),
              ),
              onTap: busy ? null : chooseSpeedServer,
            ),
          ],
        ),
      ),
    );
  }

  String _selectedServerLabel() {
    dynamic selected;
    if (cnID != null) {
      for (final x in cn) {
        if ('${x['id']}' == cnID) selected = x;
      }
    } else if (ooklaID != null) {
      for (final x in ookla) {
        if ('${x['id']}' == ooklaID) selected = x;
      }
    }
    if (selected == null) return context.l10n.selectServer;
    return '${selected['province'] ?? selected['name'] ?? ''} ${selected['city'] ?? ''} · ${selected['operator'] ?? selected['sponsor'] ?? ''}';
  }

  Future<void> chooseSpeedServer() async {
    final loading = _loadSpeedServerPicker();
    final search = TextEditingController();
    widget.onOverlayVisibilityChanged(true);
    await WidgetsBinding.instance.endOfFrame;
    if (!mounted) {
      widget.onOverlayVisibilityChanged(false);
      search.dispose();
      return;
    }
    try {
      await showModalBottomSheet<void>(
        context: context,
        isScrollControlled: true,
        useSafeArea: true,
        showDragHandle: true,
        builder: (sheetContext) => FractionallySizedBox(
          heightFactor: .94,
          child: StatefulBuilder(
            builder: (context, modalSetState) => FutureBuilder<void>(
              future: loading,
              builder: (context, snapshot) => DefaultTabController(
                length: 2,
                child: Column(
                  children: [
                    const TabBar(
                      tabs: [
                        Tab(text: 'speedtest.cn'),
                        Tab(text: 'Ookla'),
                      ],
                    ),
                    Padding(
                      padding: const EdgeInsets.all(16),
                      child: TextField(
                        controller: search,
                        onChanged: (_) => modalSetState(() {}),
                        decoration: InputDecoration(
                          hintText: context.l10n.searchTestNode,
                          prefixIcon: const Icon(Icons.search_rounded),
                        ),
                      ),
                    ),
                    if (snapshot.connectionState != ConnectionState.done)
                      const LinearProgressIndicator(),
                    Expanded(
                      child: TabBarView(
                        children: [
                          cnError == null
                              ? _speedServerList(context, cn, 'cn', search.text)
                              : _speedServerError(
                                  context,
                                  title: context.l10n.speedtestCnErrorTitle,
                                  message:
                                      '${context.l10n.speedtestCnErrorMessage}\n${apiErrorMessage(context, cnError!)}',
                                  type: 'cn',
                                ),
                          ooklaError == null
                              ? _speedServerList(
                                  context,
                                  ookla,
                                  'ookla',
                                  search.text,
                                )
                              : _speedServerError(
                                  context,
                                  title: context.l10n.ooklaErrorTitle,
                                  message:
                                      '${context.l10n.ooklaErrorMessage}\n${apiErrorMessage(context, ooklaError!)}',
                                  type: 'ookla',
                                ),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ),
        ),
      );
    } finally {
      widget.onOverlayVisibilityChanged(false);
      search.dispose();
    }
  }

  Future<void> _loadSpeedServerPicker() async {
    Map<String, dynamic>? newRegion = region;
    if (newRegion == null) {
      try {
        newRegion = await widget.api.get('regiontest', timeout: 45);
      } catch (error) {
        regionError = error;
      }
    }
    Map<String, dynamic>? cnResult, ooklaResult;
    Object? nextCnError, nextOoklaError;
    await Future.wait([
      widget.api
          .get('speedtestcn/servers', timeout: 60)
          .then((value) => cnResult = value)
          .catchError((Object error) {
            nextCnError = error;
            return <String, dynamic>{};
          }),
      widget.api
          .get('speedtest/servers', timeout: 60)
          .then((value) => ooklaResult = value)
          .catchError((Object error) {
            nextOoklaError = error;
            return <String, dynamic>{};
          }),
    ]);
    if (!mounted) return;
    setState(() {
      region = newRegion;
      cnError = nextCnError;
      ooklaError = nextOoklaError;
      if (cnResult != null) cn = cnResult!['servers'] ?? [];
      if (ooklaResult != null) ookla = ooklaResult!['servers'] ?? [];
    });
  }

  Widget _speedServerError(
    BuildContext context, {
    required String title,
    required String message,
    required String type,
  }) => ListView(
    padding: const EdgeInsets.all(18),
    children: [
      AppErrorPanel(
        title: title,
        message: message,
        onRetry: () {
          Navigator.pop(context);
          Future<void>.delayed(Duration.zero, chooseSpeedServer);
        },
        icon: Icons.cloud_off_rounded,
      ),
    ],
  );

  Widget _speedServerList(
    BuildContext c,
    List<dynamic> source,
    String type,
    String query,
  ) {
    final q = query.trim().toLowerCase();
    final rows = source
        .where((x) => q.isEmpty || x.values.join(' ').toLowerCase().contains(q))
        .toList();
    return ListView.separated(
      padding: const EdgeInsets.fromLTRB(18, 0, 18, 28),
      itemCount: rows.length,
      separatorBuilder: (_, _) => const Divider(height: 1),
      itemBuilder: (context, i) {
        final x = rows[i];
        final id = '${x['id']}';
        final selected = type == 'cn' ? cnID == id : ooklaID == id;
        final title =
            '${x['province'] ?? x['name'] ?? '-'}${x['city'] == null || x['city'] == '' ? '' : ' ${x['city']}'}';
        final sponsor = '${x['operator'] ?? x['sponsor'] ?? '-'}';
        final ping = (x['ping_ms'] as num?)?.toDouble();
        final distance = _distanceFor(x);
        return ListTile(
          contentPadding: const EdgeInsets.symmetric(vertical: 8),
          leading: CircleAvatar(
            backgroundColor: Theme.of(c).colorScheme.primaryContainer,
            child: const Icon(Icons.public_rounded),
          ),
          title: Text(
            title,
            style: const TextStyle(fontWeight: FontWeight.w700),
          ),
          subtitle: Text(sponsor),
          trailing: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            crossAxisAlignment: CrossAxisAlignment.end,
            children: [
              Text(distance == null ? '— km' : '${distance.round()} km'),
              Text(
                ping == null ? '— ms' : '${ping.round()} ms',
                style: TextStyle(
                  color: ping != null && ping < 100
                      ? Theme.of(c).colorScheme.primary
                      : Theme.of(c).colorScheme.onSurfaceVariant,
                ),
              ),
            ],
          ),
          selected: selected,
          onTap: () {
            setState(() {
              if (type == 'cn') {
                cnID = id;
                ooklaID = null;
              } else {
                ooklaID = id;
                cnID = null;
              }
            });
            Navigator.pop(context);
          },
        );
      },
    );
  }

  double? _distanceFor(dynamic server) {
    double? number(dynamic value) =>
        value is num ? value.toDouble() : double.tryParse('${value ?? ''}');
    final a = number(region?['latitude']);
    final b = number(region?['longitude']);
    var c = number(server['latitude'] ?? server['lat']);
    var d = number(server['longitude'] ?? server['lon']);
    if (c == null || d == null) {
      final coordinates = <String, (double, double)>{
        '石家庄': (38.04, 114.51),
        '西安': (34.34, 108.94),
        '西宁': (36.62, 101.78),
        '哈密': (42.82, 93.52),
        '阿勒泰': (47.85, 88.13),
        '和田': (37.11, 79.92),
        '克州': (39.71, 76.17),
        '武汉': (30.59, 114.30),
        '沈阳': (41.81, 123.43),
        '绥化': (46.65, 126.97),
      }['${server['city'] ?? ''}'];
      c = coordinates?.$1;
      d = coordinates?.$2;
    }
    if (a == null || b == null || c == null || d == null) return null;
    double rad(double v) => v * math.pi / 180;
    final h =
        math.pow(math.sin(rad(c - a) / 2), 2) +
        math.cos(rad(a)) *
            math.cos(rad(c)) *
            math.pow(math.sin(rad(d - b) / 2), 2);
    return 6371 * 2 * math.asin(math.sqrt(h));
  }
}

class SpeedGaugePainter extends CustomPainter {
  const SpeedGaugePainter({
    required this.value,
    required this.active,
    required this.color,
    required this.track,
    required this.textColor,
  });
  final double value;
  final bool active;
  final Color color, track, textColor;

  @override
  void paint(Canvas canvas, Size size) {
    final center = size.center(Offset.zero);
    final radius = size.width * .39;
    const start = math.pi * .75;
    const sweep = math.pi * 1.5;
    final rect = Rect.fromCircle(center: center, radius: radius);
    final trackPaint = Paint()
      ..style = PaintingStyle.stroke
      ..strokeCap = StrokeCap.butt
      ..strokeWidth = 26
      ..color = track;
    canvas.drawArc(rect, start, sweep, false, trackPaint);
    final normalized = (math.log(math.max(1, value + 1)) / math.log(1001))
        .clamp(0.0, 1.0);
    final activeSweep = active ? sweep * normalized : sweep;
    final activePaint = Paint()
      ..style = PaintingStyle.stroke
      ..strokeCap = StrokeCap.butt
      ..strokeWidth = 26
      ..shader = SweepGradient(
        startAngle: start,
        endAngle: start + sweep,
        colors: [const Color(0xff00b8ff), const Color(0xff5ff5cc), color],
        transform: const GradientRotation(start),
      ).createShader(rect);
    canvas.drawArc(rect, start, activeSweep, false, activePaint);

    if (active) {
      final angle = start + sweep * normalized;
      final direction = Offset(math.cos(angle), math.sin(angle));
      final perpendicular = Offset(-direction.dy, direction.dx);
      final tip = center + direction * (radius - 56);
      final tail = center - direction * 13;
      final needlePath = Path()
        ..moveTo((tail + perpendicular * 10).dx, (tail + perpendicular * 10).dy)
        ..lineTo(tip.dx, tip.dy)
        ..lineTo((tail - perpendicular * 10).dx, (tail - perpendicular * 10).dy)
        ..close();
      final needleBounds = Rect.fromPoints(tail, tip).inflate(12);
      canvas.drawShadow(needlePath, const Color(0xaa000000), 8, false);
      canvas.drawPath(
        needlePath,
        Paint()
          ..shader = LinearGradient(
            begin: Alignment.bottomCenter,
            end: Alignment.topCenter,
            colors: [
              textColor.withValues(alpha: .16),
              textColor.withValues(alpha: .92),
            ],
          ).createShader(needleBounds),
      );
    }

    const labels = ['0', '5', '10', '50', '100', '250', '500', '1000'];
    for (var i = 0; i < labels.length; i++) {
      final angle = start + sweep * i / (labels.length - 1);
      final point =
          center + Offset(math.cos(angle), math.sin(angle)) * (radius - 34);
      final painter = TextPainter(
        text: TextSpan(
          text: labels[i],
          style: TextStyle(
            color: textColor.withValues(alpha: .72),
            fontSize: 12,
            fontWeight: FontWeight.w700,
          ),
        ),
        textDirection: TextDirection.ltr,
      )..layout();
      painter.paint(
        canvas,
        point - Offset(painter.width / 2, painter.height / 2),
      );
    }
  }

  @override
  bool shouldRepaint(covariant SpeedGaugePainter old) =>
      old.value != value ||
      old.active != active ||
      old.color != color ||
      old.track != track;
}

class SettingsPage extends StatefulWidget {
  const SettingsPage({super.key, required this.api});
  final OpenSocksApi api;
  @override
  State<SettingsPage> createState() => _SettingsPageState();
}

class _SettingsPageState extends State<SettingsPage> {
  Map<String, dynamic>? s;
  final region = TextEditingController(),
      exRegions = TextEditingController(),
      inDomains = TextEditingController(),
      exDomains = TextEditingController(),
      inCidrs = TextEditingController(),
      exCidrs = TextEditingController();
  String mode = 'smart';
  bool free = true, auto = true, route = true, busy = false;
  @override
  void initState() {
    super.initState();
    load();
  }

  Future<void> load() async {
    try {
      final x = await widget.api.get('settings');
      if (mounted) {
        setState(() {
          s = x;
          mode = x['mode'] ?? 'smart';
          free = x['freeOnly'] ?? true;
          auto = x['autoConnect'] ?? true;
          route = x['autoRoute'] ?? true;
          region.text = x['region'] ?? '';
          exRegions.text = x['excludeRegions'] ?? '';
          inDomains.text = x['includeDomains'] ?? '';
          exDomains.text = x['excludeDomains'] ?? '';
          inCidrs.text = x['includeCIDRs'] ?? '';
          exCidrs.text = x['excludeCIDRs'] ?? '';
        });
      }
    } catch (e) {
      if (mounted) toast(context, e, true);
    }
  }

  Future<void> save() async {
    setState(() => busy = true);
    try {
      await widget.api.post('settings', {
        'mode': mode,
        'tun': false,
        'free_only': free,
        'auto_connect': auto,
        'auto_route': route,
        'session_count': s?['sessionCount'] ?? 1,
        'region': region.text,
        'exclude_regions': exRegions.text,
        'include_domains': inDomains.text,
        'exclude_domains': exDomains.text,
        'include_cidrs': inCidrs.text,
        'exclude_cidrs': exCidrs.text,
      });
      if (mounted) toast(context, '設定を保存しました');
    } catch (e) {
      if (mounted) toast(context, e, true);
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  @override
  Widget build(BuildContext c) {
    if (s == null) return const Center(child: CircularProgressIndicator());
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        DropdownButtonFormField(
          initialValue: mode,
          decoration: const InputDecoration(
            labelText: 'ルーティングモード',
            border: OutlineInputBorder(),
          ),
          items: const [
            DropdownMenuItem(value: 'smart', child: Text('スマートルーティング')),
            DropdownMenuItem(value: 'global', child: Text('フル中国回線ルーティング')),
          ],
          onChanged: (v) => setState(() => mode = v!),
        ),
        SwitchListTile(
          title: const Text('Freeサーバーのみ'),
          value: free,
          onChanged: (v) => setState(() => free = v),
        ),
        SwitchListTile(
          title: const Text('起動時に自動接続'),
          value: auto,
          onChanged: (v) => setState(() => auto = v),
        ),
        SwitchListTile(
          title: const Text('経路を自動復元'),
          value: route,
          onChanged: (v) => setState(() => route = v),
        ),
        _field(region, '地域フィルター'),
        _field(exRegions, '除外地域'),
        _field(inDomains, '強制対象ドメイン'),
        _field(exDomains, '除外ドメイン'),
        _field(inCidrs, '強制対象CIDR'),
        _field(exCidrs, '除外CIDR'),
        const SizedBox(height: 12),
        FilledButton.icon(
          onPressed: busy ? null : save,
          icon: const Icon(Icons.save),
          label: const Text('保存・適用'),
        ),
      ],
    );
  }

  Widget _field(TextEditingController x, String label) => Padding(
    padding: const EdgeInsets.only(top: 10),
    child: TextField(
      controller: x,
      maxLines: null,
      decoration: InputDecoration(
        labelText: label,
        border: const OutlineInputBorder(),
      ),
    ),
  );
}

class AccountPage extends StatefulWidget {
  const AccountPage({super.key, required this.api});
  final OpenSocksApi api;
  @override
  State<AccountPage> createState() => _AccountPageState();
}

class _AccountPageState extends State<AccountPage> {
  final user = TextEditingController(), pass = TextEditingController();
  bool busy = false;
  Map<String, dynamic>? status;

  @override
  void initState() {
    super.initState();
    load();
  }

  Future<void> load() async {
    try {
      final value = await widget.api.get('status');
      if (mounted) setState(() => status = value);
    } catch (e) {
      if (mounted) toast(context, e, true);
    }
  }

  Future<void> act(String p, [Map<String, dynamic>? b]) async {
    setState(() => busy = true);
    try {
      await widget.api.post(p, b);
      pass.clear();
      await load();
      if (mounted) toast(context, context.l10n.completed);
    } catch (e) {
      if (mounted) toast(context, e, true);
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  @override
  Widget build(BuildContext c) {
    final account = status?['account'] as Map<String, dynamic>?;
    return RefreshIndicator(
      onRefresh: load,
      child: ListView(
        padding: EdgeInsets.fromLTRB(18, pageTopPadding(c, 18), 18, 18),
        children: [
          Card(
            child: Padding(
              padding: const EdgeInsets.all(18),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      CircleAvatar(
                        radius: 27,
                        backgroundColor: Theme.of(
                          c,
                        ).colorScheme.primaryContainer,
                        child: const Icon(Icons.person_rounded, size: 30),
                      ),
                      const SizedBox(width: 14),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              account?['nick'] ?? c.l10n.notSignedIn,
                              style: Theme.of(c).textTheme.titleLarge?.copyWith(
                                fontWeight: FontWeight.w800,
                              ),
                            ),
                            Text(
                              status?['vip'] == true
                                  ? c.l10n.vipAccount
                                  : c.l10n.freeAccount,
                              style: TextStyle(
                                color: Theme.of(c).colorScheme.primary,
                                fontWeight: FontWeight.w700,
                              ),
                            ),
                          ],
                        ),
                      ),
                    ],
                  ),
                  const Divider(height: 28),
                  _accountRow(
                    c,
                    Icons.mail_outline,
                    c.l10n.email,
                    '${account?['email'] ?? '-'}',
                  ),
                  _accountRow(
                    c,
                    Icons.phone_outlined,
                    c.l10n.phone,
                    '${account?['phone'] ?? '-'}',
                  ),
                  _accountRow(
                    c,
                    Icons.event_outlined,
                    c.l10n.expires,
                    '${account?['expire_at'] ?? '-'}',
                  ),
                  _accountRow(
                    c,
                    Icons.timelapse_rounded,
                    c.l10n.remainingDays,
                    c.l10n.daysLeft('${account?['remaining_days'] ?? '-'}'),
                  ),
                  _accountRow(
                    c,
                    Icons.key_rounded,
                    c.l10n.credentials,
                    status?['credentialsSaved'] == true
                        ? c.l10n.saved
                        : c.l10n.notSaved,
                  ),
                ],
              ),
            ),
          ),
          const SizedBox(height: 16),
          Text(
            c.l10n.loginManagement,
            style: Theme.of(
              c,
            ).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w800),
          ),
          const SizedBox(height: 10),
          TextField(
            controller: user,
            decoration: InputDecoration(
              labelText: c.l10n.username,
              border: const OutlineInputBorder(),
            ),
          ),
          const SizedBox(height: 12),
          TextField(
            controller: pass,
            obscureText: true,
            decoration: InputDecoration(
              labelText: c.l10n.password,
              border: const OutlineInputBorder(),
            ),
          ),
          const SizedBox(height: 16),
          FilledButton.icon(
            onPressed: busy
                ? null
                : () => act('login', {
                    'username': user.text,
                    'password': pass.text,
                  }),
            icon: const Icon(Icons.login),
            label: Text(c.l10n.login),
          ),
          OutlinedButton(
            onPressed: busy ? null : () => act('register'),
            child: Text(c.l10n.anonymousRegister),
          ),
          OutlinedButton(
            onPressed: busy ? null : () => act('logout'),
            child: Text(c.l10n.logout),
          ),
        ],
      ),
    );
  }

  Widget _accountRow(
    BuildContext c,
    IconData icon,
    String label,
    String value,
  ) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 5),
    child: Row(
      children: [
        Icon(icon, size: 19, color: Theme.of(c).colorScheme.onSurfaceVariant),
        const SizedBox(width: 10),
        SizedBox(
          width: 74,
          child: Text(
            label,
            style: TextStyle(color: Theme.of(c).colorScheme.onSurfaceVariant),
          ),
        ),
        Expanded(
          child: Text(
            value,
            textAlign: TextAlign.end,
            overflow: TextOverflow.ellipsis,
          ),
        ),
      ],
    ),
  );
}

class LogsPage extends StatefulWidget {
  const LogsPage({super.key, required this.api});
  final OpenSocksApi api;
  @override
  State<LogsPage> createState() => _LogsPageState();
}

class _LogsPageState extends State<LogsPage> {
  String log = '';
  bool busy = false;
  Future<void> load() async {
    setState(() => busy = true);
    try {
      final x = await widget.api.get('logs?lines=300');
      if (mounted) setState(() => log = x['log'] ?? '');
    } catch (e) {
      if (mounted) toast(context, e, true);
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  @override
  void initState() {
    super.initState();
    load();
  }

  @override
  Widget build(BuildContext c) => Column(
    children: [
      Padding(
        padding: const EdgeInsets.all(8),
        child: FilledButton.tonalIcon(
          onPressed: busy ? null : load,
          icon: const Icon(Icons.refresh),
          label: const Text('更新'),
        ),
      ),
      Expanded(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(12),
          child: SelectableText(
            log,
            style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
          ),
        ),
      ),
    ],
  );
}
