libsquash
=========

## Library

[![GoDoc](https://godoc.org/github.com/winchman/libsquash?status.svg)](https://godoc.org/github.com/winchman/libsquash)

This is based on
[jwilder/docker-squash](https://github.com/jwilder/docker-squash),
but the squashing functionality is extracted into a package named
`libsquash`

`libsquash` is different from `docker-squash` in that `libsquash`...

0. does not write to disk
0. does not require `sudo`
0. does not depend on the installation of any particular version of
   `tar` (or any version at all)

Other information:

* Article: [Squashing Docker Images](http://jasonwilder.com/blog/2014/08/19/squashing-docker-images/)
* [Sample output](_docs/sample-output.md)

## Binary

### Installation

```bash
go get github.com/winchman/libsquash/docker-squash
```

### Usage

docker-squash works by squashing a saved image and loading the squashed image back into docker.

```
$ docker save <image id> > image.tar
$ sudo docker-squash -i image.tar -o squashed.tar
$ cat squashed.tar | docker load
$ docker images <new image id>
```

You can also tag the squashed image:

```
$ docker save <image id> > image.tar
$ sudo docker-squash -i image.tar -o squashed.tar -t newtag
$ cat squashed.tar | docker load
$ docker images <new image id>
```

You can reduce disk IO by piping the input and output to and from docker:

```
$ docker save <image id> | sudo docker-squash -t newtag | docker load
```

If you have a sufficient amount of RAM, you can also use a `tmpfs` to remove temporary
disk storage:

```
$ docker save <image_id> | sudo TMPDIR=/var/run/shm docker-squash -t newtag | docker load
```

By default, a squashed layer is inserted after the first `FROM` layer.  You can specify a different
layer with the `-from` argument.
```
$ docker save <image_id> | sudo docker-squash -from <other layer> -t newtag | docker load
```
If you are creating a base image or only want one final squashed layer, you can use the
`-from root` to squash the base layer and your changes into one layer.

```
$ docker save <image_id> | sudo docker-squash -from root -t newtag | docker load
```

## License

MIT
