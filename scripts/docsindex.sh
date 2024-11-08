#!/usr/bin/env bash

filepath="$1"
# This script appends table of contents tree to end of index file for docs synchronization
cat <<'EOF' >>"$filepath"


## Templates

### Data Sources
```{toctree}
---
maxdepth: 1 
glob: true
---
data-sources/app
data-sources/*
```
### Resources

```{toctree}
---
maxdepth: 1 
glob: true
---
resources/app_datasource
resources/*
```
EOF