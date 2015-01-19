libsquash
=========

## Library

[![GoDoc](https://godoc.org/github.com/winchman/libsquash?status.svg)](https://godoc.org/github.com/winchman/libsquash)

This is based on
[jwilder/docker-squash](https://github.com/jwilder/docker-squash),
but the squashing functionality is extracted into a package named
`libsquash`

`libsquash` is different from `docker-squash` in that `libsquash`...

0. does not write any layer data to disk
0. does not require `sudo`
0. does not shell out to or require the installation of `tar`

Other information:

* Article: [Squashing Docker Images](http://jasonwilder.com/blog/2014/08/19/squashing-docker-images/)
* [Sample output](_docs/sample-output.md)

## License

MIT
