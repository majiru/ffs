# Fake File System(ffs)

## Purpose
To create a simple interface for users to implement in memory filesystems.

## Use
Any struct that implementes the ffs.Fs interface detailed in ffs.go can make
use of the server package to serve its files over HTTP and 9p.

The fsutil package implements in-memory files that are compatible with the ffs.Writer
and ffs.File interface. The os package's File is compatible with the ffs.File interface as well.

## Examples
The diskfs package serves file from an arbitrary root path.

The domainfs package switches sub file systems based on the requested http host.

## Inspiration
https://talks.golang.org/2012/10things.slide#8

https://github.com/droyo/jsonfs

