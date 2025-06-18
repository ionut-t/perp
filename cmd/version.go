package cmd

import (
	"fmt"

	"github.com/ionut-t/perp/ui/styles"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

const logo = `
 ____  ____  ____  ____ 
(  _ \(  __)(  _ \(  _ \
 ) __/ ) _)  )   / ) __/
(__)  (____)(__\_)(__)  
`

func versionTemplate() string {
	versionTpl := styles.Primary.Margin(0, 2).Render(logo) + `
  Version        %s
  Commit         %s
  Release date   %s
`
	return fmt.Sprintf(versionTpl, version, commit, date)
}
