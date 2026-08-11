include $(TOPDIR)/rules.mk

PKG_NAME:=opensocks-minimal
PKG_VERSION:=0.2.10
PKG_RELEASE:=1
PKG_LICENSE:=GPL-3.0-or-later

include $(INCLUDE_DIR)/package.mk

define Package/opensocks-minimal
  SECTION:=net
  CATEGORY:=Network
  SUBMENU:=Proxy
  TITLE:=Download-on-boot minimal OpenSocks launcher
  DEPENDS:=+shadowsocks-libev-ss-redir +shadowsocks-libev-ss-local +nftables-json +kmod-nft-tproxy +ca-bundle +uci +wget-ssl +usign
endef

define Package/opensocks-minimal/install
	$(INSTALL_DIR) $(1)/etc/init.d $(1)/etc/config $(1)/etc/opensocks $(1)/usr/lib/opensocks
	$(INSTALL_BIN) ./files/etc/init.d/opensocks $(1)/etc/init.d/opensocks
	$(INSTALL_DATA) ./files/etc/config/opensocks $(1)/etc/config/opensocks
	$(INSTALL_BIN) ./files/usr/lib/opensocks/launcher.sh $(1)/usr/lib/opensocks/launcher.sh
endef

$(eval $(call BuildPackage,opensocks-minimal))
