include $(TOPDIR)/rules.mk

PKG_NAME:=luci-app-opensocks
PKG_VERSION:=0.2.0
PKG_RELEASE:=4
PKG_LICENSE:=GPL-3.0-or-later

LUCI_DEPENDS:=+(opensocks||opensocks-minimal) +luci-base +luci-compat +curl

include $(TOPDIR)/feeds/luci/luci.mk

# call BuildPackage - OpenWrt buildroot signature
