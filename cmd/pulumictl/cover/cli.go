package cover

import (
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	command := &cobra.Command{
		Use:   "cover",
		Short: "Manipulate coverage profiles",
		Long:  `Manipulate coverage profiles`,
	}

	command.AddCommand(coverCommand())

	return command
}
