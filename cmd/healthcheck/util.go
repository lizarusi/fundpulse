package main

import (
	"io"
	"os"
)

func stderr() io.Writer { return os.Stderr }
