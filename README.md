# Tortuga [![Build Status](https://travis-ci.org/benweidig/tortuga.svg?branch=master)](https://travis-ci.org/benweidig/tortuga)

CLI tool for fetching/pushing/rebasing multiple git repositories at once.

![Tortuga Mascot](mascot.png)  
[Based on Gopherize.me](https://gopherize.me/gopher/79e06dc4b7a8669c8aa0d6381af7f02f5474e3b7)  
[Git Logo by Jason Long under CC BY 3.0](https://git-scm.com/downloads/logos)

## Usage
```
tortuga [-l/--local-only] [-m/--monochrome] [-y/--yes] [<path>]
```

## Arguments

| Argument          | Default | Description                        |
| ----------------- | ------- | ---------------------------------- |
| -l / --local-only | false   | Don't update remotes               |
| -m / --monochrome | false   | Don't use ANSI colors              |
| -y / --yes        | false   | Automatically _yes_ any question   |
| path              | .       | Path containing your repositories  |

ANSI colors might be disabled automatically if the terminal doesn't seem to support it, but the detection is not perfect.

## License

MIT. See [LICENSE](LICENSE).
