#!/usr/bin/env bash

# Copyright (c) 2021 Anton Semjonov
# Licensed under the MIT License

# Print version information for a project manager with Git, working
# both in a checked-out repository or an exported archive file.
# For more information see: https://github.com/ansemjo/version.sh

# Note: this script requires Git version 2.32.0 or later.
# tl;dr: add "version-modern.sh export-subst" to .gitattributes

# Ignore shellcheck warnings for '$Format:..$', which looks
# like a constant / variable in single-quotes but isn't!
# shellcheck disable=SC2016,SC2050

# configure some strings, use env if given
ALWAYS_LONG_VERSION="${ALWAYS_LONG_VERSION-y}"
REVISION_SEPARATOR="${REVISION_SEPARATOR--}"
HASH_SEPARATOR="${HASH_SEPARATOR--g}"
DIRTY_MARKER="${DIRTY_MARKER--dirty}"

# get interesting strings
if [[ '$Format:%%$' = '%' ]]; then
  # exported archive, replaced values
  hash='$Format:%H$'
  version='$Format:%(describe:exclude=*-[0-9]*-g[0-9a-f]*)$'
  if [[ "${version:0:11}" = '%(describe:' ]]; then
    echo "warning: exporting git was too old! version >= 2.32.0 required" >&2
    version=""
  fi
  dirty=''
else
  # otherwise hopefully a live git repo
  git rev-parse >/dev/null || exit 1
  hash=$(git log -1 --pretty='%H') || exit 1
  version=$(git describe --exclude='*-[0-9]*-g[0-9a-f]*' 2>/dev/null)
  dirty=$(git diff-index --quiet HEAD -- || printf '%s' "${DIRTY_MARKER}")
fi

# reformat version string to match `git describe --always --long`
# and use configured separators if any, append dirty marker
if [[ -z "${version}" ]]; then
  version="${hash:0:7}"
elif [[ "${version}" =~ (.*)-([0-9]+)-g([0-9a-f]+)$ ]]; then
  re=("${BASH_REMATCH[@]}")
  version="${re[1]}${REVISION_SEPARATOR}${re[2]}${HASH_SEPARATOR}${re[3]}"
elif [[ "${ALWAYS_LONG_VERSION}" = 'y' ]]; then
  version="${version}${REVISION_SEPARATOR}0${HASH_SEPARATOR}${hash:0:7}"
fi
version="${version}${dirty}"

# parse commandline argument
case "$1" in
  '')
    echo "${hash} ${version}" ;;
  commit|hash)
    echo "${hash}" ;;
  version|describe)
    echo "${version}" ;;
  json)
    printf '{\n  "version": "%s",\n  "commit": "%s"\n}\n' \
      "${version//\"/\\\"}" "${hash//\"/\\\"}" ;;
  env)
    printf "VERSION=%q\nCOMMIT=%q\n" \
      "${version}" "${hash}" ;;
  *)
    printf '%s [version|commit|json|env]\n' "$0"; exit 1 ;;
esac
