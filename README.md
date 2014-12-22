libsquash
=========

[![GoDoc](https://godoc.org/github.com/rafecolton/libsquash?status.svg)](https://godoc.org/github.com/rafecolton/libsquash)

This is based on
[https://github.com/jwilder/docker-squash](https://github.com/jwilder/docker-squash),
but the squashing functionality is extracted into a library that runs
the extraction without writing any files to disk.

It can also be used as an executable, although that functionality is
intended only as an example.

See [Squashing Docker Images](http://jasonwilder.com/blog/2014/08/19/squashing-docker-images/)

[Sample output](_docs/sample-output.md)

## Installation

```bash
go install ./docker-squash
```

## Usage

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

## Known Issues

Currently, squashing an image may remove or invalidate the `ENTRYPOINT` and `CMD`
layers - they may need to be specified on the command line when running
the image.

## License

MIT
