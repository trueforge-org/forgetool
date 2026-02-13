#!/usr/bin/env bash
set -euo pipefail

echo "[init_cluster_env] Starting cluster environment initialization"

SOURCE_DIR="testdata/cluster_test"
TARGET_CLUSTER_DIR="clusters/main"
TARGET_TALOS_DIR="${TARGET_CLUSTER_DIR}/talos"
SOURCE_CLUSTER_ENV="${SOURCE_DIR}/clusterenv.yaml"
SOURCE_TALCONFIG="${SOURCE_DIR}/talconfig.yaml"
TARGET_CLUSTER_ENV="${TARGET_CLUSTER_DIR}/clusterenv.yaml"
TARGET_TALCONFIG="${TARGET_TALOS_DIR}/talconfig.yaml"

echo "[init_cluster_env] Verifying source template files"
if [[ ! -f "${SOURCE_CLUSTER_ENV}" ]]; then
  echo "[init_cluster_env] ERROR: Missing source file: ${SOURCE_CLUSTER_ENV}"
  exit 1
fi
if [[ ! -f "${SOURCE_TALCONFIG}" ]]; then
  echo "[init_cluster_env] ERROR: Missing source file: ${SOURCE_TALCONFIG}"
  exit 1
fi

echo "[init_cluster_env] Replacing target files with testdata templates (pre-init)"
mkdir -p "${TARGET_TALOS_DIR}"
cp "${SOURCE_CLUSTER_ENV}" "${TARGET_CLUSTER_ENV}"
cp "${SOURCE_TALCONFIG}" "${TARGET_TALCONFIG}"



echo "[init_cluster_env] Replacing target files with testdata templates (post-init)"
mkdir -p "${TARGET_TALOS_DIR}"
cp "${SOURCE_CLUSTER_ENV}" "${TARGET_CLUSTER_ENV}"
cp "${SOURCE_TALCONFIG}" "${TARGET_TALCONFIG}"

echo "[init_cluster_env] Verifying replaced files match testdata"
if ! cmp -s "${SOURCE_CLUSTER_ENV}" "${TARGET_CLUSTER_ENV}"; then
  echo "[init_cluster_env] ERROR: ${TARGET_CLUSTER_ENV} does not match ${SOURCE_CLUSTER_ENV}"
  exit 1
fi
if ! cmp -s "${SOURCE_TALCONFIG}" "${TARGET_TALCONFIG}"; then
  echo "[init_cluster_env] ERROR: ${TARGET_TALCONFIG} does not match ${SOURCE_TALCONFIG}"
  exit 1
fi

echo "[init_cluster_env] Replacement complete"
