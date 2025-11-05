# Schemas

Must ensure that schema dependencies are loaded before they are used.
File names should be sorted alphabetically so this is true.

## Changes to Schemas

When modifying schemas a full copy of all schemas will need to be taken.
And placed into a new versioned folder e.g v5, v6...
This minimises issues with nested changes, (only one model change should be needed per Azul release).

When making changes to the schema ensure they are backwards compatible by only adding or removing fields.
Any new fields added need to have a default, in case the old version is being read.

`avro.go` will also need to be updated with the new version information.
Ensure the new `AvroSchemaVersion` is added.
The list of `AvroSchemaVersion`'s has the new version added (method `getAllSchemaVersions`)
And `CurrentModelVersion` is updated appropriately with the newest schema version.

### Test cases

Ensure to add a regression test into `event_test.go` that checks the old avro file version can be loaded into the new format.
Tests should be added for the regular and bulk models.
e.g `TestLegacyV4Stability` and `TestLegacyV4StabilityBulk`

This test loads the old avro and compares it against the existing json file.
Note the version will be bumped to the new version when doing so, so the version in the json will need to be updated.

These regression tests should be kept until the model version is completely removed from the system.
