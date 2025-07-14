package cmd

import (
	"fmt"

	"github.com/ionut-t/perp/internal/constants"
	"github.com/ionut-t/perp/internal/version"
	"github.com/ionut-t/perp/ui/styles"
)

func versionTemplate() string {
	versionTpl := styles.Primary.Margin(0, 2).Render(constants.Logo) + `
  Version        %s
  Commit         %s
  Release date   %s
`
	return fmt.Sprintf(versionTpl, version.Version(), version.Commit(), version.Date())
}
