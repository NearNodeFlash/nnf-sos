# Building with Podman

If you have podman configured and running in your environment, then you may
use it to build your containers with the following:

```bash
export DOCKER=podman

# Then...
make docker-build
# or
./playground.sh docker-build
```

Use your usual docker commands with podman.  The podman commandline is
compatible with docker's commandline.

## Setup Podman on a Macbook

You should quit the Docker application on your Mac before you start the
podman machine.  Let podman have the memory and CPU that the docker VM was
using.  Likewise, stop the podman machine before restarting the Docker
application.

The podman machine is where your container image cache lives.  You'll lose that
cache if you ever remove the podman machine.  The same is true for your Docker
container image cache, but who ever thinks of removing docker?

### Install or upgrade podman

```bash
brew install podman
# or
brew upgrade podman
```

### Initialize Podman Machine

Set up a podman machine that is big enough to build nnf-sos.  The following
parameters match those that we use for our docker runtime environment.  This
needs to be done only once.

```bash
podman machine init --cpus 6 --memory 2048 --disk-size 60
```

Start the podman machine with the following.  If you're not using the
Docker application then you may leave your podman machine running at all times,
just as you did for your Docker application.

```bash
podman machine start
```

If you need to stop the podman machine:

```bash
podman machine stop
```

To remove the podman machine from your Mac, use the following command.  Note
that you'll lose your container image cache if you do this.  There really
should be no reason to do this, unless you need to resize it with a new
`machine init` command:

```bash
podman machine rm
```

## Podman with Kubernetes-in-Docker (KIND)

NOTE: As of podman 3.4.4 (latest on 1/26/22), there is a Mac-related
podman bug <https://github.com/containers/podman/pull/12283> that prevents
`kind create` from succeeding.  The bug is fixed, but is not yet in a release.

To cause KIND to use podman, set the following environment variable and
then run your usual KIND commands:

```bash
export KIND_EXPERIMENTAL_PROVIDER=podman
```
