#!/bin/bash -xe

source_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null && pwd)"
cd "$source_dir/.."

dirty_tree() {
  [[ $(git status --porcelain=v1 2>/dev/null | wc -l) != 0 ]]
}

# We put this stupid check in here because the GQL client generation library
# is silently non-deterministic if there are inconsistent annotations across
# multiple operations
# This can be removed once https://github.com/Khan/genqlient/issues/123 is addressed
for _ in {1..30}; do
  make gen-gql-client
  if dirty_tree; then
    cat 1>&2 << EOF
*** BUILD WILL FAIL: GQL client generation has created a dirty tree; this indicates that \
there are annotations on GQL operations that apply to some instances of \
a given type, but not to all instances. \
Find all instances of the type affected in the diff below, then ensure that \
the same schema annotations are used for each one. ***
EOF
    git diff
    exit 1
  fi
done
