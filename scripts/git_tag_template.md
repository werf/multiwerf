Changelog
---------

$CHANGELOG_TEXT

Installation and usage
----------------------

Download from Github assets or from Bintray mirror or curl-bash-it:

```
curl -L https://raw.githubusercontent.com/flant/multiwerf/master/get.sh | bash
```

Download latest `werf` and create a shell alias:

```
$ source $(multiwerf use --as-file 1.0 alpha)
$ werf version
v1.0.0-alpha.17
```

Go to `werf` [documentation](https://flant.github.io/werf/).

Download from Bintray mirror
----------------------------

- [Linux amd64](https://dl.bintray.com/flant/$BINTRAY_REPO/$VERSION/multiwerf-linux-amd64-$VERSION)
- [Darwin amd64](https://dl.bintray.com/flant/$BINTRAY_REPO/$VERSION/multiwerf-darwin-amd64-$VERSION)
- [Windows amd64](https://dl.bintray.com/flant/$BINTRAY_REPO/$VERSION/multiwerf-windows-amd64-$VERSION.exe)
- [SHA256SUMS](https://dl.bintray.com/flant/$BINTRAY_REPO/$VERSION/SHA256SUMS)

<!-- repo: $BINTRAY_REPO -->
