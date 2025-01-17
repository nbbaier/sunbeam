# Command

## Copy

Copy text to the clipboard.

```json
{
    // the title of the action (required)
    "title": "Copy",
    // the key to trigger the action (optional)
    "key": "c",
    // the type of the action (required)
    "type": "copy",
    // the text to copy (required)
    "text": "hello world",
    // whether to exit sunbeam after copying the text (optional)
    // if not specified, sunbeam will not exit
    "exit": true
}
```

## Open

Open a url or a file with the default app or a specific app.

```json
{

    // the title of the action (required)
    "title": "Open",
    // the key to trigger the action (optional)
    "key": "o",
    // the title of the action (required)
    "type": "open",
    // the target to open (required)
    // target can be a url or a path to a file
    "url": "https://pomdtr.github.io/sunbeam",
}
```

## Run

Run a custom command defined in the extension manifest.

```json
{
    // the title of the action (required)
    "tit // key must match the name of the param of the edit-readme commandle": "View Readme",
    // the key to trigger the action (optional)
    "key": "v",
    // the type of the action (required)
    "type": "run",
    // the command to run (must be defined in the extension manifest) (required)
    "command": "edit-readme",
    // the arguments to pass to the command (optional)
    // you can either pass a string, number or boolean
    "params": {
        // key must match the name of the param of the edit-readme command
        "full_name": "pomdtr/sunbeam"
    },
}
```

## Reload

Reload the current view.

```json
{
    // the title of the action (required)
    "title": "Reload",
    // the key to trigger the action (optional)
    "key": "r",
    // the type of the action (required)
    "type": "reload",
    // override the current params (optional)
    "params": {
        "full_name": "pomdtr/sunbeam" // key must match the name of the param
    }
}
```

## Exit

Exit sunbeam.

```json
{
    // the title of the action (required)
    "title": "Exit",
    // the key to trigger the action (optional)
    "key": "q",
    // the type of the action (required)
    "type": "exit"
}
```
