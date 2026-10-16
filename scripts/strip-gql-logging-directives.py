#!/usr/bin/env python3
"""Drop server-only @secret and @allowLogging markers from a copied GraphQL schema.

The directives tell the Observe server which values to redact from its own
telemetry. They do not change the value a client receives. Description text
that mentions the directives is left in place.
"""

import pathlib
import re
import sys

DIRECTIVE = re.compile(r"directive @(?:secret|allowLogging) on\s*$")
# A use sits before another directive, a trailing comment, a comma, a closing
# parenthesis, or the end of the line. Prose such as "is @secret." does not.
USE = re.compile(r" @(?:secret|allowLogging)(?=(?:[ \t]*[@#]|[,)]|$))", re.M)


def strip_logging_directives(text: str) -> str:
    lines = text.splitlines(keepends=True)
    kept = []
    index = 0
    while index < len(lines):
        if DIRECTIVE.match(lines[index]):
            # Drop only the """ block that ends on the previous line.
            if kept and kept[-1].strip() == '"""':
                opener = len(kept) - 2
                while opener >= 0 and kept[opener].strip() != '"""':
                    opener -= 1
                if opener >= 0:
                    del kept[opener:]
            index += 1
            while index < len(lines) and lines[index].startswith("    | "):
                index += 1
            continue
        kept.append(lines[index])
        index += 1
    return USE.sub("", "".join(kept))


def main() -> None:
    root = pathlib.Path(sys.argv[1])
    for path in root.rglob("*.graphql"):
        original = path.read_text()
        updated = strip_logging_directives(original)
        if updated != original:
            path.write_text(updated)


if __name__ == "__main__":
    main()
