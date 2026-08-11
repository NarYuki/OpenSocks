// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Chinese (`zh`).
class AppLocalizationsZh extends AppLocalizations {
  AppLocalizationsZh([String locale = 'zh']) : super(locale);

  @override
  String get pairHelp => '请输入 LuCI“手机配对”中显示的 URL 和令牌。';

  @override
  String get routerUrl => '路由器 URL';

  @override
  String get pairToken => '配对令牌';

  @override
  String get connect => '连接';

  @override
  String get traffic => '流量';

  @override
  String get test => '测试';

  @override
  String get account => '账户';

  @override
  String get authenticating => '正在认证';

  @override
  String get switching => '正在切换线路';

  @override
  String get connecting => '正在连接';

  @override
  String get networkConfig => '正在配置网络';

  @override
  String get interfaceProcessing => '正在配置接口';

  @override
  String get disconnecting => '正在断开';

  @override
  String get connected => '连接完成';

  @override
  String get disconnected => '已断开';

  @override
  String get routingMode => '路由模式';

  @override
  String get smartRouting => '智能路由';

  @override
  String get smartDescription => '仅将中国方向流量发送到 OpenSocks';

  @override
  String get globalRouting => '全中国线路路由';

  @override
  String get globalDescription => '将 LAN 的全部网页流量发送到 OpenSocks';

  @override
  String get changingMode => '正在更改路由模式';

  @override
  String get modeChanged => '路由模式已更改';

  @override
  String get protected => '回国ing';

  @override
  String get notConnected => '未连接';

  @override
  String get tapToConnect => '点击连接';

  @override
  String get selectServer => '选择服务器';

  @override
  String get routeStopped => '网络路由已停止';

  @override
  String get routeHealthy => '网络路由正常';

  @override
  String get routeRestoring => '正在恢复网络路由';

  @override
  String get notSignedIn => '未登录';

  @override
  String daysLeft(Object days) {
    return '剩余 $days 天';
  }

  @override
  String serversCount(int count) {
    return '服务器 ($count)';
  }

  @override
  String historyCount(int count) {
    return '历史 ($count)';
  }

  @override
  String get serverList => '服务器列表';

  @override
  String get measuringAllPing => '正在测量所有服务器延迟…';

  @override
  String get serversUnavailable => '无法获取服务器';

  @override
  String get reconnected => '已重新连接';

  @override
  String get chinaServiceTraffic => '中国服务流量';

  @override
  String cumulativeTraffic(Object up, Object down, Object total) {
    return '累计 ↑ $up / ↓ $down / Σ $total';
  }

  @override
  String get chinaRouteTest => '中国线路与 IP 健康测试';

  @override
  String get health => '健康度';

  @override
  String get tapToTest => '点击开始测试';

  @override
  String get notSelected => '未选择';

  @override
  String get changeServer => '更换服务器';

  @override
  String get searchTestNode => '搜索节点、地区或运营商';

  @override
  String serverFetchError(Object error) {
    return '服务器获取错误：$error';
  }

  @override
  String get retry => '重试';

  @override
  String get trafficErrorTitle => '无法获取流量数据';

  @override
  String get trafficErrorMessage => '请检查与路由器的连接后重新加载。';

  @override
  String get chinaRouteErrorTitle => '中国线路测试失败';

  @override
  String get chinaRouteErrorMessage => '请检查中国线路连接状态后重新测试。';

  @override
  String get speedtestCnErrorTitle => '无法获取SpeedTest.cn服务器';

  @override
  String get speedtestCnErrorMessage => '无法连接SpeedTest.cn中国节点API，请稍后重试。';

  @override
  String get ooklaErrorTitle => '无法获取Ookla服务器';

  @override
  String get ooklaErrorMessage => '无法连接Ookla中国服务器列表，请稍后重试。';

  @override
  String get routerConnectionError => '无法连接路由器，请检查Wi-Fi或Tailscale连接。';

  @override
  String get routerTimeoutError => '路由器未在规定时间内响应。';

  @override
  String get invalidResponseError => '路由器返回了无效响应。';

  @override
  String get serverResponseError => '路由器端处理失败。';

  @override
  String get completed => '操作完成';

  @override
  String get email => '邮箱';

  @override
  String get phone => '电话号码';

  @override
  String get expires => '有效期';

  @override
  String get remainingDays => '剩余天数';

  @override
  String get credentials => '认证信息';

  @override
  String get saved => '已保存';

  @override
  String get notSaved => '未保存';

  @override
  String get loginManagement => '登录管理';

  @override
  String get username => '邮箱 / 用户名';

  @override
  String get password => '密码';

  @override
  String get login => '登录';

  @override
  String get anonymousRegister => '匿名免费注册';

  @override
  String get logout => '退出登录';

  @override
  String get freeAccount => '免费账户';

  @override
  String get vipAccount => 'VIP 账户';

  @override
  String get singleMode => '单会话模式';

  @override
  String get dualMode => '双会话模式';

  @override
  String get dualModeDescription => '将两个相互隔离的会话连接到同一中国服务器，并在两条线路之间分配并发TCP流量。';

  @override
  String get serviceHonorOfKings => '王者荣耀';

  @override
  String get serviceOtherChina => '其他中国流量';

  @override
  String get sessionMode => '会话模式';

  @override
  String get tripleMode => '三会话模式';
}
