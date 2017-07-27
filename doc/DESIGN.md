# Design

This document attempts to describe at a high-level the implementation of both 
the `sigupdate` and `sigservers` applications.

## Signature Update Design (`sigupdate`)

The step-by-step process by which signatures are updated from public ClamAV 
mirrors is documented below.

1.  The `sigtool` utility is located in the working directory or in the system
    PATH. If it is not found, the application exits.
2.  The TXT record (typically current.cvd.clamav.net) for ClamAV signature 
    versions is retrieved from DNS.
3.  Each version value is parsed from the TXT record and stored in a data 
    structure.
4.  For each signature (main, daily, bytecode), we check to see if a .cvd file
    has already been downloaded for it. If it hasn't been downloaded, we download
    it.
5.  If the signature has already been downloaded, we use `sigtool` to parse the
    version and build time of the file and store the values. If `sigtool` fails
    to parse the file or the file fails verification, we download the file again.
6.  A .cdiff file per each version number of the delta is downloaded. 
    If any of the .cdiff files can't be downloaded (e.g. 403 Not Found), then 
    the .cvd file is downloaded again. If the new signature version the old 
    signature version are not within the range of an acceptable delta 
    (set by `diff-count-threshold`), then we download the .cvd file.
7.  When downloading files, we download all files to a temporary file and move
    the file to the final location only if all operations succeed.
8.  When downloading .cvd files, we specify the "If-Modified-Since" HTTP header
    with the value based on the build time as returned from the output of
    `sigtool`.
9.  When a .cvd file download has completed, we parse the file with `sigtool`
    and set the file system's last modified time to the value returned from
    the `sigtool`. Additionally, if the file fails verification, then it is
    considered a failed download.    
10. When a non-cvd file is downloaded, the file system's last modified time is
    set to the value as returned by the HTTP header "Last-Modified".
11. Once all signature file updates have been completed, the `sigupdate` process
    has finished.
