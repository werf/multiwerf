Changelog
---------

$CHANGELOG_TEXT

Installation and usage
----------------------

Download from Github assets or from Bintray mirror or curl-bash-it:

```
curl -L https://raw.githubusercontent.com/werf/multiwerf/master/get.sh | bash
```

Download latest `werf` and create a shell alias:

```
$ source $(multiwerf use --as-file 1.2 ea)
$ werf version
v1.0.0-alpha.17
```

Go to `werf` [documentation](https://werf.io).

Download from Bintray mirror
----------------------------

[Linux amd64](https://storage.yandexcloud.net/multiwerf/targets/releases/$VERSION/multiwerf-linux-amd64-$VERSION)

[Linux arm64](https://storage.yandexcloud.net/multiwerf/targets/releases/$VERSION/multiwerf-linux-arm64-$VERSION)

[Darwin amd64](https://storage.yandexcloud.net/multiwerf/targets/releases/$VERSION/multiwerf-darwin-amd64-$VERSION)

[Darwin arm64](https://storage.yandexcloud.net/multiwerf/targets/releases/$VERSION/multiwerf-darwin-arm64-$VERSION)

[Windows amd64](https://storage.yandexcloud.net/multiwerf/targets/releases/$VERSION/multiwerf-windows-amd64-$VERSION.exe)

<!-- repo: $BINTRAY_REPO -->
