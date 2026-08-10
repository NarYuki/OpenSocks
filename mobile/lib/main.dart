import 'dart:async';
import 'package:flutter/material.dart';
import 'api.dart';
import 'connection_store.dart';

void main() => runApp(const OpenSocksApp());

class OpenSocksApp extends StatelessWidget {
  const OpenSocksApp({super.key});
  @override
  Widget build(BuildContext context) => MaterialApp(
    debugShowCheckedModeBanner: false,
    title: 'OpenSocks',
    theme: ThemeData(
      colorScheme: ColorScheme.fromSeed(
        seedColor: const Color(0xff00a4d6),
        brightness: Brightness.dark,
      ),
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
  bool loading = true;
  String? url, token;
  @override
  void initState() {
    super.initState();
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
                const Text('LuCIの「スマホ連携」に表示されたURLとトークンを入力してください。'),
                const SizedBox(height: 28),
                TextField(
                  controller: url,
                  keyboardType: TextInputType.url,
                  decoration: const InputDecoration(
                    labelText: 'ルーターURL',
                    prefixIcon: Icon(Icons.link),
                    border: OutlineInputBorder(),
                  ),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: token,
                  obscureText: true,
                  decoration: const InputDecoration(
                    labelText: '連携トークン',
                    prefixIcon: Icon(Icons.key),
                    border: OutlineInputBorder(),
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
                  label: const Text('接続'),
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
  late final pages = <Widget>[
    OverviewPage(api: widget.api),
    LinesPage(api: widget.api),
    TrafficPage(api: widget.api),
    TestsPage(api: widget.api),
    SettingsPage(api: widget.api),
    AccountPage(api: widget.api),
    LogsPage(api: widget.api),
  ];
  @override
  Widget build(BuildContext c) => Scaffold(
    appBar: AppBar(
      title: const Text('OpenSocks'),
      actions: [
        IconButton(
          onPressed: widget.onForget,
          tooltip: '連携解除',
          icon: const Icon(Icons.phonelink_erase),
        ),
      ],
    ),
    body: IndexedStack(index: index, children: pages),
    bottomNavigationBar: NavigationBar(
      selectedIndex: index,
      onDestinationSelected: (v) => setState(() => index = v),
      destinations: const [
        NavigationDestination(icon: Icon(Icons.dashboard), label: '概要'),
        NavigationDestination(icon: Icon(Icons.dns), label: '回線'),
        NavigationDestination(icon: Icon(Icons.query_stats), label: '通信'),
        NavigationDestination(icon: Icon(Icons.speed), label: 'テスト'),
        NavigationDestination(icon: Icon(Icons.tune), label: '設定'),
        NavigationDestination(icon: Icon(Icons.person), label: 'アカウント'),
        NavigationDestination(icon: Icon(Icons.article), label: 'ログ'),
      ],
    ),
  );
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

class OverviewPage extends StatefulWidget {
  const OverviewPage({super.key, required this.api});
  final OpenSocksApi api;
  @override
  State<OverviewPage> createState() => _OverviewPageState();
}

class _OverviewPageState extends State<OverviewPage> {
  Map<String, dynamic>? s, t;
  Timer? timer;
  bool busy = false;
  @override
  void initState() {
    super.initState();
    refresh();
    timer = Timer.periodic(const Duration(seconds: 1), (_) => traffic());
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
    try {
      final x = await widget.api.get('traffic');
      if (mounted) setState(() => t = x);
    } catch (_) {}
  }

  Future<void> action(String p, [Map<String, dynamic>? b]) async {
    setState(() => busy = true);
    try {
      await widget.api.post(p, b);
      await refresh();
      if (mounted) toast(context, '完了しました');
    } catch (e) {
      if (mounted) toast(context, e, true);
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  @override
  Widget build(BuildContext c) {
    final x = s;
    if (x == null) return const Center(child: CircularProgressIndicator());
    return RefreshIndicator(
      onRefresh: refresh,
      child: ListView(
        padding: const EdgeInsets.all(16),
        children: [
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
                width: 190,
                height: 190,
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  color: (x['running'] == true ? Colors.green : Colors.blueGrey)
                      .withValues(alpha: .16),
                  border: Border.all(
                    color: x['running'] == true
                        ? Colors.greenAccent
                        : Colors.white38,
                    width: 5,
                  ),
                  boxShadow: [
                    BoxShadow(
                      color:
                          (x['running'] == true
                                  ? Colors.greenAccent
                                  : Colors.black)
                              .withValues(alpha: .22),
                      blurRadius: 32,
                      spreadRadius: 4,
                    ),
                  ],
                ),
                child: busy
                    ? const Padding(
                        padding: EdgeInsets.all(68),
                        child: CircularProgressIndicator(strokeWidth: 4),
                      )
                    : Icon(
                        Icons.power_settings_new,
                        size: 82,
                        color: x['running'] == true
                            ? Colors.greenAccent
                            : Colors.white70,
                      ),
              ),
            ),
          ),
          const SizedBox(height: 12),
          Center(
            child: Text(
              x['running'] == true ? 'タップして切断' : 'タップして接続',
              style: Theme.of(c).textTheme.titleMedium,
            ),
          ),
          const SizedBox(height: 18),
          Card(
            child: ListTile(
              leading: Icon(
                x['running'] == true ? Icons.check_circle : Icons.cancel,
                color: x['running'] == true ? Colors.green : Colors.red,
              ),
              title: Text(x['running'] == true ? '接続中' : '切断中'),
              subtitle: Text(
                '${x['lineName'] ?? '-'}\n${x['mode'] == 'global' ? 'フル中国回線ルーティング' : 'スマートルーティング'} · ${x['routingApplied'] == true ? '経路適用済み' : '経路未適用'}',
              ),
              isThreeLine: true,
            ),
          ),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(18),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.spaceAround,
                children: [
                  _metric('上り', rate(t?['up_bps'] ?? 0), Icons.upload),
                  _metric('下り', rate(t?['down_bps'] ?? 0), Icons.download),
                  _metric(
                    '合計',
                    bytes(t?['total_bytes'] ?? 0),
                    Icons.data_usage,
                  ),
                ],
              ),
            ),
          ),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              OutlinedButton.icon(
                onPressed: busy
                    ? null
                    : () => action('setup', {'mode': 'smart'}),
                icon: const Icon(Icons.route),
                label: const Text('スマート構成'),
              ),
              OutlinedButton.icon(
                onPressed: busy
                    ? null
                    : () => action('setup', {'mode': 'global'}),
                icon: const Icon(Icons.public),
                label: const Text('フル中国構成'),
              ),
            ],
          ),
          const SizedBox(height: 12),
          AccountCard(
            account: x['account'] is Map
                ? Map<String, dynamic>.from(x['account'])
                : null,
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
    Text(b, style: const TextStyle(fontWeight: FontWeight.bold)),
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
      title: Text(account?['nick'] ?? '未ログイン'),
      subtitle: Text(
        account == null
            ? '-'
            : '${account!['email'] ?? account!['phone'] ?? ''} · 残り${account!['remaining_days'] ?? '-'}日',
      ),
    ),
  );
}

class LinesPage extends StatefulWidget {
  const LinesPage({super.key, required this.api});
  final OpenSocksApi api;
  @override
  State<LinesPage> createState() => _LinesPageState();
}

class _LinesPageState extends State<LinesPage> {
  List<dynamic> lines = [], history = [];
  bool loading = false, ping = false;
  @override
  void initState() {
    super.initState();
    load();
  }

  Future<void> load([bool p = false]) async {
    setState(() {
      loading = true;
      ping = p;
    });
    try {
      final a = await widget.api.get(
            p ? 'lines?sort=ping' : 'lines',
            timeout: 90,
          ),
          h = await widget.api.get('history');
      if (mounted) {
        setState(() {
          lines = a['lines'] ?? [];
          history = h['history'] ?? [];
        });
      }
    } catch (e) {
      if (mounted) toast(context, e, true);
    } finally {
      if (mounted) setState(() => loading = false);
    }
  }

  Future<void> connect(dynamic id) async {
    toast(context, '接続・切替中…');
    try {
      await widget.api.post('connect', {'line_id': id});
      await load();
      if (mounted) toast(context, '接続しました');
    } catch (e) {
      if (mounted) toast(context, e, true);
    }
  }

  @override
  Widget build(BuildContext c) => DefaultTabController(
    length: 2,
    child: Column(
      children: [
        TabBar(
          tabs: [
            Tab(text: 'サーバー (${lines.length})'),
            Tab(text: '履歴 (${history.length})'),
          ],
        ),
        Expanded(
          child: TabBarView(
            children: [
              RefreshIndicator(
                onRefresh: () => load(),
                child: ListView(
                  padding: const EdgeInsets.all(12),
                  children: [
                    Row(
                      children: [
                        FilledButton.tonalIcon(
                          onPressed: loading ? null : () => load(true),
                          icon: const Icon(Icons.network_ping),
                          label: Text(ping ? 'Ping順' : '全サーバーPing測定'),
                        ),
                        if (loading)
                          const Padding(
                            padding: EdgeInsets.all(12),
                            child: SizedBox.square(
                              dimension: 18,
                              child: CircularProgressIndicator(strokeWidth: 2),
                            ),
                          ),
                      ],
                    ),
                    ...lines.map(
                      (l) => Card(
                        child: ListTile(
                          title: Text('${l['name'] ?? '#${l['id']}'}'),
                          subtitle: Text(
                            '${l['location'] ?? '-'} · ${l['isFree'] == true ? 'Free' : 'VIP'}${l['latency_ms'] != null ? ' · ${l['latency_ms'].toStringAsFixed(1)} ms' : ''}',
                          ),
                          trailing: IconButton(
                            icon: const Icon(Icons.play_arrow),
                            onPressed: () => connect(l['id']),
                          ),
                        ),
                      ),
                    ),
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
                                if (mounted) toast(context, '再接続しました');
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
  const TrafficPage({super.key, required this.api});
  final OpenSocksApi api;
  @override
  State<TrafficPage> createState() => _TrafficPageState();
}

class _TrafficPageState extends State<TrafficPage> {
  Map<String, dynamic>? data;
  Timer? timer;
  @override
  void initState() {
    super.initState();
    load();
    timer = Timer.periodic(const Duration(seconds: 1), (_) => load());
  }

  @override
  void dispose() {
    timer?.cancel();
    super.dispose();
  }

  Future<void> load() async {
    try {
      final x = await widget.api.get('traffic');
      if (mounted) setState(() => data = x);
    } catch (_) {}
  }

  @override
  Widget build(BuildContext c) {
    final d = data;
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
    return ListView(
      padding: const EdgeInsets.all(12),
      children: [
        Card(
          child: Padding(
            padding: const EdgeInsets.all(18),
            child: Column(
              children: [
                Text(
                  '↑ ${rate(d['up_bps'] ?? 0)}  /  ↓ ${rate(d['down_bps'] ?? 0)}',
                  style: Theme.of(c).textTheme.titleLarge,
                ),
                const SizedBox(height: 8),
                Text(
                  '累積 ↑ ${bytes(d['up_bytes'] ?? 0)} / ↓ ${bytes(d['down_bytes'] ?? 0)} / Σ ${bytes(d['total_bytes'] ?? 0)}',
                ),
              ],
            ),
          ),
        ),
        const ListTile(title: Text('中国サービス別トラフィック')),
        for (final e in services)
          Card(
            child: ListTile(
              title: Text(e.key.replaceAll('_', ' / ')),
              subtitle: Text(
                '↑ ${bytes(e.value['up_bytes'] ?? 0)}  ↓ ${bytes(e.value['down_bytes'] ?? 0)}',
              ),
              trailing: Text(bytes(e.value['total_bytes'] ?? 0)),
            ),
          ),
      ],
    );
  }
}

class TestsPage extends StatefulWidget {
  const TestsPage({super.key, required this.api});
  final OpenSocksApi api;
  @override
  State<TestsPage> createState() => _TestsPageState();
}

class _TestsPageState extends State<TestsPage> {
  Map<String, dynamic>? region, progress;
  List<dynamic> ookla = [], cn = [];
  String? ooklaID, cnID;
  bool busy = false;
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
      if (mounted) setState(() => region = x);
    } catch (e) {
      if (mounted) toast(context, e, true);
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
            ooklaID = ookla.isEmpty ? null : '${ookla.first['id']}';
          } else {
            cn = x['servers'] ?? [];
            cnID = cn.isEmpty ? null : '${cn.first['id']}';
          }
        });
      }
    } catch (e) {
      if (mounted) toast(context, e, true);
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
        const Duration(milliseconds: 100),
        (_) => pollProgress(),
      );
      await pollProgress();
    } catch (e) {
      if (mounted) toast(context, e, true);
    }
  }

  Future<void> pollProgress() async {
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
    } catch (_) {}
  }

  @override
  Widget build(BuildContext c) => ListView(
    padding: const EdgeInsets.all(12),
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
                label: const Text('中国経路・IP健全度テスト'),
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
                  '健全度 ${region!['health_score']}/100 · Risk ${region!['risk_score']} · Proxy ${region!['proxy']}',
                ),
                Text(
                  '${region!['provider'] ?? ''} / ${region!['organization'] ?? ''}',
                ),
              ],
            ],
          ),
        ),
      ),
      _liveGauge(c),
      DefaultTabController(
        length: 2,
        child: SizedBox(
          height: 300,
          child: Column(
            children: [
              const TabBar(
                tabs: [
                  Tab(text: 'Ookla'),
                  Tab(text: 'SpeedTest.cn'),
                ],
              ),
              Expanded(
                child: TabBarView(
                  children: [
                    _speedCard(
                      c,
                      'Ookla 中国サーバー',
                      'ookla',
                      ookla,
                      ooklaID,
                      (v) => setState(() => ooklaID = v),
                    ),
                    _speedCard(
                      c,
                      'SpeedTest.cn',
                      'cn',
                      cn,
                      cnID,
                      (v) => setState(() => cnID = v),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    ],
  );

  Widget _liveGauge(BuildContext c) {
    final p = progress ?? const <String, dynamic>{};
    final current = (p['current_mbps'] as num?)?.toDouble() ?? 0;
    final stage = p['stage'] ?? 'ready';
    final normalized = (current / 200).clamp(0.0, 1.0);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          children: [
            SizedBox(
              width: 210,
              height: 210,
              child: Stack(
                alignment: Alignment.center,
                children: [
                  SizedBox.expand(
                    child: CircularProgressIndicator(
                      value: stage == 'preparing' || stage == 'ping'
                          ? null
                          : normalized,
                      strokeWidth: 15,
                      backgroundColor: Colors.white10,
                    ),
                  ),
                  Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Text(
                        current.toStringAsFixed(1),
                        style: Theme.of(c).textTheme.displayMedium,
                      ),
                      const Text('Mbps'),
                      Text(stage.toString().toUpperCase()),
                    ],
                  ),
                ],
              ),
            ),
            const SizedBox(height: 12),
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceEvenly,
              children: [
                _metric(
                  'PING',
                  '${((p['ping_ms'] as num?) ?? 0).toStringAsFixed(1)} ms',
                  Icons.network_ping,
                ),
                _metric(
                  'DOWNLOAD',
                  '${((p['download_mbps'] as num?) ?? 0).toStringAsFixed(1)} Mbps',
                  Icons.download,
                ),
                _metric(
                  'UPLOAD',
                  '${((p['upload_mbps'] as num?) ?? 0).toStringAsFixed(1)} Mbps',
                  Icons.upload,
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _speedCard(
    BuildContext c,
    String title,
    String type,
    List<dynamic> list,
    String? id,
    ValueChanged<String?> change,
  ) => Card(
    child: Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(title, style: Theme.of(c).textTheme.titleMedium),
          const SizedBox(height: 8),
          if (list.isEmpty)
            OutlinedButton(
              onPressed: busy ? null : () => servers(type),
              child: const Text('サーバー一覧を取得'),
            )
          else
            DropdownButton<String>(
              isExpanded: true,
              value: id,
              items: list
                  .map(
                    (x) => DropdownMenuItem(
                      value: '${x['id']}',
                      child: Text(
                        '${x['name'] ?? x['province'] ?? ''} ${x['city'] ?? ''} ${x['operator'] ?? x['sponsor'] ?? ''}',
                      ),
                    ),
                  )
                  .toList(),
              onChanged: change,
            ),
          FilledButton.icon(
            onPressed: busy || list.isEmpty ? null : () => run(type),
            icon: const Icon(Icons.speed),
            label: const Text('測定開始'),
          ),
        ],
      ),
    ),
  );
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
  Future<void> act(String p, [Map<String, dynamic>? b]) async {
    setState(() => busy = true);
    try {
      await widget.api.post(p, b);
      pass.clear();
      if (mounted) toast(context, '完了しました');
    } catch (e) {
      if (mounted) toast(context, e, true);
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  @override
  Widget build(BuildContext c) => ListView(
    padding: const EdgeInsets.all(18),
    children: [
      TextField(
        controller: user,
        decoration: const InputDecoration(
          labelText: 'メールアドレス / ユーザー名',
          border: OutlineInputBorder(),
        ),
      ),
      const SizedBox(height: 12),
      TextField(
        controller: pass,
        obscureText: true,
        decoration: const InputDecoration(
          labelText: 'パスワード',
          border: OutlineInputBorder(),
        ),
      ),
      const SizedBox(height: 16),
      FilledButton.icon(
        onPressed: busy
            ? null
            : () =>
                  act('login', {'username': user.text, 'password': pass.text}),
        icon: const Icon(Icons.login),
        label: const Text('ログイン'),
      ),
      OutlinedButton(
        onPressed: busy ? null : () => act('register'),
        child: const Text('匿名無料登録'),
      ),
      OutlinedButton(
        onPressed: busy ? null : () => act('logout'),
        child: const Text('ログアウト'),
      ),
    ],
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
