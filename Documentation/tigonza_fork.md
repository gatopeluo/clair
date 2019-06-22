# Tigonza's Fork

This is a fork for the v2.0.6 tag of clair. It brings changes to small interactions within the main package 
along with some changes within its utility packages. It mostly focuses on adding new drivers and expanding
the amounts of data that clair is capable of collecting. refer to the documentation on the README.md on the 
main directory for info on the development of drivers for clair.

## Data Model Customizations

The data model has been modified so that there can be more than one active feature driver collecting data at the time. This change enforces the possibility of more than one version format linked to each layer. To achieve this the pgsql packages(were the migrations occur) now also add an extra versionfmt column to each featureversion inserted in the DB. Now each feature is linked through its respective featurefmt driver to its own version format. This change also implies that the database packages in which the pgsql predifined functions are used have had their methods modified/extended so that they can deal with the extra input/output.

## Drivers

This fork presents the addition of 4 new drivers to the v2.0.6 version of clair. The first two are featurefmt drivers, and the others are an imagefmt driver and a vulnsrc driver.

### Features

The drivers that extract the features on this project funciton as threads independent of eachother. For each driver type there is a manager that contains the list of such interfaces. At init(), all drivers get registered by its manager. During the posting of a Layer, it is the driver manager which is in charge of the data given to each driver for analysis. This last point is especcially important for the first two drivers, which are featurefmt drivers. Given the nature of this package managers, it's possible to have development enviroments that would change the position of the manager of the libraries itself, rendering the mechanism for filtering sensitive files useless. So just in case that this drivers are initialized, the behaviour for obtaining this sensitive files changes and acts a bit more thoroughly, going through more files names while looking for clues of a develompent enviroment, choosing to use everything within them.

The pip driver checks the full filename ( this includes the route for it within the FS ) for any possible clue of a python 2.7/3.6 enviroment and adds all files that seem to be in it as input. The driver checks for .egg-info or .dist-info files which have the standarized metadata of each package, and generates structs accordingly.

The npm driver works the same, with the difference that it looks for an "node_modules/" folder, that will encompass an npm dev env. Within each folder that exists inside "node_modules/" theres info on a particular module/library that has parsable metadata.

### Vulnerabilities

The third driver is a vulnerability source driver that adds the parsed information of an open-DB of security information for pip packages called safety-db.

Vulnerabilities in clair are linked to two entities: 
	- The namespace (OS)
	- and the feature.

Safety-db doesnt have info on influenced OSs, since the vulnerabilities are just linked to the packages. To solve this we just add all namespaces known to the list, and add the vulnerabilities multiple times. This proves to require much less work than reworking the whole of the vulnerabilities model.

### Image Format

More than a new driver, a new behaviour was added to the driver manager for the explicit case of recieving a request with "Singularity" as "Format". It parses the "Path" property of the request's body, recognizing it as an singularity-hub reference. The driver itself doesnt do much more than actually initializing the manager's behaviour. 

Several packages from singularity have been added to the vendor folder, alongside an utlis package that compiles a handful of methods to deal with the interactions with the SHUB API. 

Clair now pulls the .simg file from singularity-hub service, unfolds and archives the filesystem within into a .tar file that then gets analyzed just as any serviced .tar/.tar.gz file would. 





## Usage

This version of clair's API functions just the same as the [original  of the same version](https://github.com/tigonza/clair/blob/Devel2.0/Documentation/api_v1.md) would, with the exception of the added possibility of having another "image format".  This refering to the .simg archiving standard for singularity. 

A simple example of the body of a POST query that pushes a singularity-hub image to clair would be:

```http
HTTP/1.1 POST
```
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
