# Tortuga [![Build Status](https://travis-ci.org/benweidig/tortuga.svg?branch=master)](https://travis-ci.org/benweidig/tortuga)

CLI tool for fetching/pushing/rebasing multiple git repositories at once.

## Work in Progress

THIS TOOL IS A WORK IN PROGRESS!
NOT BATTLE-TESTED AT ALL!
TREAD WITH CARE!


## Usage
```
tortuga [-l/--local-only] [<path>]
```

## Arguments

| Argument          | Default | Description                        |
| ----------------- | ------- | ---------------------------------- |
| -l / --local-only | false   | Don't update remotes               |
| -m / --monochrome | false   | Don't use ANSI colors              |
| <path>            | .       | Path containing your repositories  |

ANSI colors might be disabled automatically if the terminal doesn't seem to support it, but the detection is not perfect.

## License

MIT. See [LICENSE](LICENSE).
