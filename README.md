ClamAV Private Mirror
=====================

This project is intended to be a collection of stand-alone tools and unified
components that allow for the easy install of a private ClamAV signature mirror.

## Components

### sigupdate

The sigupdate component works in a similar way to the perl script posted on the
[ClamAV site called `clamdownloader.pl`](https://www.clamav.net/documents/private-local-mirrors).
However, it is written in golang and doesn't require any application runtimes
(Perl) in order to run. Additionally, it handles some failure cases better. Typically,
you would use this tool in a cron job to periodically update your signatures and then
use your own webserver to serve those signatures out of your data directory.

#### Usage

```
Usage: sigupdate [-vV] [-d value] [-m value] [-t value] [parameters ...]
 -d, --data-file-path=value
                Path to ClamAV data files
 -m, --download-mirror-url=value
                URL to download signature updates from
 -t, --diff-count-threshold=value
                Number of diffs to download until we redownload the
                signature files
 -v, --verbose  Enable verbose mode with additional debugging information
 -V, --version  Display the version and exit        
```

##### Data Directory
Usage of the sigupdate utility requires a directory that is writable by the executing
user. Specify that directory with the `-d` flag.

##### Download Mirror URL
The default URL for downloading updates is `http://database.clamav.net`. However, 
that URL doesn't make use of the global mirrors available. Please refer to the [ClamAV
mirrors page](https://www.clamav.net/documents/mirrors) for a list of region specific
mirrors. For example, if you are in the USA, you may want to use the URL: 
`http://db.us.clamav.net` to download signatures from a list of mirrors in the US region.

##### Diff Count Threshold
Sometimes your signatures may be so out of date that there are not enough diff files
available on the mirrors to provide updates. In particular, this happens if you start
with definitions that come directly from a package manager. This value sets the number
of versions to download diffs for until we update the base signature data file.

### sigserver

The sigserver component is a stand-alone HTTP server that serves ClamAV signatures. 
Signatures will be updated periodically  using the sigupdate component. The update
interval will be configurable.

## License

This project is licensed under the MPLv2. Please see the LICENSE file for more details.

## Credits

I'm grateful for all of the hard work that goes into ClamAV. Thank you!