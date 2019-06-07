# Fake File System

## Purpose
To create a simple interface for users to implement in memory filesystems.

## Use
Any struct that implementes the ffs.Fs interface detailed in ffs.go can make
use of the server package to serve its files over HTTP and 9p.

The fsutil package implements in-memory files that are compatible with the ffs.Writer
and ffs.File interface. The *os.File struct also implements both of these as well.

## Filesystems
* Diskfs: Serve arbitrary folder from the host OS.
* Pastefs: A fileserver for saving and sharing text snippets.
* MKVfs: Creates files and folders for exploring mkv file structure.
* Domainfs: Mux's between sub filesystem based on http header, or folders over 9p.
* Mediafs: Filesystem counterpart to [anidb2json](https://github.com/majiru/anidb2json).
* Jukeboxfs: Parses directory to create file tree based on audio file metainfo

## Example
`cmd/ffs/main.go` shows simple use of almost all filesystems, with limited command line configuration.

## Inspiration
https://talks.golang.org/2012/10things.slide#8

https://github.com/droyo/jsonfs

