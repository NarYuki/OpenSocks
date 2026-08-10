# OpenSocks opkg signing key

`d24a5e234001294c` is the OpenSocks repository public usign key. Install it as:

```sh
wget -O /etc/opkg/keys/d24a5e234001294c https://rel.n4t.su/opkg/d24a5e234001294c
```

The matching secret key is never stored in this repository. GitHub Actions
loads it only from the `OPKG_SIGNING_KEY` repository secret and refuses to
publish unsigned artifacts.
