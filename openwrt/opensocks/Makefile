include $(TOPDIR)/rules.mk

PKG_NAME:=opensocks
PKG_VERSION:=0.1.0
PKG_RELEASE:=1

PKG_MAINTAINER:=OpenSocks Developers
PKG_LICENSE:=GPL-3.0-or-later

PKG_BUILD_DEPENDS:=golang/host
PKG_BUILD_PARALLEL:=1
PKG_USE_MIPS16:=0

GO_PKG:=dev.opensocks.daemon
GO_PKG_BUILD_PKG:=dev.opensocks.daemon
GO_PKG_LDFLAGS_X:=main.version=$(PKG_VERSION)

include $(INCLUDE_DIR)/package.mk
include ../../feeds/packages/lang/golang/golang-package.mk

define Package/opensocks
  SECTION:=net
  CATEGORY:=Network
  SUBMENU:=Proxy
  TITLE:=Low-memory Transocks proxy daemon for OpenWrt
  DEPENDS:=+shadowsocks-libev-ss-redir +shadowsocks-libev-ss-local +nftables-json +ca-bundle +uci
endef

define Package/opensocks/description
  Clean-room daemon for the Transocks VPN service on OpenWrt.
  Fetches lines, resolves Shadowsocks server configs and supervises a tiny
  ss-redir process. Smart routing uses an nftables China CIDR set and is
  designed for low-memory routers.
endef

define Package/opensocks/install
	$(call GoPackage/Pack/Default,$(1))
	$(INSTALL_DIR) $(1)/etc/init.d
	$(INSTALL_BIN) ./files/etc/init.d/opensocks $(1)/etc/init.d/opensocks
	$(INSTALL_DIR) $(1)/etc/config
	$(INSTALL_DATA) ./files/etc/config/opensocks $(1)/etc/config/opensocks
	$(INSTALL_DIR) $(1)/etc/opensocks
endef

$(eval $(call GoPackage/Build))
$(eval $(call BuildPackage,opensocks))
