# METADATA
# title: conflict2
# custom:
#   key: value
#   tags: ["foo", "bar"]

package metadata_test

import rego.v1

resp contains "a" if {
    input.path == "/tmp/data"
}
