#!/usr/bin/env python3

import json
import pathlib
import argparse
import os
import sys

parser = argparse.ArgumentParser()
parser.add_argument("--root", required=True, type=pathlib.Path)
args = parser.parse_args()

root: pathlib.Path = args.root
entries = root.iterdir()
if os.getenv("SHOW_HIDDEN") != "1":
    entries = filter(lambda p: not p.name.startswith("."), entries)

json.dump(
    {
        "type": "list",
        "items": [
            {
                "title": path.name,
                "accessories": [str(root.absolute())],
                "actions": [
                    (
                        {
                            "type": "open-url",
                            "url": f"file://{path.absolute()}",
                            "title": "Open File",
                        }
                        if path.is_file()
                        else {
                            "type": "run-command",
                            "command": "browse-files",
                            "title": "Browse Directory",
                            "with": {"root": str(path.absolute())},
                        }
                    ),
                    {
                        "type": "run-command",
                        "title": "Edit",
                        "command": "edit-file",
                        "with": {"file": str(path.absolute())},
                    },
                    {
                        "type": "copy-text",
                        "title": "Copy Path",
                        "shorcut": "ctrl+y",
                        "text": str(path.absolute()),
                    },
                    {
                        "type": "run-command",
                        "title": "Delete File",
                        "command": "delete-file",
                        "with": {"path": str(path.absolute())},
                    },
                ],
            }
            for path in sorted(entries, key=lambda p: p.name)
        ],
    },
    sys.stdout,
)
