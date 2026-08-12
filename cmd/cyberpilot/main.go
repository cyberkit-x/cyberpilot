package main

import (
	"fmt"
	"os"

	"github.com/cyberkit-x/cyberpilot/internal/buildinfo"
)

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "version" || os.Args[1] == "--version") {
		fmt.Println(buildinfo.String())
		return
	}

	fmt.Fprintln(os.Stderr, "CyberPilot is in its bootstrap stage. Run 'cyberpilot version' for build information.")
}
