# Tigonza's Fork

This is a fork for the v2.0.6 tag of clair. It brings changes to small interactions within the main package 
along with some changes within its utility packages. It mostly focuses on adding new drivers and expanding
the amounts of data that clair is capable of collecting. refer to the documentation on the README.md on the 
main directory for info on the development of drivers for clair.

## Data Model Customizations

The data model has been modified so that there can be more than one active feature driver collecting data at the time. This change enforces the possibility of more than one version format linked to each layer. To achieve this the pgsql packages(were the migrations occur) now also add an extra versionfmt column to each featureversion inserted in the DB. Now each feature is linked through its respective featurefmt driver to its own version format. This change also implies that the database packages in which the pgsql predifined functions are used have had their methods modified/extended so that they can deal with the extra input/output.

## Drivers

This fork presents the addition of 3 new drivers to the v2.0.6 version of clair



## Usage

This version of clair's API functions just the same as the [original  of the same version](https://github.com/tigonza/clair/Documentation/api_v1.md) would, with the exception of the added possibility of having another "image format".  This refering to the .simg archiving standard for singularity. 

A simple example of the body of a POST query that pushes a singularity-hub image to clair would be:

```json
{"Layer":{
	"Name":"CF6393C20357A8003115ADED3D873D7C543387AF7BEF6A93FBB578FF09EF6ED5",
	"Path":"shub://jdwheaton/singularity-ngs",
	"ParentName":"",
	"Format":"Singularity",
	}
}

```

Since singularity doesn't add up the layers on top of each other, opting instead for an entire image, we push it to clair as a one layer image.
In short, just change the format from Docker to Singularity and dont add the hash for the parent layer.

<!-- 
Clair can also be compiled with custom notifiers by importing them in `main.go`.
Custom notifiers are any Go package that implements the `Notifier` interface and registers themselves with the `notifier` package.
Notifiers are registered in [init()] similar to drivers for Go's standard [database/sql] package.

[init()]: https://golang.org/doc/effective_go.html#init
[database/sql]: https://godoc.org/database/sql -->
