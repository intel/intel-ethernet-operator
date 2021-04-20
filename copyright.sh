# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2020-2021 Intel Corporation

#!/bin/bash

FOLDER=${FOLDER:-.}
COPYRIGHT_FILE=${COPYRIGHT_FILE:-COPYRIGHT}
copyright=`cat "$COPYRIGHT_FILE"`

for file in `find ${FOLDER} -name '*.yaml'`
do
  if ! grep -q "${copyright}" "$file"
  then
    echo "$file"
    cat "$COPYRIGHT_FILE" "$file" >"$file".new && mv "$file".new "$file"
  fi
done
