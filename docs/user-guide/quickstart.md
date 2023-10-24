# Using Sunbeam

## Installing Extensions

A fresh install of sunbeam is quite boring. In order to make it useful, you need to add some extensions.

You can manage your extensions using the `sunbeam extension` command.

To get a list of available extensions, run `sunbeam extension browse`.
It will open the [extension registry](https://github.com/topics/sunbeam-extension) in your browser.

Once you find an extension you like, you can install it using `sunbeam extension install <url>`.

> ⚠️ Extensions are not verified, nor sandboxed. They can do anything you can do on your computer. Make sure you trust the source before installing an extension.

For example, to install the [devdocs extension](https://github.com/pomdtr/sunbeam-devdocs), run:

```sh
sunbeam extension install github:pomdtr/sunbeam-devdocs
```

Other managements commands are available:

- `sunbeam extension list` -> list installed extensions
- `sunbeam extension remove` -> uninstall an extension
- `sunbeam extension rename` -> rename an extension
- `sunbeam extension upgrade` -> upgrade an extension

## Running Commands

If you run `sunbeam` without any arguments, it will open the default view, which is a list of all the available commands.

You can also run a command directly using `sunbeam <extension> [command]`.\
For example, `sunbeam devdocs list-docsets` will open the devdocs extension.

You can also pass parameters to the command using `sunbeam <extension> [command] --param1 value1 --param2 value2`. \
For example, `sunbeam devdocs list-entries --docset go` will list all the entries in the go docset.


## Using the Sunbeam UI

Sunbeam is designed to be used with your keyboard. Depending on the current view, multiple keyboard shortcuts are available:

- all views:
    - `ctrl+r` -> refresh the current view
    - `ctrl+c` -> exit sunbeam
    - `escape` -> go back to the previous page
- list view:
    - `up` / `ctrl+k` -> move selection up
    - `down` / `ctrl+j` -> move selection down
    - `enter` -> execute the selected command
    - `tab` -> show the available actions for the selected item
- detail view:
    - `up`, `k` -> scroll one line up
    - `down`, `j` -> scroll one line down
    - `ctrl+u` -> scroll half a page up
    - `ctrl+d` -> scroll half a page down
    - `q` -> exit sunbeam
    - `tab` -> show the available actions