# eserver

In addition to the obvious components - eve itself, adam as a controller,
redis as a backing store for adam - eden starts up a Web server
called `eserver`.

eserver simply serves up files via http/https. That is its job.

However, it is needed because eve needs to retrieve images over the network to:

* update its base OS
* launch VMs
* launch containers

eden uses eserver to make any of these files, other than the actual docker
containers from an OCI registry, available to eve. It uses eserver
rather than direct eve access to the files for several reasons:

* eve requires file size and sha256 hash, which not every
Web endpoint can provide
* eden does not want eve to have to go out to the Internet
for the every repeated request, and so caches it locally
* eve cannot retrieve local files, which eden may use

To solve all of these, all files passed to eve are shared via eserver, which:

* caches files from the Internet
* shares local files
* calculates sha256 hash and file size
