# Base manifest

Explanation on how was the `config/manifests/bases/ibm-license-service-scanner-operator.clusterserviceversion.yaml`
file created.

## Template

### Metadata

Mostly copied from the `reporter` repository.

TODO: `olm.skipRange` parameters must be added in the feature to disable referring to development sources.

### Spec

Mostly copied from the `reporter` repository.

Some values were adjusted to refer to the `scanner`.

TODO: Most parameters will need further adjustments, e.g. to provide an accurate description of the Scanner operator.

### Example

TODO: Once released, provide an example of the YAML file.

## Dynamic values

Some values in the file can be auto-updated when running the `update-version` makefile target. This automation is
related to the release process.

Furthermore, related images are also adjusted on the development build (to point to the correct registry).
