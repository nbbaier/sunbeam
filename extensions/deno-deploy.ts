#!/usr/bin/env -S deno run -A

import type * as sunbeam from "https://raw.githubusercontent.com/pomdtr/sunbeam/main/deno/mod.ts"
import * as dates from "npm:date-fns"

if (Deno.args.length == 0) {
    const manifest: sunbeam.Manifest = {
        title: "Deno Deploy",
        description: "Manage your Deno Deploy projects",
        root: [
            "projects", "dashboard"
        ],
        preferences: [
            {
                name: "token",
                title: "Access Token",
                type: "text",
                required: true
            }
        ],
        commands: [
            {
                name: "projects",
                title: "List Projects",
                mode: "filter",
            },
            {
                name: "dashboard",
                title: "Open Dashboard",
                mode: "silent"
            },
            {
                name: "deployments",
                title: "List Deployments",
                mode: "filter",
                params: [
                    { name: "project", title: "Project", required: true, type: "text" }
                ]
            },
            {
                name: "playground",
                title: "View Playground",
                mode: "detail",
                params: [
                    { name: "project", title: "Project", required: true, type: "text" }
                ]
            }
        ]
    }

    console.log(JSON.stringify(manifest));
    Deno.exit(0);
}

const payload = JSON.parse(Deno.args[0]) as sunbeam.Payload;
const deployToken = payload.preferences.token;
if (!deployToken) {
    console.error("Missing deploy token");
    Deno.exit(1);
}

try {
    const res = await run(payload);
    if (res) {
        console.log(JSON.stringify(res));
    }
} catch (e) {
    console.error(e);
    Deno.exit(1);
}

async function run(payload: sunbeam.Payload) {
    switch (payload.command) {
        case "dashboard": {
            await new Deno.Command("sunbeam", {
                args: ["open", "https://dash.deno.com"],
            }).output()
            return
        }
        case "projects": {
            const resp = await fetchDeployAPI("/projects");
            if (resp.status != 200) {
                throw new Error("Failed to fetch projects");
            }
            const projects = await resp.json();

            return {
                items: projects.map((project: any) => {
                    const item: sunbeam.ListItem = {
                        title: project.name,
                        accessories: [project.type],
                        actions: [{
                            title: "Open Dashboard",
                            type: "open",
                            url: `https://dash.deno.com/projects/${project.id}`,
                            exit: true,
                        }]
                    }

                    if (project.hasProductionDeployment) {
                        const domains = project.productionDeployment.deployment.domainMappings
                        const domain = domains.length ? domains[domains.length - 1].domain : "No domain"
                        item.subtitle = domain
                        item.actions?.push({
                            title: "Open Production URL",
                            type: "open",
                            url: `https://${domain}`,
                            exit: true,
                        })
                    }

                    item.actions?.push({
                        title: "Copy Dashboard URL",
                        type: "copy",
                        key: "c",
                        text: `https://dash.deno.com/projects/${project.id}`,
                        exit: true,
                    })


                    return item
                })
            } as sunbeam.List;
        }
        case "playground": {
            const name = payload.params.project as string;
            const resp = await fetchDeployAPI(`/projects/${name}`);
            if (resp.status != 200) {
                throw new Error("Failed to fetch project");
            }

            const project = await resp.json();
            if (project.type != "playground") {
                throw new Error("Project is not a playground");
            }

            const snippet = project.playground.snippet;
            const lang = project.playground.mediaType
            return {
                markdown: `\`\`\`${lang}\n${snippet}\n\`\`\``,
                actions: [
                    {
                        title: "Copy Snippet",
                        key: "c",
                        type: "copy",
                        text: snippet,
                        exit: true
                    },
                    {
                        title: "Open in Browser",
                        key: "o",
                        type: "open",
                        url: `https://dash.deno.com/playground/${project.id}`,
                        exit: true
                    }
                ],
            } as sunbeam.Detail;
        }
        case "deployments": {
            const project = payload.params.project as string;

            const resp = await fetchDeployAPI(`/projects/${project}/deployments`);
            if (resp.status != 200) {
                throw new Error("Failed to fetch deployments");
            }

            const [deployments] = await resp.json();
            return {
                items: deployments.map(({ id, createdAt, deployment, relatedCommit }: any) => {
                    const item = {
                        title: id,
                        accessories: [dates.formatDistance(new Date(createdAt), new Date(), {
                            addSuffix: true,
                        })],
                        actions: [],
                    } as sunbeam.ListItem;

                    if (deployment.domainMappings?.length) {
                        item.actions?.push({
                            title: "Open URL",
                            type: "open",
                            url: `https://${deployment.domainMappings[0].domain}`,
                            exit: true,
                        })
                    }

                    if (relatedCommit) {
                        item.title = relatedCommit.message;
                        item.actions?.push({
                            title: "Open Commit",
                            type: "open",
                            url: relatedCommit.url,
                            exit: true,
                        })
                    }

                    return item;
                })
            } as sunbeam.List;
        }
        case "logs": {
            const { project, deployment } = payload.params as { project: string, deployment: string };
            const resp = await fetchDeployAPI(`/projects/${project}/deployments/${deployment}`);
            if (resp.status != 200) {
                throw new Error("Failed to fetch deployment");
            }


        }
    }
}

function fetchDeployAPI(endpoint: string, init?: RequestInit) {
    return fetch(`https://dash.deno.com/api${endpoint}`, {
        ...init,
        headers: {
            ...init?.headers,
            "Authorization": `Bearer ${deployToken}`,
        }
    })
}
