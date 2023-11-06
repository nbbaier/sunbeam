#!/usr/bin/env deno run -A

import * as path from "https://deno.land/std@0.205.0/path/mod.ts";
import * as sunbeam from "../pkg/typescript/src/manifest.ts";

const dirname = new URL(".", import.meta.url).pathname;
const rows = []

rows.push(
    "---",
    "sidebar: false",
    "outline: 2",
    "---",
    "",
    "# Extension Catalog"
)


const extensionDir = path.join(dirname, "..", "extensions");
const entries = Deno.readDirSync(extensionDir);
for (const entry of entries) {
    const entrypoint = path.join(extensionDir, entry.name);
    const command = new Deno.Command(entrypoint)
    const { stdout } = await command.output()

    const manifest: sunbeam.Manifest = JSON.parse(new TextDecoder().decode(stdout));
    rows.push(
        "",
        `## [${manifest.title}](https://github.com/pomdtr/sunbeam/tree/main/extensions/${entry.name})`,
        "",
        `${manifest.description}`,
    )

    if (manifest.requirements?.length) {
        rows.push(
            "",
            "### Requirements",
            ""
        )

        for (const requirement of manifest.requirements) {
            rows.push(
                requirement.link ? `- [\`${requirement.name}\`](${requirement.link})` : `- \`${requirement.name}\``
            )
        }
    }

    rows.push(
        "",
        "### Commands",
        ""
    )

    for (const command of manifest.commands) {
        rows.push(
            `- \`${command.name}\`: ${command.title}`
        )
    }

    rows.push(
        "",
        "### Installation",
        "",
        "```",
        `sunbeam extension install https://raw.githubusercontent.com/pomdtr/sunbeam/main/extensions/${entry.name}`,
        "```"
    )
}

console.log(rows.join("\n"))
