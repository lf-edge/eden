#!/usr/bin/env python3


# dependencies
# python3-docker
# python3-docopt
# python3-github

""" eve-image.

    Uploads latest kvm eve image to zedcloud configured in
      ${HOME}/.config/zededa/zcli.json
    as name <tag> title eve-<commit timestamp>-<tag>

Usage:
  eve-image

"""

import datetime
import logging
import json
import os
import sys
import tempfile

import docker
import docopt
import github

logging.getLogger().setLevel(logging.INFO)

EVE_BRANCH = "master"
EVE_GITHUB_REPO = "lf-edge/eve"
EVE_DOCKER_REPO = "lfedge/eve"

EVE_TAG_SHA_LENGTH = 8
ZCLI_CONFIG_FILE = os.path.join(os.environ["HOME"],
                                ".config",
                                "zededa",
                                "zcli.json")
ZCLI_DOCKER_REPO = "zededa/zcli"
ZCLI_TAG = "latest"
ZCLI_DATASTORE = "Zededa-AWS-Image"


def pull_image(docker_client, commit, hypervisor="kvm", arch="amd64"):
    """ Pull latest image available for given git commit
        Will fall back to older commits if the current commit is not present.
        Arguments:
            docker_client : docker.client.DockerClient
            commit: github.Commit.Commit
            hypervisor: str
            arch: str
        Returns:
           (None, None, None) or tuple of
           docker.models.images.Image
           str of tag name pulled
           datetime.datetime
    """
    sha = commit.sha[0:EVE_TAG_SHA_LENGTH]
    tag_name = f"0.0.0-snapshot-master-{sha}-{hypervisor}-{arch}"
    logging.debug(f"Trying {tag_name}")
    try:
        image = docker_client.images.pull(EVE_DOCKER_REPO, tag_name)
        commit_time = datetime.datetime.strptime(commit.stats.last_modified,
                                                 "%a, %d %b %Y %H:%M:%S %Z")
        return (image, tag_name, commit_time)
    except docker.errors.NotFound:
        logging.info(f"Tag {tag_name} not available")
        for parent in commit.parents:
            result = pull_image(docker_client, parent, hypervisor, arch)
            if result:
                return result
        return None, None


def write_eve_rootfs(docker_client, tag_name, image_format="raw"):
    """ write out eve image from docker image
        Arguments:
            docker_client : docker.client.DockerClient
            tag_name: str of tag of eve image
            image_format: str
        Returns: Tuple of
           tempfile.TemporaryDirectory
           str of absolute filename
    """
    tdir = tempfile.TemporaryDirectory()
    tfilename = f"image.{image_format}"
    container = docker_client.containers.run(f"{EVE_DOCKER_REPO}:{tag_name}",
                                             ["-f", image_format, "rootfs"],
                                             auto_remove=True,
                                             detach=True,
                                             stderr=False,
                                             stdout=True)
    image_data = container.attach(stderr=False,
                                  stdout=True,
                                  stream=True,
                                  logs=True)
    with open(os.path.join(tdir.name, tfilename), 'wb') as tfile:
        for data in image_data:
            tfile.write(data)
    container.stop()
    logging.debug(f"Wrote {tdir.name}/{tfilename}")
    return (tdir, tfilename)


class Zcli():
    def __init__(self, docker_client, volumes):
        docker_client.images.pull(ZCLI_DOCKER_REPO, ZCLI_TAG)
        self.docker_client = docker_client
        self.volumes = {f"{os.path.dirname(ZCLI_CONFIG_FILE)}":
                        {"bind": "/root/.config/zededa", "mode": "rw"}}
        self.volumes.update(volumes)

    def run(self, command):
        result = {}
        result["stderr"] = ""
        result["exit_code"] = 0

        logging.info(" ".join(["zcli"] + command))
        try:
            out = self.docker_client.containers.run(
                f"{ZCLI_DOCKER_REPO}:{ZCLI_TAG}",
                ["zcli"] + command,
                detach=False,
                stdout=True,
                stderr=True,
                remove=True,
                volumes=self.volumes)
        except docker.errors.ContainerError as err:
            result["stderr"] = err.stderr.decode("utf-8")
            result["exit_code"] = err.exit_status
            return result

        out = out.decode("utf-8")
        result["stdout"] = out
        try:
            result.update(json.loads(out))
        except json.JSONDecodeError:
            pass
        return result

    def configure(self):
        # pylint: disable=no-self-use
        config = json.load(open(ZCLI_CONFIG_FILE))
        config["format"] = "json"
        for field in ("userid", "token"):
            if field in config:
                del config[field]
        with open(ZCLI_CONFIG_FILE, 'w') as output:
            json.dump(config, output)


def upload_eve_rootfs(zcli, tag_name, commit_time, image_path,
                      image_format="raw",
                      arch="amd64"):
    timestamp = commit_time.strftime("%s")
    image_name = tag_name
    image_title = f"eve-{timestamp}-{tag_name}"
    result = zcli.run(["image",
                       "show",
                       image_name, ])
    if result["exit_code"] == 0:
        # assume this was uploaded by us
        return True

    result = zcli.run(["image",
                       "create",
                       image_name,
                       f"--title={image_title}",
                       f"--datastore-name={ZCLI_DATASTORE}",
                       f"--arch={arch.upper()}",
                       f"--image-format={image_format}",
                       "--type=Eve", ])
    if result["exit_code"] != 0:
        return False

    result = zcli.run(["image", "upload",
                       image_name,
                       f"--path={image_path}",
                       "--chunked"])
    return result["exit_code"] == 0


def main():
    arguments = docopt.docopt(__doc__)
    branch = github.Github().get_repo(EVE_GITHUB_REPO).get_branch(branch=EVE_BRANCH)
    docker_client = docker.from_env()
    _, tag_name, commit_time = pull_image(docker_client, branch.commit)
    if tag_name:
        logging.info(f"Pulled {tag_name}")
    else:
        logging.error("Failed to download anything")
        sys.exit(1)

    tdir, tfilename = write_eve_rootfs(docker_client, tag_name)
    volumes = {f"{tdir.name}": {"bind": "/images", "mode": "ro"}}
    zcli = Zcli(docker_client, volumes=volumes)
    zcli.configure()
    result = zcli.run(["login"])
    if result["exit_code"] != 0:
        logging.error("Could not log in")
        sys.exit(1)

    upload_eve_rootfs(zcli,
                      tag_name,
                      commit_time,
                      os.path.join("/images", tfilename))

    # zcli edge-node eveimage-update nodename --image=imagename
    # zcli edge-node eveimage-update nodename --image=imagename --activate


if __name__ == '__main__':
    main()
