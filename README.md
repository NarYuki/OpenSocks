# OpenSocks for OpenWrt

[中文](README.md) | [English](README.en.md) | [日本語](README.ja.md)

OpenSocks 是面向中国线路的低内存 OpenWrt 客户端、LuCI 管理界面和手机控制端。数据平面使用 `shadowsocks-libev` 与 nftables，不需要大型一体化代理核心，适合存储和内存有限的路由器。

## 主要功能

- 智能中国路由：中国服务经 Transocks 中国线路，X、YouTube、Google 等经原有 WAN
- 全中国线路路由：LAN/Wi-Fi 的公网 TCP/UDP 全部经 Transocks 中国线路
- TCP REDIRECT 与 UDP TPROXY（包括《王者荣耀》PVP UDP 通信）
- 自动恢复路由、登录会话和固定选择的服务器
- AES-256-GCM 加密保存登录信息
- 服务器延迟排序、历史记录和一键重连
- 中国服务分类流量统计、实时速率和累计流量
- Ookla 中国节点与 SpeedTest.cn 测速
- LuCI、Android 和 iOS 支持中文、英文、日文
- 约 1.8KB 的 minimal IPK；程序下载到 `/tmp`，设置保留在 `/etc`

## 安装

推荐使用软件源：

```sh
echo 'src/gz opensocks https://rel.n4t.su/opkg' >> /etc/opkg/customfeeds.conf
opkg update
opkg install opensocks-minimal luci-app-opensocks
```

也可以从 [GitHub Releases](https://github.com/NarYuki/OpenSocks/releases/latest) 下载 IPK 后直接安装。完整的安装、登录、功能和手机配对说明请参阅：

- [中文 Wiki](https://github.com/NarYuki/OpenSocks/wiki)
- [English Wiki](https://github.com/NarYuki/OpenSocks/wiki/Home-en)
- [日本語 Wiki](https://github.com/NarYuki/OpenSocks/wiki/Home-ja)

## 项目结构

```text
openwrt/opensocks/          Go 服务和完整软件包
openwrt/opensocks-minimal/  下载到 tmpfs 的最小启动器
openwrt/luci-app-opensocks/ LuCI 界面和本地化
mobile/                     Flutter Android/iOS 客户端
tools/build-release.sh      IPK、opkg 索引和移动端发布构建
```

## 开发测试

```sh
cd openwrt/opensocks/src
go test ./...
go vet ./...

cd ../../../mobile
flutter analyze
flutter test
```

## 许可证

GPL-3.0-or-later，详见 [LICENSE](LICENSE)。
