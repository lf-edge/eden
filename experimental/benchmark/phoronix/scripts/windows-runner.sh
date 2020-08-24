#!/bin/bash

# Run phoronix on windows system via SSH

set -e
set -u
set -o pipefail

DIR="$(readlink -f "$(dirname "$0")")"

set -x

#PORT=8022
#HOST=127.0.0.1
#KEY="${DIR}/../id_rsa"
#SUITE=fio-basic

if [[ -z "${PORT}" ]]; then
		echo "PORT environment variable must be defined." >&2
		exit 1
fi
if [[ -z "${HOST}" ]]; then
		echo "HOST environment variable must be defined." >&2
		exit 1
fi
if [[ -z "${KEY}" ]]; then
		echo "KEY environment variable must be defined." >&2
		exit 1
fi
if [[ -z "${SUITE}" ]]; then
		echo "SUITE environment variable must be defined." >&2
		exit 1
fi
USER=IEUser
TEST_ITERATIONS=1
OUTPUT="${1}"

BASE_SSH_OPTS="-o IdentitiesOnly=yes -i ${KEY}"
SCP_CMD="scp $BASE_SSH_OPTS -P $PORT"
SSH_CMD="ssh $BASE_SSH_OPTS -p $PORT ${USER}@${HOST}"

# wait for ssh
while sleep 1; do
		$SSH_CMD dir && break
done

# shellcheck disable=SC2016
SYSTEMDRIVE="$($SSH_CMD 'echo $Env:SYSTEMDRIVE' | cut -c1-2)"

PHORONIX_COMMAND="${SYSTEMDRIVE}\\phoronix-test-suite\\phoronix-test-suite.bat"
TEST_RESULT_COMMAND="gci ${SYSTEMDRIVE}\\users\\${USER}\.phoronix-test-suite\test-results | sort LastWriteTime | select -last 1 | format-table -hidetableheaders Name"
SUITE_PATH="${SYSTEMDRIVE}/Users/${USER}/.phoronix-test-suite/test-suites/local/suite/suite-definition.xml"

# copy suite definition to windows
$SCP_CMD "${DIR}/suites/${SUITE}.xml" ${USER}@${HOST}:"${SUITE_PATH}"

# Install tests
$SSH_CMD "${PHORONIX_COMMAND} batch-install suite"
# Set number of times to run test
$SSH_CMD "SetX FORCE_TIMES_TO_RUN ${TEST_ITERATIONS}"
# Run tests
$SSH_CMD "${PHORONIX_COMMAND} batch-run suite"

# Get test result directory
TEST_RESULT_NAME="$($SSH_CMD "$TEST_RESULT_COMMAND" | grep -Po '[^\s]+')"
TEST_RESULT="${SYSTEMDRIVE}/Users/${USER}/.phoronix-test-suite/test-results/${TEST_RESULT_NAME}/composite.xml"
$SCP_CMD ${USER}@${HOST}:"${TEST_RESULT}" "${OUTPUT}"
