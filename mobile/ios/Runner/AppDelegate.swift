import Flutter
import Network
import UIKit

@main
@objc class AppDelegate: FlutterAppDelegate, FlutterImplicitEngineDelegate {
  private var localNetworkBrowser: NWBrowser?

  override func application(
    _ application: UIApplication,
    didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?
  ) -> Bool {
    return super.application(application, didFinishLaunchingWithOptions: launchOptions)
  }

  func didInitializeImplicitFlutterEngine(_ engineBridge: FlutterImplicitEngineBridge) {
    GeneratedPluginRegistrant.register(with: engineBridge.pluginRegistry)
    let channel = FlutterMethodChannel(
      name: "moe.n4tsu.opensocks/native",
      binaryMessenger: engineBridge.applicationRegistrar.messenger()
    )
    channel.setMethodCallHandler { [weak self] call, result in
      guard call.method == "requestLocalNetworkAccess" else {
        result(FlutterMethodNotImplemented)
        return
      }
      self?.requestLocalNetworkAccess()
      result(nil)
    }
  }

  private func requestLocalNetworkAccess() {
    localNetworkBrowser?.cancel()
    let parameters = NWParameters.tcp
    parameters.includePeerToPeer = true
    let browser = NWBrowser(
      for: .bonjour(type: "_opensocks._tcp", domain: nil),
      using: parameters
    )
    localNetworkBrowser = browser
    browser.stateUpdateHandler = { [weak self, weak browser] state in
      switch state {
      case .ready, .failed, .cancelled:
        if case .ready = state { return }
        browser?.cancel()
        self?.localNetworkBrowser = nil
      default:
        break
      }
    }
    browser.start(queue: .main)
  }
}
