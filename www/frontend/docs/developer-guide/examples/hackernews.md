---
title: Hacker News
---
This script leverages the `root` property to contribute two items to the root menu.

The first item will list the articles from the front page, and the second item will list the articles from the "Show HN" section.

```ts
#!/usr/bin/env -S deno run -A

import Parser from "npm:rss-parser";
import { formatDistance } from "npm:date-fns";
import * as sunbeam from "https://deno.land/x/sunbeam/mod.ts"

if (Deno.args.length == 0) {
    const manifest: sunbeam.Manifest = {
        title: "Hacker News",
        description: "Browse Hacker News",
        root: ["browse"],
        commands: [
            {
                name: "browse",
                title: "Show a feed",
                mode: "filter",
                params: [
                    { name: "topic", description: "Topic", required: false, default: "frontpage", type: "string" }
                ],
            },
        ]
    };

    console.log(JSON.stringify(manifest));
    Deno.exit(0);
}

const payload = JSON.parse(Deno.args[0]) as sunbeam.CommandInput;
if (payload.command == "browse") {
    const { topic } = payload.params as { topic: string };
    const feed = await new Parser().parseURL(`https://hnrss.org/${topic}?description=0&count=25`);
    const page: sunbeam.List = {
        items: feed.items.map((item) => ({
            title: item.title || "",
            subtitle: item.categories?.join(", ") || "",
            accessories: item.isoDate
                ? [
                    formatDistance(new Date(item.isoDate), new Date(), {
                        addSuffix: true,
                    }),
                ]
                : [],
            actions: [
                {
                    title: "Open in browser",
                    type: "open",
                    url: item.link || "",
                    exit: true
                },
                {
                    title: "Open Comments in Browser",
                    type: "open",
                    url: item.guid || "",
                    exit: true
                },
                {
                    title: "Copy Link",
                    type: "copy",
                    key: "c",
                    text: item.link || "",
                    exit: true
                },
            ],
        })),
    };

    console.log(JSON.stringify(page));
} else {
    console.error("Unknown command");
    Deno.exit(1);
}
```

You can add new sections by adding new items in your config.

```json
{
    "extensions": {
        "hackernews": {
            "origin": "...",
            "items": [ {
                "title": "Show HN",
                "command": "browse",
                "params": {
                    "topic": "show"
                }
            }]
        }
    }
}
```
