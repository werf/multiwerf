# How to create ordinary release

Use `scripts/create_release_tag.sh VERSION` script.

For example to create version `v1.1.8` run following script: `scripts/create_release_tag.sh v1.1.8`.

Script will ask for release message and push a new git tag, which will be published shortly by the "release publisher" github actions workflow.

# How to create experimental release

Merge your branch into `experimental` branch and push, then github actions workflow "experimental releaser" will create a new tag in the form `vYEAR.MONTH.DAY-HOUR.MINUTE.SECOND` (for example `v19.12.25-13.02.39`) and then release will be published by the "release publisher" github actions workflow.
