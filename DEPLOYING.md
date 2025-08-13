# Deploying the application for test purposes

> [!IMPORTANT]
> This project is not recommended for production use and it is tuned to make
> test and debug while developing easier. Some shortcuts regarding permissions
> were taken, please inspect the YAML files under the `helm` directory if you
> want to see what is being deployed.

## Prerequisites

- A [kind](https://kind.sigs.k8s.io/) cluster (or similar).
- Admin permissions on the cluster.
- The [Docker](https://www.docker.com/) CLI installed.
- The [Helm](https://helm.sh/) CLI installed.

## TLDR

```bash
$ make build-image
$ kind load docker-image kombiner:latest
$ make install
```

## Building the images

The following command will build the needed image and push it to the specified
registry (use this if you want to use a remote registry):

```bash
$ make build-image-and-push IMAGE_NAME=quay.io/me/kombiner IMAGE_TAG=latest
```

Optionally you can save the image locally if you want to load it directly into
the node's container runtime:

```bash
$ make build-image-and-save
```

The resulting image will be saved in the `_output/images` directory, from there
you can load it using `kind load docker-image`.

## Deploying

To deploy it in the cluster pointed by your `kubectl` context use:

```bash
$ make install IMAGE_NAME=quay.io/me/kombiner IMAGE_TAG=latest
```

This will use `helm` to install the chart in the `kube-system` namespace.

> [!IMPORTANT]
> Helm charts do not support CRD updates so once the chart is installed you
> will need to change them manually if needed. CRD deletion may also needed
> after uninstalling the chart.

To uninstall use:

```bash
$ make uninstall
```

> [!NOTE]
> Both the controller and the scheduler processes are deployed on the same pod.
> The scheduler is configured to process pods that use the 'kombiner-scheduler'
> scheduler name.
