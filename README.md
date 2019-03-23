# Tortuga [![Build Status](https://travis-ci.org/benweidig/tortuga.svg?branch=master)](https://travis-ci.org/benweidig/tortuga)

CLI tool for fetching/pushing/rebasing multiple git repositories at once.

![Tortuga Mascot](mascot.png)  
[Based on Gopherize.me](https://gopherize.me/gopher/79e06dc4b7a8669c8aa0d6381af7f02f5474e3b7)  
[Git Logo by Jason Long under CC BY 3.0](https://git-scm.com/downloads/logos)

## Requirements

The tool won't ask for your git credentials because it checks multiple repositories at once async. You should have the credentials available via git-cerdentials-helper/-cache or it will display _Error_ for repositories it can't authenticate with.

## Install

You can either build from source, use the .deb-files, or on macOS just use homebrew with `brew install benweidig/homebrew-tap/tortuga`.

## Usage
```
tt [-m/--monochrome] [-y/--yes] [-v/--verbose] [<path>]
```

## Arguments

| Argument          | Default | Description                        |
| ----------------- | ------- | ---------------------------------- |
| -m / --monochrome | false   | Don't use ANSI colors              |
| -y / --yes        | false   | Automatically _yes_ any question   |
| -v / --verbose    | false   | Verbose error output               |
| path              | .       | Path containing your repositories  |

ANSI colors might be disabled automatically if the terminal doesn't seem to support it, but the detection is not perfect.
The environment variable [`NO_COLOR`](http://no-color.org/) is also checked.

## License

MIT. See [LICENSE](LICENSE).

Parts of the library (package ui / StdoutWriter) are _inspired_ by [gosuri/uilive](https://github.com/gosuri/uilive) which itself was released under MIT [license](https://github.com/gosuri/uilive/blob/master/LICENSE).
